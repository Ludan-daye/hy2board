package handler

import (
	"testing"
	"time"

	"github.com/ludandaye/hy2board/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSyncPlanBoundUsersCopiesSubscriptionFieldsOnly(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.Plan{}, &model.User{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	plan := model.Plan{
		Name:          "Plan A",
		TrafficLimit:  100,
		DurationDays:  30,
		NodeIDs:       "all",
		RuleChina:     true,
		ProxyType:     "socks5",
		ProxyHost:     "old.example.com",
		ProxyPort:     1080,
		ProxyUsername: "old-user",
		ProxyPassword: "old-pass",
	}
	if err := db.Create(&plan).Error; err != nil {
		t.Fatalf("create plan: %v", err)
	}
	otherPlan := model.Plan{Name: "Plan B", RuleChina: true}
	if err := db.Create(&otherPlan).Error; err != nil {
		t.Fatalf("create other plan: %v", err)
	}

	expiresAt := time.Date(2026, 6, 1, 8, 0, 0, 0, time.UTC)
	bound := model.User{
		Username:      "bound",
		Hy2Password:   "bound-password",
		SubToken:      "bound-token",
		TrafficLimit:  10,
		ExpiresAt:     expiresAt,
		Enabled:       true,
		NodeIDs:       "1",
		RuleChina:     true,
		PlanID:        &plan.ID,
		ProxyType:     "http",
		ProxyHost:     "stale.example.com",
		ProxyPort:     8080,
		ProxyUsername: "stale-user",
		ProxyPassword: "stale-pass",
	}
	unbound := model.User{
		Username:     "unbound",
		Hy2Password:  "unbound-password",
		SubToken:     "unbound-token",
		TrafficLimit: 20,
		ExpiresAt:    expiresAt,
		Enabled:      true,
		NodeIDs:      "9",
		RuleChina:    true,
		PlanID:       &otherPlan.ID,
	}
	if err := db.Create(&bound).Error; err != nil {
		t.Fatalf("create bound user: %v", err)
	}
	if err := db.Create(&unbound).Error; err != nil {
		t.Fatalf("create unbound user: %v", err)
	}

	updated := plan
	updated.TrafficLimit = 987654321
	updated.DurationDays = 365
	updated.NodeIDs = "2,3"
	updated.RuleAI = true
	updated.RuleStreaming = true
	updated.RuleChina = false
	updated.RuleAdBlock = true
	updated.AutoReset = true
	updated.ProxyType = "socks5"
	updated.ProxyHost = "fresh.example.com"
	updated.ProxyPort = 2080
	updated.ProxyUsername = "fresh-user"
	updated.ProxyPassword = "fresh-pass"

	affected, err := syncPlanBoundUsers(db, updated)
	if err != nil {
		t.Fatalf("sync users: %v", err)
	}
	if affected != 1 {
		t.Fatalf("affected users = %d, want 1", affected)
	}

	var got model.User
	if err := db.First(&got, bound.ID).Error; err != nil {
		t.Fatalf("reload bound user: %v", err)
	}
	if got.TrafficLimit != updated.TrafficLimit ||
		got.NodeIDs != updated.NodeIDs ||
		got.RuleAI != updated.RuleAI ||
		got.RuleStreaming != updated.RuleStreaming ||
		got.RuleChina != updated.RuleChina ||
		got.RuleAdBlock != updated.RuleAdBlock ||
		got.AutoReset != updated.AutoReset ||
		got.ProxyType != updated.ProxyType ||
		got.ProxyHost != updated.ProxyHost ||
		got.ProxyPort != updated.ProxyPort ||
		got.ProxyUsername != updated.ProxyUsername ||
		got.ProxyPassword != updated.ProxyPassword {
		t.Fatalf("bound user was not synced from plan: got %#v", got)
	}
	if !got.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("expires_at changed to %s, want %s", got.ExpiresAt, expiresAt)
	}

	var untouched model.User
	if err := db.First(&untouched, unbound.ID).Error; err != nil {
		t.Fatalf("reload unbound user: %v", err)
	}
	if untouched.TrafficLimit != unbound.TrafficLimit ||
		untouched.NodeIDs != unbound.NodeIDs ||
		untouched.ProxyHost != unbound.ProxyHost {
		t.Fatalf("unbound user changed: got %#v", untouched)
	}
}
