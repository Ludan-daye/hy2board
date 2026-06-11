package model

import "gorm.io/gorm"

type CustomRoutingRule struct {
	gorm.Model
	Enabled   bool   `gorm:"default:true" json:"enabled"`
	Name      string `gorm:"not null" json:"name"`
	Kind      string `gorm:"not null" json:"kind"`
	Value     string `gorm:"not null" json:"value"`
	Policy    string `gorm:"not null" json:"policy"`
	SortOrder int    `gorm:"default:0" json:"sort_order"`
	Note      string `gorm:"default:''" json:"note"`
}
