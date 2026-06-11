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
}
