package databus

import (
	"testing"
	"time"
)

func TestNew_InitializesMaps(t *testing.T) {
	db := New()
	if db == nil {
		t.Fatalf("New() returned nil")
	}
	if db.data == nil || db.lastNetRx == nil || db.lastNetTx == nil {
		t.Fatalf("New() did not initialize internal maps")
	}
}

func TestSetGetAndSetSource(t *testing.T) {
	db := New()
	db.Set("k1", 42, "%")
	v, ok := db.Get("k1")
	if !ok {
		t.Fatalf("Get(k1) not found")
	}
	if v.Value != 42 || v.Unit != "%" || v.Source != "api" {
		t.Fatalf("unexpected value from Set/Get: %#v", v)
	}

	db.SetSource("k2", "abc", "u", "mpris")
	v2, ok := db.Get("k2")
	if !ok {
		t.Fatalf("Get(k2) not found")
	}
	if v2.Value != "abc" || v2.Unit != "u" || v2.Source != "mpris" {
		t.Fatalf("unexpected value from SetSource/Get: %#v", v2)
	}
}

func TestSnapshot_ReturnsDefensiveCopy(t *testing.T) {
	db := New()
	db.Set("key", 1, "")
	snap := db.Snapshot()
	snap["key"] = DataValue{Value: 999, Unit: "", Source: "tampered"}

	v, ok := db.Get("key")
	if !ok {
		t.Fatalf("Get(key) not found")
	}
	if v.Value != 1 {
		t.Fatalf("Snapshot should be defensive copy, got %#v", v)
	}
}

func TestStartMetrics_DoesNotBreakConcurrentAccess(t *testing.T) {
	db := New()
	db.StartMetrics(10 * time.Millisecond)

	deadline := time.Now().Add(80 * time.Millisecond)
	for time.Now().Before(deadline) {
		db.Set("tick", time.Now().UnixNano(), "")
		_, _ = db.Get("tick")
		_ = db.Snapshot()
		time.Sleep(5 * time.Millisecond)
	}
}


