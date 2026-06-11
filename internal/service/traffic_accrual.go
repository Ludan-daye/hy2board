package service

import (
	"github.com/ludandaye/hy2board/internal/model"
	"gorm.io/gorm"
)

// trafficUsageDelta computes, per username, the positive byte delta (TX+RX) to
// add to users.traffic_used between two cumulative per-node-per-user snapshots.
//
// Deltas are computed per node so a single node restarting (its counter resets
// to a smaller value) is handled independently via counterDelta and never makes
// another node's growth go negative.
//
// A (node, user) pair seen for the FIRST time (absent from prev) contributes 0
// and only establishes a baseline. This is critical: on the very first snapshot
// after deploy prev is empty, so the large cumulative counters are NOT added
// retroactively -- usage accrues forward from now and no existing user is blocked.
func trafficUsageDelta(prev, now map[uint]map[string]TrafficData) map[string]int64 {
	out := make(map[string]int64)
	for nodeID, users := range now {
		prevUsers := prev[nodeID]
		if prevUsers == nil {
			continue // first time we observe this node: baseline only
		}
		for u, cur := range users {
			p, ok := prevUsers[u]
			if !ok {
				continue // first time we observe this user on this node: baseline only
			}
			d := counterDelta(p.TX, cur.TX) + counterDelta(p.RX, cur.RX)
			if d > 0 {
				out[u] += d
			}
		}
	}
	return out
}

// persistTrafficUsage increments users.traffic_used by each user's accrued delta.
// Uses an atomic SQL increment so a concurrent reset (traffic_used = 0) is not
// clobbered by a stale read-modify-write.
func persistTrafficUsage(db *gorm.DB, deltas map[string]int64) {
	for u, d := range deltas {
		if d <= 0 {
			continue
		}
		db.Model(&model.User{}).
			Where("username = ?", u).
			UpdateColumn("traffic_used", gorm.Expr("traffic_used + ?", d))
	}
}
