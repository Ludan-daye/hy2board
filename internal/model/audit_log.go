package model

import "gorm.io/gorm"

type AuditLog struct {
	gorm.Model
	Actor      string `gorm:"index" json:"actor"`
	Action     string `gorm:"index" json:"action"`
	Entity     string `gorm:"index" json:"entity"`
	EntityID   uint   `gorm:"index" json:"entity_id"`
	EntityName string `json:"entity_name"`
	Detail     string `json:"detail"`
	IP         string `json:"ip"`
}
