package service

import (
	"log"
	"sync"
	"time"

	"github.com/ludandaye/hy2board/internal/database"
	"gorm.io/gorm"
)

// TodayTraffic is one user's upload and download over the last 24 hours.
// Values are summed from per-node counter deltas, matching the previous
// MAX(tx)-MIN(tx) calculation.
type TodayTraffic struct {
	TX int64
	RX int64
}

func (t TodayTraffic) Total() int64 { return t.TX + t.RX }

var (
	todayMu      sync.RWMutex
	todayRefresh sync.Mutex
	todayByUser  = map[string]TodayTraffic{}
)

// StartTodayTrafficCache refreshes the 24-hour totals in the background.
// List pages read the cache instead of scanning traffic_logs on every poll.
func StartTodayTrafficCache(interval time.Duration) {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	go refreshTodayTraffic()
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			refreshTodayTraffic()
		}
	}()
}

// TodayTrafficFor returns the cached 24-hour total. It is empty until the
// first refresh finishes.
func TodayTrafficFor(username string) TodayTraffic {
	todayMu.RLock()
	defer todayMu.RUnlock()
	return todayByUser[username]
}

// ReloadTodayTraffic recalculates the cache immediately.
func ReloadTodayTraffic() error {
	todayRefresh.Lock()
	defer todayRefresh.Unlock()
	return storeTodayTraffic()
}

// ClearTodayTrafficCache drops the cached totals.
func ClearTodayTrafficCache() {
	todayMu.Lock()
	todayByUser = map[string]TodayTraffic{}
	todayMu.Unlock()
}

func refreshTodayTraffic() {
	if database.DB == nil {
		return
	}
	if !todayRefresh.TryLock() {
		return
	}
	defer todayRefresh.Unlock()

	start := time.Now()
	if err := storeTodayTraffic(); err != nil {
		log.Printf("today traffic refresh failed after %s: %v", time.Since(start).Round(time.Millisecond), err)
		return
	}
	todayMu.RLock()
	n := len(todayByUser)
	todayMu.RUnlock()
	log.Printf("today traffic refresh users=%d took=%s", n, time.Since(start).Round(time.Millisecond))
}

func storeTodayTraffic() error {
	if database.DB == nil {
		return nil
	}
	totals, err := loadTodayTraffic(database.DB, time.Now().Add(-24*time.Hour))
	if err != nil {
		return err
	}
	todayMu.Lock()
	todayByUser = totals
	todayMu.Unlock()
	return nil
}

func loadTodayTraffic(db *gorm.DB, cutoff time.Time) (map[string]TodayTraffic, error) {
	cutoff = cutoff.UTC()
	type row struct {
		Username string
		TX       int64
		RX       int64
	}
	var rows []row
	// Range-scan the sampled_at index. A per-user lookup uses the username
	// index and walks that user's whole history, which is millions of rows.
	err := db.Raw(`
		SELECT username,
		       COALESCE(SUM(dtx), 0) AS tx,
		       COALESCE(SUM(drx), 0) AS rx
		FROM (
			SELECT username,
			       MAX(tx) - MIN(tx) AS dtx,
			       MAX(rx) - MIN(rx) AS drx
			FROM traffic_logs INDEXED BY idx_traffic_logs_sampled_at
			WHERE sampled_at >= ?
			GROUP BY username, node_id
		)
		GROUP BY username`, cutoff).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[string]TodayTraffic, len(rows))
	for _, r := range rows {
		if r.Username == "" {
			continue
		}
		out[r.Username] = TodayTraffic{TX: r.TX, RX: r.RX}
	}
	return out, nil
}
