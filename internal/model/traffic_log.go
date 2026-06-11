package model

import "time"

type TrafficLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	SampledAt time.Time `gorm:"index" json:"sampled_at"`
	NodeID    uint      `gorm:"index" json:"node_id"`
	Username  string    `gorm:"index" json:"username"`
	TX        int64     `json:"tx"`
	RX        int64     `json:"rx"`
}
