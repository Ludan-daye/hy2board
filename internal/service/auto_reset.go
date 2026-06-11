package service

import (
	"log"
	"time"

	"github.com/ludandaye/hy2board/internal/database"
	"github.com/ludandaye/hy2board/internal/model"
)

// StartAutoResetScheduler ticks every interval and zeros TrafficUsed for any
// user with AutoReset=true whose LastResetAt is >= 30 days old.
func StartAutoResetScheduler(interval time.Duration) {
	tick := func() {
		threshold := time.Now().Add(-30 * 24 * time.Hour)
		var due []model.User
		if err := database.DB.Where("auto_reset = ? AND last_reset_at <= ?", true, threshold).
			Find(&due).Error; err != nil {
			log.Printf("auto-reset query failed: %v", err)
			return
		}
		for _, u := range due {
			oldUsed := u.TrafficUsed
			err := database.DB.Model(&model.User{}).
				Where("id = ? AND auto_reset = ? AND last_reset_at <= ?", u.ID, true, threshold).
				Updates(map[string]interface{}{
					"traffic_used":  0,
					"last_reset_at": time.Now(),
				}).Error
			if err != nil {
				log.Printf("auto-reset failed user=%s err=%v", u.Username, err)
				continue
			}
			ResetUserSpeedBaseline(u.Username)
			log.Printf("auto-reset user=%s old_used=%d", u.Username, oldUsed)
		}
	}
	tick()
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for range t.C {
			tick()
		}
	}()
}
