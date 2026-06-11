package service

import (
	"sort"
	"time"

	"github.com/ludandaye/hy2board/internal/model"
	"gorm.io/gorm"
)

type TrafficDelta struct {
	Username string
	TX       int64
	RX       int64
	Total    int64
}

type trafficDeltaLogRow struct {
	SampledAt time.Time
	NodeID    uint
	Username  string
	TX        int64
	RX        int64
}

type trafficCounterState struct {
	seen bool
	tx   int64
	rx   int64
}

func TrafficDeltas(db *gorm.DB, start, end time.Time, limit int) ([]TrafficDelta, error) {
	if db == nil {
		return nil, nil
	}
	var logs []trafficDeltaLogRow
	q := db.Model(&model.TrafficLog{}).
		Select("sampled_at, node_id, username, tx, rx").
		Where("sampled_at >= ?", start)
	if !end.IsZero() {
		q = q.Where("sampled_at < ?", end)
	}
	if err := q.Order("username asc, node_id asc, sampled_at asc, id asc").Scan(&logs).Error; err != nil {
		return nil, err
	}

	byUser := map[string]*TrafficDelta{}
	prev := map[struct {
		username string
		nodeID   uint
	}]trafficCounterState{}

	for _, row := range logs {
		if row.Username == "" {
			continue
		}
		key := struct {
			username string
			nodeID   uint
		}{username: row.Username, nodeID: row.NodeID}
		state := prev[key]
		if !state.seen {
			prev[key] = trafficCounterState{seen: true, tx: row.TX, rx: row.RX}
			continue
		}

		dtx := counterDelta(state.tx, row.TX)
		drx := counterDelta(state.rx, row.RX)
		if dtx > 0 || drx > 0 {
			d := byUser[row.Username]
			if d == nil {
				d = &TrafficDelta{Username: row.Username}
				byUser[row.Username] = d
			}
			d.TX += dtx
			d.RX += drx
			d.Total += dtx + drx
		}
		prev[key] = trafficCounterState{seen: true, tx: row.TX, rx: row.RX}
	}

	out := make([]TrafficDelta, 0, len(byUser))
	for _, d := range byUser {
		out = append(out, *d)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Total == out[j].Total {
			return out[i].Username < out[j].Username
		}
		return out[i].Total > out[j].Total
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func AggregateTrafficDelta(db *gorm.DB, start, end time.Time) (int64, int64, error) {
	rows, err := TrafficDeltas(db, start, end, 0)
	if err != nil {
		return 0, 0, err
	}
	var tx, rx int64
	for _, row := range rows {
		tx += row.TX
		rx += row.RX
	}
	return tx, rx, nil
}

func counterDelta(prev, cur int64) int64 {
	if cur >= prev {
		return cur - prev
	}
	if cur > 0 {
		return cur
	}
	return 0
}
