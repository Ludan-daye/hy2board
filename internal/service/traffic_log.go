package service

import (
	"log"
	"time"

	"github.com/ludandaye/hy2board/internal/database"
	"github.com/ludandaye/hy2board/internal/model"
)

// StartTrafficLogger ticks every interval and records a TrafficLog row for each
// (node, user) pair with non-zero traffic in the current snapshot.
// Also spawns a daily prune goroutine that deletes rows older than 365 days.
func StartTrafficLogger(interval time.Duration) {
	tick := func() {
		snap := GetTrafficSnapshot()
		now := time.Now()
		rows := make([]model.TrafficLog, 0, len(snap))
		for _, n := range snap {
			if n.Traffic == nil {
				continue
			}
			for user, t := range n.Traffic {
				if t.TX == 0 && t.RX == 0 {
					continue
				}
				rows = append(rows, model.TrafficLog{
					SampledAt: now,
					NodeID:    n.ID,
					Username:  user,
					TX:        t.TX,
					RX:        t.RX,
				})
			}
		}
		if len(rows) > 0 {
			if err := database.DB.CreateInBatches(rows, 50).Error; err != nil {
				log.Printf("traffic_log insert failed: %v", err)
			}
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

	// Daily prune at 03:00 UTC
	go func() {
		for {
			now := time.Now().UTC()
			next := time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, time.UTC)
			if !next.After(now) {
				next = next.Add(24 * time.Hour)
			}
			time.Sleep(time.Until(next))
			cutoff := time.Now().Add(-365 * 24 * time.Hour)
			res := database.DB.Where("sampled_at < ?", cutoff).Delete(&model.TrafficLog{})
			log.Printf("traffic_log prune: deleted %d rows older than %s",
				res.RowsAffected, cutoff.Format("2006-01-02"))
		}
	}()
}
