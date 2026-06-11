package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ludandaye/hy2board/internal/model"
	"gorm.io/gorm"
)

var auditSecretKeys = map[string]bool{
	"password":       true,
	"hy2_password":   true,
	"sub_token":      true,
	"traffic_secret": true,
	"obfs_password":  true,
	"proxy_password": true,
	"login_password": true,
	"private_key":    true,
	"preshared_key":  true,
	"authorization":  true,
}

func RecordAudit(db *gorm.DB, log model.AuditLog) error {
	if db == nil {
		return nil
	}
	if strings.TrimSpace(log.Actor) == "" {
		log.Actor = "unknown"
	}
	return db.Create(&log).Error
}

func AuditDetail(v interface{}) string {
	sanitized := sanitizeAuditValue(v)
	b, err := json.Marshal(sanitized)
	if err != nil {
		return fmt.Sprintf("%v", sanitized)
	}
	return string(b)
}

func sanitizeAuditValue(v interface{}) interface{} {
	switch x := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(x))
		for k, val := range x {
			if auditSecretKeys[strings.ToLower(k)] {
				out[k] = "***"
				continue
			}
			out[k] = sanitizeAuditValue(val)
		}
		return out
	case map[string]string:
		out := make(map[string]interface{}, len(x))
		for k, val := range x {
			if auditSecretKeys[strings.ToLower(k)] {
				out[k] = "***"
				continue
			}
			out[k] = val
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(x))
		for i, val := range x {
			out[i] = sanitizeAuditValue(val)
		}
		return out
	default:
		return v
	}
}
