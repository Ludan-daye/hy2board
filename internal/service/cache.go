package service

import (
	"sync"
	"time"

	"github.com/ludandaye/hy2board/internal/database"
	"github.com/ludandaye/hy2board/internal/model"
)

type NodeSnapshot struct {
	ID      uint
	Name    string
	Host    string
	Port    int
	Healthy bool
	Online  int
	OnlineUsers map[string]int
	Traffic map[string]TrafficData
}

// HistoryPoint is one moment in the rolling time-series kept for charts.
// TotalTX/TotalRX is aggregated across all nodes at that moment.
// SpeedTX/SpeedRX is bytes/sec computed vs the prior point.
// PerNode holds each node's speed at that moment keyed by node name.
type HistoryPoint struct {
	Time    time.Time                   `json:"time"`
	TotalTX int64                       `json:"total_tx"`
	TotalRX int64                       `json:"total_rx"`
	SpeedTX int64                       `json:"speed_tx"`
	SpeedRX int64                       `json:"speed_rx"`
	PerNode map[string]NodeSpeedAtPoint `json:"per_node,omitempty"`
}

type NodeSpeedAtPoint struct {
	SpeedTX int64 `json:"speed_tx"`
	SpeedRX int64 `json:"speed_rx"`
}

type UserBrief struct {
	ID           uint      `json:"id"`
	Username     string    `json:"username"`
	TrafficUsed  int64     `json:"traffic_used"`
	TrafficLimit int64     `json:"traffic_limit"`
	ExpiresAt    time.Time `json:"expires_at"`
	DaysUntil    int       `json:"days_until"`
	PercentUsed  int       `json:"percent_used"`
}

type NodeBrief struct {
	ID      uint   `json:"id"`
	Name    string `json:"name"`
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Healthy bool   `json:"healthy"`
}

type UserBuckets struct {
	ExpiringSoon []UserBrief `json:"expiring_soon"`
	NearLimit    []UserBrief `json:"near_limit"`
	OverLimit    []UserBrief `json:"over_limit"`
	Expired      []UserBrief `json:"expired"`
	Disabled     []UserBrief `json:"disabled"`
	DeadNodes    []NodeBrief `json:"dead_nodes"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

// UserStat is pre-aggregated per-user traffic across all nodes, with speed.
type UserStat struct {
	Username string                 `json:"username"`
	TotalTX  int64                  `json:"tx"`
	TotalRX  int64                  `json:"rx"`
	SpeedTX  int64                  `json:"speed_tx"`
	SpeedRX  int64                  `json:"speed_rx"`
	PerNode  map[string]TrafficData `json:"per_node"`
}

const historyLimit = 180 // 3 min at 1s / 15 min at 5s interval

var (
	cacheMu sync.RWMutex

	trafficCache []NodeSnapshot
	cachedAt     time.Time

	history         []HistoryPoint                    // rolling ring buffer
	userStats       map[string]*UserStat              // pre-aggregated per user
	prevByUser      map[string]TrafficData            // for speed delta
	prevByNodeUser  map[uint]map[string]TrafficData   // per-node per-user delta
	prevByNodeTotal map[string]struct{ TX, RX int64 } // per-node totals prev
)

var (
	userBuckets   UserBuckets
	userBucketsMu sync.RWMutex
)

// GetTrafficSnapshot returns the last cached node-level snapshot. Instant.
func GetTrafficSnapshot() []NodeSnapshot {
	cacheMu.RLock()
	defer cacheMu.RUnlock()
	out := make([]NodeSnapshot, len(trafficCache))
	copy(out, trafficCache)
	return out
}

// GetHistory returns the rolling time-series. Instant.
func GetHistory() []HistoryPoint {
	cacheMu.RLock()
	defer cacheMu.RUnlock()
	out := make([]HistoryPoint, len(history))
	copy(out, history)
	return out
}

// GetUserStat returns the pre-aggregated stats for one user. Instant.
func GetUserStat(username string) (UserStat, bool) {
	cacheMu.RLock()
	defer cacheMu.RUnlock()
	s, ok := userStats[username]
	if !ok {
		return UserStat{}, false
	}
	return *s, true
}

func GetUserBuckets() UserBuckets {
	userBucketsMu.RLock()
	defer userBucketsMu.RUnlock()
	return userBuckets
}

// ResetUserSpeedBaseline zeroes the internal prev_by_user entry for a user
// so the next speed delta computes against 0 (called after traffic reset).
func ResetUserSpeedBaseline(username string) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if prevByUser != nil {
		delete(prevByUser, username)
	}
}

func StartUserBucketCache(interval time.Duration) {
	refresh := func() {
		var users []model.User
		database.DB.Find(&users)
		var nodes []model.Node
		database.DB.Find(&nodes)

		b := BuildUserBuckets(users, nodes, time.Now(), func(username string) int64 {
			if s, ok := GetUserStat(username); ok {
				return s.TotalTX + s.TotalRX
			}
			return 0
		})
		userBucketsMu.Lock()
		userBuckets = b
		userBucketsMu.Unlock()
	}
	refresh()
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for range t.C {
			refresh()
		}
	}()
}

func BuildUserBuckets(users []model.User, nodes []model.Node, now time.Time, liveUsed func(username string) int64) UserBuckets {
	b := UserBuckets{UpdatedAt: now}
	for _, u := range users {
		used := u.TrafficUsed
		if liveUsed != nil {
			if live := liveUsed(u.Username); live > used {
				used = live
			}
		}
		days := -1
		if !u.ExpiresAt.IsZero() {
			days = int(u.ExpiresAt.Sub(now).Hours() / 24)
		}
		pct := 0
		if u.TrafficLimit > 0 {
			pct = int(float64(used) * 100.0 / float64(u.TrafficLimit))
		}
		brief := UserBrief{
			ID:           u.ID,
			Username:     u.Username,
			TrafficUsed:  used,
			TrafficLimit: u.TrafficLimit,
			ExpiresAt:    u.ExpiresAt,
			DaysUntil:    days,
			PercentUsed:  pct,
		}
		if !u.Enabled {
			b.Disabled = append(b.Disabled, brief)
			continue
		}
		if !u.ExpiresAt.IsZero() && u.ExpiresAt.Before(now) {
			b.Expired = append(b.Expired, brief)
		} else if days >= 0 && days <= 7 {
			b.ExpiringSoon = append(b.ExpiringSoon, brief)
		}
		if u.TrafficLimit > 0 && used >= u.TrafficLimit {
			b.OverLimit = append(b.OverLimit, brief)
		} else if pct >= 80 {
			b.NearLimit = append(b.NearLimit, brief)
		}
	}
	for _, n := range nodes {
		if !n.Healthy {
			b.DeadNodes = append(b.DeadNodes, NodeBrief{
				ID: n.ID, Name: n.Name, Host: n.Host, Port: n.Port, Healthy: false,
			})
		}
	}
	return b
}

// StartTrafficCache refreshes the cache concurrently every interval and
// also maintains a rolling history + per-user speed.
func StartTrafficCache(interval time.Duration) {
	refresh := func() {
		var nodes []model.Node
		database.DB.Find(&nodes)

		results := make([]NodeSnapshot, len(nodes))
		var wg sync.WaitGroup
		for i, n := range nodes {
			wg.Add(1)
			go func(i int, n model.Node) {
				defer wg.Done()
				snap := NodeSnapshot{
					ID:      n.ID,
					Name:    n.Name,
					Host:    n.Host,
					Port:    n.Port,
					Healthy: n.Healthy,
				}
				var inner sync.WaitGroup
				inner.Add(2)
				go func() {
					defer inner.Done()
					if t, err := GetNodeTraffic(n); err == nil {
						snap.Traffic = t
					}
				}()
				go func() {
					defer inner.Done()
					if m, err := GetNodeOnlineMap(n); err == nil {
						snap.OnlineUsers = m
						snap.Online = len(m)
					}
				}()
				inner.Wait()
				results[i] = snap
			}(i, n)
		}
		wg.Wait()

		enforceSharing(nodes, results)

		// Aggregate per-user totals across nodes for this snapshot
		nowByUser := make(map[string]TrafficData)
		nowByNodeUser := make(map[uint]map[string]TrafficData)
		var totalTX, totalRX int64
		for _, n := range results {
			if n.Traffic == nil {
				continue
			}
			nowByNodeUser[n.ID] = make(map[string]TrafficData)
			for u, t := range n.Traffic {
				nowByNodeUser[n.ID][u] = t
				agg := nowByUser[u]
				agg.TX += t.TX
				agg.RX += t.RX
				nowByUser[u] = agg
				totalTX += t.TX
				totalRX += t.RX
			}
		}

		// Build per-user stats with speed (delta vs previous snapshot)
		now := time.Now()
		intervalSec := float64(interval) / float64(time.Second)
		if intervalSec <= 0 {
			intervalSec = 1
		}

		cacheMu.Lock()

		// Compute delta per user
		newUserStats := make(map[string]*UserStat, len(nowByUser))
		for u, t := range nowByUser {
			stat := &UserStat{
				Username: u,
				TotalTX:  t.TX,
				TotalRX:  t.RX,
				PerNode:  make(map[string]TrafficData),
			}
			if prev, ok := prevByUser[u]; ok {
				dtx := t.TX - prev.TX
				drx := t.RX - prev.RX
				if dtx < 0 {
					dtx = 0
				}
				if drx < 0 {
					drx = 0
				}
				stat.SpeedTX = int64(float64(dtx) / intervalSec)
				stat.SpeedRX = int64(float64(drx) / intervalSec)
			}
			newUserStats[u] = stat
		}
		// Fill per-node traffic per user
		for nodeID, userMap := range nowByNodeUser {
			// Find node name
			nodeName := ""
			for _, n := range results {
				if n.ID == nodeID {
					nodeName = n.Name
					break
				}
			}
			for u, t := range userMap {
				if s, ok := newUserStats[u]; ok {
					s.PerNode[nodeName] = t
				}
			}
		}

		// Per-node aggregate totals at this moment
		perNodeTotal := make(map[string]struct{ TX, RX int64 })
		for _, n := range results {
			if n.Traffic == nil {
				continue
			}
			var tx, rx int64
			for _, t := range n.Traffic {
				tx += t.TX
				rx += t.RX
			}
			perNodeTotal[n.Name] = struct{ TX, RX int64 }{tx, rx}
		}

		// Append to rolling history with per-node speeds
		point := HistoryPoint{
			Time:    now,
			TotalTX: totalTX,
			TotalRX: totalRX,
			PerNode: make(map[string]NodeSpeedAtPoint),
		}
		if len(history) > 0 {
			prev := history[len(history)-1]
			dtx := totalTX - prev.TotalTX
			drx := totalRX - prev.TotalRX
			if dtx < 0 {
				dtx = 0
			}
			if drx < 0 {
				drx = 0
			}
			point.SpeedTX = int64(float64(dtx) / intervalSec)
			point.SpeedRX = int64(float64(drx) / intervalSec)

			// Per-node speed = delta from prev snapshot
			for name, cur := range perNodeTotal {
				prevTotal, ok := prevByNodeTotal[name]
				var sTX, sRX int64
				if ok {
					dtx := cur.TX - prevTotal.TX
					drx := cur.RX - prevTotal.RX
					if dtx < 0 {
						dtx = 0
					}
					if drx < 0 {
						drx = 0
					}
					sTX = int64(float64(dtx) / intervalSec)
					sRX = int64(float64(drx) / intervalSec)
				}
				point.PerNode[name] = NodeSpeedAtPoint{SpeedTX: sTX, SpeedRX: sRX}
			}
		}
		history = append(history, point)
		if len(history) > historyLimit {
			history = history[len(history)-historyLimit:]
		}

		// Accrue per-user usage into users.traffic_used. Computed against the
		// previous snapshot BEFORE we overwrite it; persisted after the unlock so
		// DB I/O never runs under cacheMu. On the first snapshot prevByNodeUser is
		// empty, so nothing is added retroactively (no user is blocked on deploy).
		usageDeltas := trafficUsageDelta(prevByNodeUser, nowByNodeUser)

		// Commit state
		trafficCache = results
		cachedAt = now
		userStats = newUserStats
		prevByUser = nowByUser
		prevByNodeUser = nowByNodeUser
		prevByNodeTotal = perNodeTotal

		cacheMu.Unlock()

		persistTrafficUsage(database.DB, usageDeltas)
	}

	refresh()
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for range t.C {
			refresh()
		}
	}()
}
