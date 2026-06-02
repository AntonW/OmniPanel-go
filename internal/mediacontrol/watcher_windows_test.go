//go:build windows

package mediacontrol

import (
	"encoding/json"
	"log/slog"
	"testing"

	"omnipanel-go/internal/config"
	"omnipanel-go/internal/databus"
)

func TestAppIDToName(t *testing.T) {
	cases := map[string]string{
		"Spotify.exe":                          "Spotify",
		"MICROSOFT.ZUNEMUSIC_8wekyb3d8bbwe":   "Groove Music",
		"My.Player.exe":                        "Player",
		"com.vendor.Product_abcdef":            "Product",
		"vlc.exe":                              "VLC",
	}
	for in, want := range cases {
		if got := appIDToName(in); got != want {
			t.Fatalf("appIDToName(%q)=%q, want %q", in, got, want)
		}
	}
}

func TestWatcherNilSafetyAndSelection(t *testing.T) {
	var nilWatcher *Watcher
	if err := nilWatcher.Start(); err != nil {
		t.Fatalf("nil Start should be no-op: %v", err)
	}
	if err := nilWatcher.Close(); err != nil {
		t.Fatalf("nil Close should be no-op: %v", err)
	}
	if nilWatcher.ListPlayers() != nil {
		t.Fatal("nil watcher ListPlayers should return nil")
	}
	if nilWatcher.GetPlayerState("x") != nil {
		t.Fatal("nil watcher GetPlayerState should return nil")
	}
	if nilWatcher.GetSelectedPlayer() != "" {
		t.Fatal("nil watcher selected player should be empty")
	}
	if err := nilWatcher.SetSelectedPlayer("x"); err == nil {
		t.Fatal("nil watcher SetSelectedPlayer should fail")
	}
	if err := nilWatcher.CallMethod("x", "Play"); err == nil {
		t.Fatal("nil watcher CallMethod should fail")
	}
	if err := nilWatcher.SetVolume("x", 0.5); err != nil {
		t.Fatalf("SetVolume is no-op on windows watcher, got %v", err)
	}

	w := &Watcher{players: map[string]*PlayerState{"p": {PlayerName: "p"}}, databus: databus.New(), logger: slog.Default()}
	if err := w.SetSelectedPlayer("missing"); err == nil {
		t.Fatal("expected error for unknown player")
	}
	if err := w.SetSelectedPlayer("p"); err != nil {
		t.Fatalf("set selected player failed: %v", err)
	}
	if w.GetSelectedPlayer() != "p" {
		t.Fatalf("unexpected selected player: %q", w.GetSelectedPlayer())
	}
	if st := w.GetPlayerState("p"); st == nil || st.PlayerName != "p" {
		t.Fatalf("unexpected player state: %+v", st)
	}
	if players := w.ListPlayers(); len(players) != 1 || players[0] != "p" {
		t.Fatalf("unexpected player list: %v", players)
	}
}

func TestNewConfigGuard(t *testing.T) {
	if New(nil, databus.New(), slog.Default()) != nil {
		t.Fatal("New should return nil for nil config")
	}
	if New(&config.MediaPlayerConfig{Enabled: false}, databus.New(), slog.Default()) != nil {
		t.Fatal("New should return nil when disabled")
	}
	w := New(&config.MediaPlayerConfig{Enabled: true, PollInterval: 1000}, databus.New(), nil)
	if w == nil {
		t.Fatal("New should create watcher when enabled")
	}
	if w.tempDir == "" {
		t.Fatal("watcher tempDir should be set")
	}
}

func TestPublishHelpers(t *testing.T) {
	db := databus.New()
	w := &Watcher{
		databus:        db,
		players:        map[string]*PlayerState{"p": {Identity: "Player"}},
		selectedPlayer: "p",
	}

	state := &PlayerState{
		PlayerName:     "p",
		Identity:       "Player",
		PlaybackStatus: "Playing",
		Title:          "Song",
		Artist:         "Artist",
		Album:          "Album",
		ArtURL:         "file:///x.jpg",
		Length:         200,
		Position:       100,
		Volume:         0.5,
		CanPlay:        true,
		CanPause:       true,
		CanGoNext:      true,
		CanGoPrevious:  true,
		CanControl:     true,
	}

	w.publishToDataBus(state)
	if got, ok := db.Get("mediacontrol_title"); !ok || got.Value != "Song" {
		t.Fatalf("expected mediacontrol_title=Song, got %#v ok=%v", got, ok)
	}
	if got, ok := db.Get("mediacontrol_progress"); !ok || got.Value.(float64) != 50 {
		t.Fatalf("expected mediacontrol_progress=50, got %#v ok=%v", got, ok)
	}

	// Simulate player stopping (publishing empty state)
	if got, _ := db.Get("mediacontrol_title"); got.Value != "Song" {
	if got, _ := db.Get("mpris_title"); got.Value != "Song" {
		t.Fatalf("non-selected player should not update databus, got %#v", got)
	}

	w.publishPlayersList()
	raw, ok := db.Get("mediacontrol_available_players")
	if !ok {
		t.Fatal("expected mediacontrol_available_players in databus")
	}
	var list []map[string]string
	if err := json.Unmarshal([]byte(raw.Value.(string)), &list); err != nil {
		t.Fatalf("players list is not valid json: %v", err)
	}
	if len(list) != 1 || list[0]["name"] != "p" {
		t.Fatalf("unexpected players list: %v", list)
	}

	w.clearDataBus()
	if got, _ := db.Get("mediacontrol_title"); got.Value != "" {
		t.Fatalf("expected cleared databus field, got %#v", got)
	}
}

func TestCallMethodAndGetSession(t *testing.T) {
	w := &Watcher{
		players:        map[string]*PlayerState{"p": {PlayerName: "p"}},
		selectedPlayer: "p",
	}

	// CallMethod on nil watcher should fail
	var nilW *Watcher
	if err := nilW.CallMethod("x", "Play"); err == nil {
		t.Fatal("nil CallMethod should fail")
	}

	// CallMethod with unknown player should fail
	if err := w.CallMethod("unknown", "Play"); err == nil {
		t.Fatal("unknown player should fail")
	}

	// Out-of-range method should fail (or succeed as no-op)
	if err := w.CallMethod("p", "InvalidMethod"); err == nil {
		t.Fatalf("expected unknown method to fail")
	}
}

func TestPlayerStateZeroValues(t *testing.T) {
	// Test zero PlayerState to ensure proper defaults
	s := &PlayerState{}
	if s.PlayerName != "" || s.Volume != 0.0 {
		t.Fatalf("zero PlayerState should have empty values")
	}
}
