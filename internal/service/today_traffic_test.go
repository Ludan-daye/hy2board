package service

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/ludandaye/hy2board/internal/model"
)

func TestLoadTodayTrafficSumsPerNodeDeltasInsideWindow(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:today-traffic?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&model.TrafficLog{}); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 11, 21, 8, 0, 0, 0, time.UTC)
	rows := []model.TrafficLog{
		{SampledAt: now.Add(-48 * time.Hour), NodeID: 1, Username: "a", TX: 10, RX: 10},
		{SampledAt: now.Add(-2 * time.Hour), NodeID: 1, Username: "a", TX: 100, RX: 50},
		{SampledAt: now.Add(-time.Hour), NodeID: 1, Username: "a", TX: 180, RX: 80},
		{SampledAt: now.Add(-2 * time.Hour), NodeID: 2, Username: "a", TX: 5, RX: 5},
		{SampledAt: now.Add(-time.Hour), NodeID: 2, Username: "a", TX: 9, RX: 7},
		{SampledAt: now.Add(-48 * time.Hour), NodeID: 1, Username: "b", TX: 1000, RX: 1000},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}

	got, err := loadTodayTraffic(db, now.Add(-24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if got["b"].Total() != 0 {
		t.Fatalf("old user b was included: %+v", got["b"])
	}
	want := TodayTraffic{TX: 84, RX: 32}
	if got["a"] != want {
		t.Fatalf("user a = %+v, want %+v", got["a"], want)
	}
}
