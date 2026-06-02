package rssfeed

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestParseFeedSources(t *testing.T) {
	sources := parseFeedSources([]string{" https://a.example/rss | Alpha ", "", "https://b.example/rss"})
	if len(sources) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(sources))
	}
	if sources[0].URL != "https://a.example/rss" || sources[0].Label != "Alpha" {
		t.Fatalf("unexpected first source: %+v", sources[0])
	}
	if sources[1].URL != "https://b.example/rss" || sources[1].Label != "" {
		t.Fatalf("unexpected second source: %+v", sources[1])
	}
}

func TestConfigureAndClientTracking(t *testing.T) {
	m := New(nil)

	m.Configure("block-1", 10, []string{"https://a.example/rss"}, 1, 0)
	cfg, ok := m.configs["block-1"]
	if !ok {
		t.Fatal("expected block config to exist")
	}
	if cfg.RefreshInterval != 10 {
		t.Fatalf("refresh interval should be clamped to 10, got %d", cfg.RefreshInterval)
	}
	if cfg.MaxEntries != 20 {
		t.Fatalf("max entries should default to 20, got %d", cfg.MaxEntries)
	}
	if _, ok := m.seenPerClient["block-1"][10]; !ok {
		t.Fatal("expected initial client seen map")
	}

	m.RegisterClient(11)
	if _, ok := m.seenPerClient["block-1"][11]; !ok {
		t.Fatal("expected newly registered client on existing block")
	}

	m.UnregisterClient(10)
	if _, ok := m.seenPerClient["block-1"][10]; ok {
		t.Fatal("expected client to be removed")
	}

	m.Remove("block-1")
	if m.configs["block-1"].BlockID != "" {
		t.Fatal("expected removed block config")
	}
}

func TestFetchFeedDedupAndIsNewLifecycle(t *testing.T) {
	now := time.Now().UTC()
	newer := now.Format(time.RFC1123Z)
	older := now.Add(-time.Hour).Format(time.RFC1123Z)

	xml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"><channel><title>Demo</title>
<item><guid>a</guid><title>A</title><link>https://example.com/a</link><pubDate>%s</pubDate><description>first</description></item>
<item><guid>b</guid><title>B</title><link>https://example.com/b</link><pubDate>%s</pubDate><description>second</description></item>
<item><guid>a</guid><title>A-DUP</title><link>https://example.com/a</link><pubDate>%s</pubDate></item>
</channel></rss>`, newer, older, older)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(xml))
	}))
	defer ts.Close()

	type call struct {
		clientID uint64
		entries  []EntryWithNew
	}
	var calls []call
	m := New(func(clientID uint64, blockID string, entries []EntryWithNew) {
		if blockID != "b1" {
			t.Fatalf("unexpected block id: %s", blockID)
		}
		copied := make([]EntryWithNew, len(entries))
		copy(copied, entries)
		calls = append(calls, call{clientID: clientID, entries: copied})
	})
	defer m.Close()

	m.configs["b1"] = FeedConfig{
		BlockID:         "b1",
		FeedSources:     []FeedSource{{URL: ts.URL, Label: "Label"}},
		RefreshInterval: 3600,
		MaxEntries:      10,
	}
	m.seenPerClient["b1"] = map[uint64]map[string]bool{
		1: {},
		2: {},
	}

	m.fetchFeed("b1")
	if len(m.entries["b1"]) != 2 {
		t.Fatalf("expected deduped 2 entries, got %d", len(m.entries["b1"]))
	}

	if len(calls) != 2 {
		t.Fatalf("expected 2 broadcasts, got %d", len(calls))
	}
	for _, c := range calls {
		if !c.entries[0].IsNew || !c.entries[1].IsNew {
			t.Fatalf("first fetch should mark entries new for client %d: %+v", c.clientID, c.entries)
		}
	}

	calls = nil
	m.fetchFeed("b1")
	if len(calls) != 2 {
		t.Fatalf("expected 2 broadcasts on second fetch, got %d", len(calls))
	}
	for _, c := range calls {
		if !c.entries[0].IsNew {
			t.Fatalf("newest entry should remain new for client %d", c.clientID)
		}
		if c.entries[1].IsNew {
			t.Fatalf("older entry should be marked seen for client %d", c.clientID)
		}
	}
}

func TestGetEntriesStartCloseAndScheduleStop(t *testing.T) {
	m := New(nil)
	m.Start()

	m.entries["x"] = []FeedEntry{{GUID: "1"}}
	got := m.GetEntries("x")
	if len(got) != 1 || got[0].GUID != "1" {
		t.Fatalf("unexpected entries: %+v", got)
	}

	m.configs["x"] = FeedConfig{BlockID: "x", RefreshInterval: 10}
	done := make(chan struct{})
	go func() {
		m.scheduleNextFetch("x")
		close(done)
	}()
	m.Close()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("scheduleNextFetch did not stop after close")
	}
}

func TestConfigureIgnoresInvalidSources(t *testing.T) {
	m := New(nil)
	m.Configure("bad", 1, []string{"   ", "|label-only"}, 30, 5)
	if _, ok := m.configs["bad"]; ok {
		t.Fatal("invalid sources must not create config")
	}
}

func TestFeedEntriesSortedDescending(t *testing.T) {
	now := time.Now().UTC()
	xml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"><channel><title>Demo</title>
<item><guid>old</guid><title>old</title><pubDate>%s</pubDate></item>
<item><guid>new</guid><title>new</title><pubDate>%s</pubDate></item>
</channel></rss>`, now.Add(-time.Hour).Format(time.RFC1123Z), now.Format(time.RFC1123Z))
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(xml))
	}))
	defer ts.Close()

	m := New(nil)
	defer m.Close()
	m.configs["s"] = FeedConfig{BlockID: "s", FeedSources: []FeedSource{{URL: ts.URL}}, RefreshInterval: 3600, MaxEntries: 10}
	m.seenPerClient["s"] = map[uint64]map[string]bool{1: {}}
	m.fetchFeed("s")

	guids := []string{m.entries["s"][0].GUID, m.entries["s"][1].GUID}
	if guids[0] != "new" || guids[1] != "old" {
		t.Fatalf("expected newest first, got %v", guids)
	}
}

func TestFetchAndScheduleOnMissingBlock(t *testing.T) {
	m := New(nil)
	defer m.Close()

	// Must be a no-op if block config is missing.
	m.fetchFeed("missing")

	done := make(chan struct{})
	go func() {
		m.scheduleNextFetch("missing")
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("scheduleNextFetch should return immediately for missing config")
	}
}

func TestFetchFeedHandlesParseErrorsAndFallbackLabel(t *testing.T) {
	t.Run("parse error keeps previous entries", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("not xml"))
		}))
		defer ts.Close()

		m := New(nil)
		defer m.Close()
		m.configs["b"] = FeedConfig{BlockID: "b", FeedSources: []FeedSource{{URL: ts.URL}}, RefreshInterval: 3600, MaxEntries: 5}
		m.seenPerClient["b"] = map[uint64]map[string]bool{1: {}}

		m.fetchFeed("b")
		if len(m.entries["b"]) != 0 {
			t.Fatalf("entries should be empty when all sources fail, got %+v", m.entries["b"])
		}
	})

	t.Run("feed title used when label empty", func(t *testing.T) {
		xml := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"><channel><title>FeedTitle</title>
<item><guid>x</guid><title>X</title><pubDate>Mon, 02 Jan 2006 15:04:05 -0700</pubDate></item>
</channel></rss>`
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(xml))
		}))
		defer ts.Close()

		m := New(nil)
		defer m.Close()
		m.configs["b"] = FeedConfig{BlockID: "b", FeedSources: []FeedSource{{URL: ts.URL, Label: ""}}, RefreshInterval: 3600, MaxEntries: 5}
		m.seenPerClient["b"] = map[uint64]map[string]bool{1: {}}

		m.fetchFeed("b")
		if len(m.entries["b"]) != 1 || m.entries["b"][0].FeedLabel != "FeedTitle" {
			t.Fatalf("expected fallback feed title label, got %+v", m.entries["b"])
		}
	})
}

