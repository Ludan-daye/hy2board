package service

import (
	"testing"

	"github.com/ludandaye/hy2board/internal/config"
	"github.com/ludandaye/hy2board/internal/database"
	"github.com/ludandaye/hy2board/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGetTelegramStatusReportsConfiguredGroupAndAdminBinding(t *testing.T) {
	db := setupTelegramStatusTestDB(t)
	db.Create(&model.Setting{Key: keyGroupChatID, Value: "-1003517236471"})
	db.Create(&model.User{Username: "admin", Hy2Password: "secret", SubToken: "token", TelegramID: 5986791808})
	config.C = config.Config{
		Admin:    config.AdminConfig{Username: "admin"},
		Telegram: config.TelegramConfig{Enabled: true, BotToken: "telegram-token"},
	}

	got := GetTelegramStatus()

	if !got.Enabled || !got.BotConfigured {
		t.Fatalf("expected enabled configured bot, got %#v", got)
	}
	if !got.GroupRegistered || got.GroupChatID != -1003517236471 {
		t.Fatalf("expected registered group id, got %#v", got)
	}
	if !got.AdminBound || got.AdminUsername != "admin" || got.AdminTelegramID != 5986791808 {
		t.Fatalf("expected bound admin, got %#v", got)
	}
	if got.DailyPostTime == "" || got.WeeklyLeaderboardTime == "" {
		t.Fatalf("expected schedule labels, got %#v", got)
	}
}

func TestGetTelegramStatusReportsMissingAdminBinding(t *testing.T) {
	setupTelegramStatusTestDB(t)
	config.C = config.Config{
		Admin:    config.AdminConfig{Username: "admin"},
		Telegram: config.TelegramConfig{Enabled: true},
	}

	got := GetTelegramStatus()

	if got.BotConfigured {
		t.Fatalf("expected bot token to be missing, got %#v", got)
	}
	if got.AdminBound || got.AdminTelegramID != 0 {
		t.Fatalf("expected missing admin binding, got %#v", got)
	}
}

func setupTelegramStatusTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	oldDB := database.DB
	oldConfig := config.C
	t.Cleanup(func() {
		database.DB = oldDB
		config.C = oldConfig
	})

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Setting{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	database.DB = db
	return db
}
