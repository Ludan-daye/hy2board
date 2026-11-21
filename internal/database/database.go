package database

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ludandaye/hy2board/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func sqliteDSN(path string) string {
	if strings.Contains(path, "?") || strings.HasPrefix(path, "file:") {
		return path
	}
	return path + "?_busy_timeout=5000&_journal_mode=WAL&_synchronous=NORMAL&_txlock=immediate"
}

func Init(dbPath string) error {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	var err error
	DB, err = gorm.Open(sqlite.Open(sqliteDSN(dbPath)), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return err
	}
	if sqlDB, err := DB.DB(); err == nil {
		sqlDB.SetMaxOpenConns(4)
		sqlDB.SetMaxIdleConns(4)
	}

	if err := DB.AutoMigrate(&model.User{}, &model.Node{}, &model.Plan{}, &model.TrafficLog{}, &model.Setting{}, &model.Checkin{}, &model.Payment{}, &model.Cost{}, &model.NodeProbeResult{}, &model.NodeProbeState{}, &model.AuditLog{}, &model.CustomRoutingRule{}); err != nil {
		return err
	}

	// Back-compat: existing chain_proxy=true users → rule_ai=true
	DB.Model(&model.User{}).
		Where("chain_proxy = ? AND rule_ai = ?", true, false).
		Update("rule_ai", true)

	// Initialize LastResetAt for users that don't have it yet (older than year 2000).
	DB.Model(&model.User{}).
		Where("last_reset_at IS NULL OR last_reset_at < ?", time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)).
		UpdateColumn("last_reset_at", gorm.Expr("created_at"))

	// Seed default Plans if table empty.
	var planCount int64
	DB.Model(&model.Plan{}).Count(&planCount)
	if planCount == 0 {
		const GB = int64(1024 * 1024 * 1024)
		DB.Create(&[]model.Plan{
			{Name: "Trial", TrafficLimit: 5 * GB, DurationDays: 7, RuleChina: true},
			{Name: "Basic", TrafficLimit: 30 * GB, DurationDays: 30, RuleChina: true, AutoReset: true},
			{Name: "Pro", TrafficLimit: 100 * GB, DurationDays: 30, RuleChina: true, RuleAI: true, RuleStreaming: true, AutoReset: true},
			{Name: "Unlimited", TrafficLimit: 0, DurationDays: 365, RuleChina: true, RuleAI: true, RuleStreaming: true, RuleAdBlock: true},
		})
	}

	return nil
}
