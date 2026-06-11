package model

import (
	"time"

	"gorm.io/gorm"
)

// Payment is one income event. AmountCents stored in CNY cents (int64) to
// avoid float precision. Soft-deleted via gorm.Model.DeletedAt.
type Payment struct {
	gorm.Model
	UserID      uint      `gorm:"index;not null" json:"user_id"`
	PlanID      uint      `gorm:"index;default:0" json:"plan_id"`
	AmountCents int64     `gorm:"not null;default:0" json:"amount_cents"`
	DaysAdded   int       `gorm:"not null;default:0" json:"days_added"`
	Kind        string    `gorm:"default:'renew'" json:"kind"` // "new" | "renew"
	Note        string    `gorm:"default:''" json:"note"`
	Operator    string    `gorm:"default:''" json:"operator"`
	PaidAt      time.Time `gorm:"index;not null" json:"paid_at"`
}
