package service

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/ludandaye/hy2board/internal/database"
	"github.com/ludandaye/hy2board/internal/model"
)

// parseVlessStats decodes a node agent's cumulative per-user counters.
func parseVlessStats(body []byte) (map[string]TrafficData, error) {
	var m map[string]TrafficData
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func fetchVlessStats(n model.Node) map[string]TrafficData {
	if n.VlessStatsAPI == "" {
		return nil
	}
	req, err := http.NewRequest("GET", n.VlessStatsAPI, nil)
	if err != nil {
		return nil
	}
	if n.VlessStatsSecret != "" {
		req.Header.Set("Authorization", n.VlessStatsSecret)
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil
	}
	m, err := parseVlessStats(body)
	if err != nil {
		return nil
	}
	return m
}

// StartVlessTrafficPoller meters VLESS usage into users.traffic_used using the
// same per-node-per-user delta accrual as HY2 (handles first-sight baseline and
// counter resets via trafficUsageDelta).
func StartVlessTrafficPoller(interval time.Duration) {
	prev := map[uint]map[string]TrafficData{}
	tick := func() {
		var nodes []model.Node
		database.DB.Where("vless_enabled = ?", true).Find(&nodes)
		now := map[uint]map[string]TrafficData{}
		for _, n := range nodes {
			if s := fetchVlessStats(n); s != nil {
				now[n.ID] = s
			}
		}
		persistTrafficUsage(database.DB, trafficUsageDelta(prev, now))
		prev = now
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
