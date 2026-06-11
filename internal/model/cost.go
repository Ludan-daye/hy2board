package model

import (
	"time"

	"gorm.io/gorm"
)

// Cost is one operating expense event. AmountCents is stored in CNY cents.
type Cost struct {
	gorm.Model
	Name        string    `gorm:"not null" json:"name"`
	Category    string    `gorm:"default:'';index" json:"category"`
	AmountCents int64     `gorm:"not null;default:0" json:"amount_cents"`
	Note        string    `gorm:"default:''" json:"note"`
	Operator    string    `gorm:"default:''" json:"operator"`
	IncurredAt  time.Time `gorm:"index;not null" json:"incurred_at"`
}
