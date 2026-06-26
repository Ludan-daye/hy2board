package model

import (
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username      string    `gorm:"uniqueIndex;not null" json:"username"`
	LoginPassword string    `gorm:"default:''" json:"-"`
	Hy2Password   string    `gorm:"not null" json:"hy2_password"`
	SubToken      string    `gorm:"uniqueIndex;not null" json:"sub_token"`
	TrafficLimit  int64     `gorm:"default:0" json:"traffic_limit"`
	TrafficUsed   int64     `gorm:"default:0" json:"traffic_used"`
	MaxIPs        int       `gorm:"default:0" json:"max_ips"` // 0 = unlimited; max distinct concurrent source IPs
	ExpiresAt     time.Time `json:"expires_at"`
	Enabled       bool      `gorm:"default:true" json:"enabled"`
	NodeIDs       string    `gorm:"default:'all'" json:"node_ids"`
	ChainProxy    bool      `gorm:"default:false" json:"chain_proxy"`

	// Metadata
	Email         string    `gorm:"default:''" json:"email"`
	Notes         string    `gorm:"default:''" json:"notes"`
	Tags          string    `gorm:"default:''" json:"tags"`

	// Rule toggles (replace chain_proxy semantically; chain_proxy kept for back-compat)
	RuleAI        bool      `gorm:"default:false" json:"rule_ai"`
	RuleStreaming bool      `gorm:"default:false" json:"rule_streaming"`
	RuleChina     bool      `gorm:"default:true"  json:"rule_china"`
	RuleAdBlock   bool      `gorm:"default:false" json:"rule_ad_block"`

	// Auto monthly reset (per-user 30-day cycle)
	AutoReset     bool      `gorm:"default:false" json:"auto_reset"`
	LastResetAt   time.Time `json:"last_reset_at"`

	// Plan attribution (informational, nullable)
	PlanID        *uint     `json:"plan_id"`

	// Telegram integration (0 = unlinked)
	TelegramID    int64     `gorm:"default:0;index" json:"telegram_id"`

	// Static IP override — populated by ApplyPlanToUser when Plan has proxy fields
	ProxyType     string `gorm:"default:''" json:"proxy_type"`
	ProxyHost     string `gorm:"default:''" json:"proxy_host"`
	ProxyPort     int    `gorm:"default:0"  json:"proxy_port"`
	ProxyUsername string `gorm:"default:''" json:"proxy_username"`
	ProxyPassword string `gorm:"default:''" json:"-"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.SubToken == "" {
		b := make([]byte, 16)
		rand.Read(b)
		u.SubToken = hex.EncodeToString(b)
	}
	if u.Hy2Password == "" {
		b := make([]byte, 16)
		rand.Read(b)
		u.Hy2Password = u.Username + ":" + hex.EncodeToString(b)
	}
	return nil
}

func (u *User) SetLoginPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.LoginPassword = string(hash)
	return nil
}

func (u *User) CheckLoginPassword(password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(u.LoginPassword), []byte(password)) == nil
}

func (u *User) IsExpired() bool {
	return !u.ExpiresAt.IsZero() && time.Now().After(u.ExpiresAt)
}

func (u *User) TrafficExceeded() bool {
	return u.TrafficLimit > 0 && u.TrafficUsed >= u.TrafficLimit
}

func (u *User) IsActive() bool {
	return u.Enabled && !u.IsExpired() && !u.TrafficExceeded()
}

// EffectiveNodeIDs returns nil if user has unrestricted access (NodeIDs "all" or empty),
// otherwise a set of allowed node IDs parsed from CSV.
func (u *User) EffectiveNodeIDs() map[uint]bool {
	if u.NodeIDs == "" || u.NodeIDs == "all" {
		return nil
	}
	result := make(map[uint]bool)
	for _, part := range strings.Split(u.NodeIDs, ",") {
		part = strings.TrimSpace(part)
		if part == "" { continue }
		if id, err := strconv.ParseUint(part, 10, 64); err == nil {
			result[uint(id)] = true
		}
	}
	return result
}
