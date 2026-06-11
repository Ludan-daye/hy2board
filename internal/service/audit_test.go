package service

import (
	"strings"
	"testing"

	"github.com/ludandaye/hy2board/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRecordAuditCreatesLogWithSanitizedDetail(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.AuditLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	err = RecordAudit(db, model.AuditLog{
		Actor:      "admin",
		Action:     "user.update",
		Entity:     "user",
		EntityID:   7,
		EntityName: "alice",
		Detail:     AuditDetail(map[string]interface{}{"password": "secret", "field": "email"}),
		IP:         "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("record audit: %v", err)
	}

	var got model.AuditLog
	if err := db.First(&got).Error; err != nil {
		t.Fatalf("read audit: %v", err)
	}
	if got.Actor != "admin" || got.Action != "user.update" || got.Entity != "user" || got.EntityID != 7 {
		t.Fatalf("unexpected audit row: %#v", got)
	}
	if strings.Contains(got.Detail, "secret") {
		t.Fatalf("detail leaked secret: %s", got.Detail)
	}
	if !strings.Contains(got.Detail, `"password":"***"`) || !strings.Contains(got.Detail, `"field":"email"`) {
		t.Fatalf("detail was not sanitized as expected: %s", got.Detail)
	}
}
