package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ludandaye/hy2board/internal/model"
)

type TrafficData struct {
	TX int64 `json:"tx"`
	RX int64 `json:"rx"`
}

func GetNodeTraffic(node model.Node) (map[string]TrafficData, error) {
	if node.TrafficAPI == "" {
		return nil, fmt.Errorf("no traffic API configured")
	}

	client := &http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequest("GET", node.TrafficAPI+"/traffic", nil)
	if err != nil {
		return nil, err
	}
	if node.TrafficSecret != "" {
		req.Header.Set("Authorization", node.TrafficSecret)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var data map[string]TrafficData
	json.Unmarshal(body, &data)
	return data, nil
}

func GetNodeOnline(node model.Node) (int, error) {
	if node.TrafficAPI == "" {
		return 0, fmt.Errorf("no traffic API configured")
	}

	client := &http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequest("GET", node.TrafficAPI+"/online", nil)
	if err != nil {
		return 0, err
	}
	if node.TrafficSecret != "" {
		req.Header.Set("Authorization", node.TrafficSecret)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	return len(result), nil
}

// GetNodeOnlineMap returns HY2's per-user online connection counts (username -> count).
func GetNodeOnlineMap(node model.Node) (map[string]int, error) {
	if node.TrafficAPI == "" {
		return nil, fmt.Errorf("no traffic API configured")
	}
	client := &http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequest("GET", node.TrafficAPI+"/online", nil)
	if err != nil {
		return nil, err
	}
	if node.TrafficSecret != "" {
		req.Header.Set("Authorization", node.TrafficSecret)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var result map[string]int
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// KickUser disconnects all of a user's connections on a node via HY2 trafficStats /kick.
func KickUser(node model.Node, username string) error {
	if node.TrafficAPI == "" {
		return fmt.Errorf("no traffic API configured")
	}
	payload, _ := json.Marshal([]string{username})
	client := &http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequest("POST", node.TrafficAPI+"/kick", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if node.TrafficSecret != "" {
		req.Header.Set("Authorization", node.TrafficSecret)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("kick status %d", resp.StatusCode)
	}
	return nil
}
