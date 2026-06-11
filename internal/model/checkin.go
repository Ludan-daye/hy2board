package model

import "time"

type Checkin struct {
	UserID             uint      `gorm:"primaryKey" json:"user_id"` // references User.ID
	LastCheckIn        time.Time `json:"last_checkin"`
	Streak             int       `json:"streak"`
	TotalCheckIns      int       `json:"total_checkins"`
	LastRewardedStreak int       `json:"last_rewarded_streak"` // track last milestone rewarded
}
