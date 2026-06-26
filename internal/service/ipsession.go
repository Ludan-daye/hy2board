package service

import (
	"sort"
	"sync"
	"time"
)

const (
	ipWindow   = 15 * time.Minute // an IP is "active" if seen (re-authed) within this window
	alertEvery = 30 * time.Minute // per-user alert rate limit
)

type userIPState struct {
	ips         map[string]time.Time // ip -> last auth time
	lastAlert   time.Time
	lastBlocked time.Time
}

var (
	ipMu    sync.Mutex
	ipStore = map[uint]*userIPState{}
)

func ipStateLocked(userID uint) *userIPState {
	s := ipStore[userID]
	if s == nil {
		s = &userIPState{ips: map[string]time.Time{}}
		ipStore[userID] = s
	}
	return s
}

func pruneLocked(s *userIPState, now time.Time) {
	for ip, t := range s.ips {
		if now.Sub(t) > ipWindow {
			delete(s.ips, ip)
		}
	}
}

// TouchIP records an auth event. Returns whether the connection is allowed and the
// resulting distinct-IP count. A known IP always refreshes/allows; a new IP is allowed
// only when under the limit (0 = unlimited). A blocked attempt is NOT recorded but stamps
// lastBlocked for the enforcement loop to alert on.
func TouchIP(userID uint, ip string, limit int, now time.Time) (bool, int) {
	ipMu.Lock()
	defer ipMu.Unlock()
	s := ipStateLocked(userID)
	pruneLocked(s, now)
	if _, ok := s.ips[ip]; ok {
		s.ips[ip] = now
		return true, len(s.ips)
	}
	if limit == 0 || len(s.ips) < limit {
		s.ips[ip] = now
		return true, len(s.ips)
	}
	s.lastBlocked = now
	return false, len(s.ips)
}

// TrimOverLimit prunes stale IPs, then if the user holds more than limit IPs (e.g. the
// limit was just lowered) removes the oldest until len==limit. Returns how many were
// trimmed and the final distinct count.
func TrimOverLimit(userID uint, limit int, now time.Time) (int, int) {
	ipMu.Lock()
	defer ipMu.Unlock()
	s := ipStore[userID]
	if s == nil {
		return 0, 0
	}
	pruneLocked(s, now)
	if limit <= 0 || len(s.ips) <= limit {
		return 0, len(s.ips)
	}
	type kv struct {
		ip string
		t  time.Time
	}
	arr := make([]kv, 0, len(s.ips))
	for ip, t := range s.ips {
		arr = append(arr, kv{ip, t})
	}
	sort.Slice(arr, func(i, j int) bool { return arr[i].t.Before(arr[j].t) }) // oldest first
	remove := len(arr) - limit
	for i := 0; i < remove; i++ {
		delete(s.ips, arr[i].ip)
	}
	return remove, len(s.ips)
}

// DistinctActiveIPs returns the count of IPs seen within the active window.
func DistinctActiveIPs(userID uint, now time.Time) int {
	ipMu.Lock()
	defer ipMu.Unlock()
	s := ipStore[userID]
	if s == nil {
		return 0
	}
	pruneLocked(s, now)
	return len(s.ips)
}

// RecentlyBlocked reports whether a new IP was rejected for this user within `within`.
func RecentlyBlocked(userID uint, within time.Duration, now time.Time) bool {
	ipMu.Lock()
	defer ipMu.Unlock()
	s := ipStore[userID]
	if s == nil || s.lastBlocked.IsZero() {
		return false
	}
	return now.Sub(s.lastBlocked) <= within
}

// ShouldAlertSharing returns true at most once per alertEvery per user (and stamps it).
func ShouldAlertSharing(userID uint, now time.Time) bool {
	ipMu.Lock()
	defer ipMu.Unlock()
	s := ipStore[userID]
	if s == nil {
		return false
	}
	if !s.lastAlert.IsZero() && now.Sub(s.lastAlert) < alertEvery {
		return false
	}
	s.lastAlert = now
	return true
}
