package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	NodeProbeStatusOK      = "ok"
	NodeProbeStatusWarning = "warning"
	NodeProbeStatusFailed  = "failed"
)

type NodeProbeResult struct {
	gorm.Model
	NodeID    uint      `gorm:"index" json:"node_id"`
	Target    string    `gorm:"index" json:"target"`
	Status    string    `gorm:"index" json:"status"`
	LatencyMS int       `json:"latency_ms"`
	Error     string    `json:"error"`
	CheckedAt time.Time `gorm:"index" json:"checked_at"`
}

type NodeProbeState struct {
	gorm.Model
	NodeID        uint       `gorm:"uniqueIndex" json:"node_id"`
	Status        string     `gorm:"default:'warning';index" json:"status"`
	LastOKAt      *time.Time `json:"last_ok_at"`
	LastFailedAt  *time.Time `json:"last_failed_at"`
	LastCheckedAt *time.Time `json:"last_checked_at"`
	LastLatencyMS int        `json:"last_latency_ms"`
	LastError     string     `json:"last_error"`
	FailStreak    int        `gorm:"default:0" json:"fail_streak"`
	LastAlertAt   *time.Time `json:"last_alert_at"`
	AlertedDown   bool       `gorm:"default:false" json:"alerted_down"`
}
