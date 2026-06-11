package service

import (
	"testing"
	"time"

	"github.com/ludandaye/hy2board/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestTrafficDeltasUseCounterDeltasNotRawSnapshotSums(t *testing.T) {
	db := setupTrafficDeltaDB(t)
	start := time.Date(2026, 5, 3, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	rows := []model.TrafficLog{
		{SampledAt: start.Add(1 * time.Hour), NodeID: 1, Username: "alice", TX: 1000, RX: 2000},
		{SampledAt: start.Add(2 * time.Hour), NodeID: 1, Username: "alice", TX: 1300, RX: 2600},
		{SampledAt: start.Add(3 * time.Hour), NodeID: 1, Username: "alice", TX: 1500, RX: 3000},
		{SampledAt: start.Add(1 * time.Hour), NodeID: 2, Username: "alice", TX: 500, RX: 700},
		{SampledAt: start.Add(2 * time.Hour), NodeID: 2, Username: "alice", TX: 800, RX: 900},
		{SampledAt: start.Add(1 * time.Hour), NodeID: 1, Username: "bob", TX: 9000, RX: 9000},
		{SampledAt: start.Add(2 * time.Hour), NodeID: 1, Username: "bob", TX: 9100, RX: 9200},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("insert logs: %v", err)
	}

	got, err := TrafficDeltas(db, start, end, 10)
	if err != nil {
		t.Fatalf("traffic deltas: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 users, got %#v", got)
	}
	if got[0].Username != "alice" || got[0].TX != 800 || got[0].RX != 1200 || got[0].Total != 2000 {
		t.Fatalf("expected alice delta 2000 first, got %#v", got[0])
	}
	if got[1].Username != "bob" || got[1].TX != 100 || got[1].RX != 200 || got[1].Total != 300 {
		t.Fatalf("expected bob delta 300 second, got %#v", got[1])
	}
}

func TestTrafficDeltasHandleCounterReset(t *testing.T) {
	db := setupTrafficDeltaDB(t)
	start := time.Date(2026, 5, 3, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	rows := []model.TrafficLog{
		{SampledAt: start.Add(1 * time.Hour), NodeID: 1, Username: "alice", TX: 1000, RX: 1000},
		{SampledAt: start.Add(2 * time.Hour), NodeID: 1, Username: "alice", TX: 1300, RX: 1600},
		{SampledAt: start.Add(3 * time.Hour), NodeID: 1, Username: "alice", TX: 50, RX: 80},
		{SampledAt: start.Add(4 * time.Hour), NodeID: 1, Username: "alice", TX: 120, RX: 200},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("insert logs: %v", err)
	}

	got, err := TrafficDeltas(db, start, end, 10)
	if err != nil {
		t.Fatalf("traffic deltas: %v", err)
	}

	if len(got) != 1 || got[0].TX != 420 || got[0].RX != 800 || got[0].Total != 1220 {
		t.Fatalf("expected reset-aware delta 1220, got %#v", got)
	}
}

func setupTrafficDeltaDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.TrafficLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}
