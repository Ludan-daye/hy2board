package model

import "gorm.io/gorm"

type Node struct {
	gorm.Model
	Name         string `gorm:"not null" json:"name"`
	Host         string `gorm:"not null" json:"host"`
	Port         int    `gorm:"not null" json:"port"`
	Password     string `gorm:"not null" json:"password"`
	SNI          string `json:"sni"`
	Insecure     bool   `json:"insecure"`
	ObfsType     string `json:"obfs_type"`
	ObfsPassword string `json:"obfs_password"`
	TrafficAPI   string `json:"traffic_api"`
	TrafficSecret string `json:"traffic_secret"`
	Healthy      bool   `gorm:"default:true" json:"healthy"`
	SortOrder    int    `gorm:"default:0" json:"sort_order"`

	// VLESS+Reality fallback (Phase 1). Private key never stored here.
	VlessEnabled     bool   `gorm:"default:false" json:"vless_enabled"`
	VlessPort        int    `gorm:"default:0" json:"vless_port"`
	RealityPubkey    string `gorm:"default:''" json:"reality_pubkey"`
	RealityShortID   string `gorm:"default:''" json:"reality_shortid"`
	RealitySNI       string `gorm:"default:''" json:"reality_sni"`
	VlessStatsAPI    string `gorm:"default:''" json:"vless_stats_api"`
	VlessStatsSecret string `gorm:"default:''" json:"-"`
}
