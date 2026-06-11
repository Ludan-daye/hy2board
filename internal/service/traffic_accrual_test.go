package service

import (
	"testing"

	"github.com/ludandaye/hy2board/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// node/user cumulative snapshot helper
func snap(m map[uint]map[string]TrafficData) map[uint]map[string]TrafficData { return m }

func TestTrafficUsageDeltaAccumulatesAcrossNodes(t *testing.T) {
	prev := snap(map[uint]map[string]TrafficData{
		1: {"a": {TX: 100, RX: 50}},
		2: {"a": {TX: 0, RX: 0}},
	})
	now := snap(map[uint]map[string]TrafficData{
		1: {"a": {TX: 200, RX: 100}},
		2: {"a": {TX: 10, RX: 5}},
	})
	got := trafficUsageDelta(prev, now)
	if got["a"] != 165 {
		t.Fatalf("expected a=165 (150+15), got %d", got["a"])
	}
}

func TestTrafficUsageDeltaHandlesPerNodeCounterReset(t *testing.T) {
	// node 1 restarted (cur < prev) -> counts cur; node 2 keeps climbing.
	prev := snap(map[uint]map[string]TrafficData{
		1: {"a": {TX: 1000, RX: 500}},
		2: {"a": {TX: 2000, RX: 1000}},
	})
	now := snap(map[uint]map[string]TrafficData{
		1: {"a": {TX: 300, RX: 100}},
		2: {"a": {TX: 2500, RX: 1200}},
	})
	got := trafficUsageDelta(prev, now)
	if got["a"] != 1100 { // (300+100) + (500+200)
		t.Fatalf("expected a=1100, got %d", got["a"])
	}
}

func TestTrafficUsageDeltaFirstSnapshotAddsNothing(t *testing.T) {
	// Empty prev = deploy moment. Must NOT add cumulative counters retroactively,
	// or heavy users would be blocked instantly. Baseline only.
	prev := map[uint]map[string]TrafficData{}
	now := snap(map[uint]map[string]TrafficData{
		1: {"heavy": {TX: 80_000_000_000, RX: 40_000_000_000}},
	})
	got := trafficUsageDelta(prev, now)
	if got["heavy"] != 0 {
		t.Fatalf("expected heavy=0 on first snapshot, got %d", got["heavy"])
	}
}

func TestTrafficUsageDeltaNewUserOnExistingNodeBaselinesOnly(t *testing.T) {
	prev := snap(map[uint]map[string]TrafficData{
		1: {"a": {TX: 100, RX: 0}},
	})
	now := snap(map[uint]map[string]TrafficData{
		1: {"a": {TX: 150, RX: 0}, "b": {TX: 500, RX: 0}},
	})
	got := trafficUsageDelta(prev, now)
	if got["a"] != 50 {
		t.Fatalf("expected a=50, got %d", got["a"])
	}
	if got["b"] != 0 {
		t.Fatalf("expected b=0 (first sighting baseline), got %d", got["b"])
	}
}

func TestPersistTrafficUsageIncrementsAndEnforcesLimit(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	u := model.User{Username: "wdz", Hy2Password: "wdz:x", SubToken: "tok", Enabled: true, TrafficLimit: 1000, TrafficUsed: 0}
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if u.IsActive() != true || u.TrafficExceeded() {
		t.Fatalf("precondition: user should be active and under limit")
	}

	persistTrafficUsage(db, map[string]int64{"wdz": 1500})

	var got model.User
	db.First(&got, u.ID)
	if got.TrafficUsed != 1500 {
		t.Fatalf("expected traffic_used=1500, got %d", got.TrafficUsed)
	}
	if !got.TrafficExceeded() || got.IsActive() {
		t.Fatalf("expected over-limit user to be inactive (enforcement), got active")
	}

	// increment semantics (adds, not overwrites)
	persistTrafficUsage(db, map[string]int64{"wdz": 100})
	db.First(&got, u.ID)
	if got.TrafficUsed != 1600 {
		t.Fatalf("expected traffic_used=1600 after second accrual, got %d", got.TrafficUsed)
	}
}
