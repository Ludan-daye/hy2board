package model

import "gorm.io/gorm"

type Plan struct {
	gorm.Model
	Name          string `gorm:"uniqueIndex;not null" json:"name"`
	TrafficLimit  int64  `json:"traffic_limit"`              // bytes; 0 = unlimited
	MaxIPs        int    `gorm:"default:0" json:"max_ips"`   // 0 = unlimited concurrent source IPs
	DurationDays  int    `json:"duration_days"`              // for ExpiresAt calc
	NodeIDs       string `gorm:"default:'all'" json:"node_ids"`
	RuleAI        bool   `json:"rule_ai"`
	RuleStreaming bool   `json:"rule_streaming"`
	RuleChina     bool   `gorm:"default:true" json:"rule_china"`
	RuleAdBlock   bool   `json:"rule_ad_block"`
	AutoReset     bool   `json:"auto_reset"`
	SortOrder     int    `gorm:"default:0" json:"sort_order"`

	// Static IP / chain-proxy override (Plan-as-IP)
	ProxyType     string `gorm:"default:''" json:"proxy_type"`     // "socks5" | "http" | ""
	ProxyHost     string `gorm:"default:''" json:"proxy_host"`
	ProxyPort     int    `gorm:"default:0"  json:"proxy_port"`
	ProxyUsername string `gorm:"default:''" json:"proxy_username"`
	ProxyPassword string `gorm:"default:''" json:"-"`              // write-only via API
	ProxyNote     string `gorm:"default:''" json:"proxy_note"`
	PriceCents    int64  `gorm:"default:0" json:"price_cents"`
}
