package service

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/ludandaye/hy2board/internal/config"
	"github.com/ludandaye/hy2board/internal/database"
	"github.com/ludandaye/hy2board/internal/model"
)

const (
	nodeProbeFailThreshold = 3
	nodeProbeAlertCooldown = 30 * time.Minute
	nodeProbeResultKeep    = 24 * time.Hour
)

type nodeProbeCheck struct {
	target    string
	status    string
	latencyMS int
	err       string
}

// StartHealthChecker probes each node on a fixed interval and updates both the
// probe state table and nodes.healthy with debounce.
func StartHealthChecker(interval time.Duration) {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	go func() {
		runNodeHealthCheck()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			runNodeHealthCheck()
		}
	}()
}

func runNodeHealthCheck() {
	var nodes []model.Node
	if err := database.DB.Order("sort_order asc, id asc").Find(&nodes).Error; err != nil {
		log.Printf("node probe: list nodes failed: %v", err)
		return
	}
	for _, n := range nodes {
		probeNode(n)
	}
	database.DB.Where("checked_at < ?", time.Now().Add(-nodeProbeResultKeep)).Delete(&model.NodeProbeResult{})
}

func probeNode(n model.Node) {
	now := time.Now()
	checks := []nodeProbeCheck{
		checkTrafficAPI(n, "/online"),
		checkTrafficAPI(n, "/traffic"),
		checkUDPPort(n),
	}

	status := model.NodeProbeStatusOK
	bestLatency := 0
	errs := make([]string, 0)
	for _, c := range checks {
		if c.latencyMS > 0 && (bestLatency == 0 || c.latencyMS < bestLatency) {
			bestLatency = c.latencyMS
		}
		if c.status == model.NodeProbeStatusFailed {
			status = model.NodeProbeStatusFailed
			errs = append(errs, fmt.Sprintf("%s: %s", c.target, c.err))
		} else if c.status == model.NodeProbeStatusWarning && status == model.NodeProbeStatusOK {
			status = model.NodeProbeStatusWarning
			if c.err != "" {
				errs = append(errs, fmt.Sprintf("%s: %s", c.target, c.err))
			}
		}
		database.DB.Create(&model.NodeProbeResult{
			NodeID:    n.ID,
			Target:    c.target,
			Status:    c.status,
			LatencyMS: c.latencyMS,
			Error:     c.err,
			CheckedAt: now,
		})
	}

	lastErr := strings.Join(errs, "; ")
	if len(lastErr) > 500 {
		lastErr = lastErr[:500]
	}

	var state model.NodeProbeState
	err := database.DB.Where("node_id = ?", n.ID).First(&state).Error
	if err != nil {
		state = model.NodeProbeState{NodeID: n.ID}
	}
	prevAlerted := state.AlertedDown

	state.Status = status
	state.LastCheckedAt = &now
	state.LastLatencyMS = bestLatency
	state.LastError = lastErr
	if status == model.NodeProbeStatusFailed {
		state.FailStreak++
		state.LastFailedAt = &now
	} else {
		state.FailStreak = 0
		state.LastOKAt = &now
		if state.AlertedDown {
			state.AlertedDown = false
		}
	}

	database.DB.Save(&state)
	syncNodeHealthy(n, state)
	maybeAlertNodeProbe(n, state, prevAlerted)
}

func checkTrafficAPI(n model.Node, path string) nodeProbeCheck {
	target := "traffic_api" + path
	if n.TrafficAPI == "" {
		return nodeProbeCheck{target: target, status: model.NodeProbeStatusWarning, err: "not configured"}
	}
	client := &http.Client{Timeout: 4 * time.Second}
	req, err := http.NewRequest("GET", strings.TrimRight(n.TrafficAPI, "/")+path, nil)
	if err != nil {
		return nodeProbeCheck{target: target, status: model.NodeProbeStatusFailed, err: err.Error()}
	}
	if n.TrafficSecret != "" {
		req.Header.Set("Authorization", n.TrafficSecret)
	}
	start := time.Now()
	resp, err := client.Do(req)
	latency := int(time.Since(start).Milliseconds())
	if err != nil {
		return nodeProbeCheck{target: target, status: model.NodeProbeStatusFailed, latencyMS: latency, err: err.Error()}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nodeProbeCheck{target: target, status: model.NodeProbeStatusFailed, latencyMS: latency, err: fmt.Sprintf("http %d", resp.StatusCode)}
	}
	if latency > 2000 {
		return nodeProbeCheck{target: target, status: model.NodeProbeStatusWarning, latencyMS: latency, err: fmt.Sprintf("slow %dms", latency)}
	}
	return nodeProbeCheck{target: target, status: model.NodeProbeStatusOK, latencyMS: latency}
}

func checkUDPPort(n model.Node) nodeProbeCheck {
	target := "udp_port"
	addr := fmt.Sprintf("%s:%d", n.Host, n.Port)
	start := time.Now()
	conn, err := net.DialTimeout("udp", addr, 3*time.Second)
	latency := int(time.Since(start).Milliseconds())
	if err != nil {
		return nodeProbeCheck{target: target, status: model.NodeProbeStatusWarning, latencyMS: latency, err: err.Error()}
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
	if _, err := conn.Write([]byte{0}); err != nil {
		return nodeProbeCheck{target: target, status: model.NodeProbeStatusWarning, latencyMS: latency, err: err.Error()}
	}
	return nodeProbeCheck{target: target, status: model.NodeProbeStatusOK, latencyMS: latency}
}

func syncNodeHealthy(n model.Node, state model.NodeProbeState) {
	healthy := n.Healthy
	if state.Status == model.NodeProbeStatusFailed && state.FailStreak >= nodeProbeFailThreshold {
		healthy = false
	}
	if state.Status != model.NodeProbeStatusFailed {
		healthy = true
	}
	if healthy != n.Healthy {
		database.DB.Model(&model.Node{}).Where("id = ?", n.ID).Update("healthy", healthy)
	}
}

func maybeAlertNodeProbe(n model.Node, state model.NodeProbeState, prevAlerted bool) {
	if !config.C.HasTelegram() {
		return
	}
	adminID, ok := getAdminTelegramID()
	if !ok {
		return
	}

	now := time.Now()
	if state.Status == model.NodeProbeStatusFailed && state.FailStreak >= nodeProbeFailThreshold {
		cooling := state.LastAlertAt != nil && now.Sub(*state.LastAlertAt) < nodeProbeAlertCooldown
		if state.AlertedDown && cooling {
			return
		}
		text := fmt.Sprintf(
			"🚨 节点异常\n\n节点：%s\n地址：%s:%d\n连续失败：%d 次\n原因：%s\n时间：%s",
			n.Name, n.Host, n.Port, state.FailStreak, emptyDash(state.LastError), now.In(cstLoc()).Format("2006-01-02 15:04:05"),
		)
		if sendAdminNodeProbeAlert(adminID, text) {
			state.AlertedDown = true
			state.LastAlertAt = &now
			database.DB.Model(&model.NodeProbeState{}).Where("node_id = ?", n.ID).Updates(map[string]interface{}{
				"alerted_down":  true,
				"last_alert_at": now,
			})
		}
		return
	}

	if prevAlerted {
		text := fmt.Sprintf(
			"✅ 节点恢复\n\n节点：%s\n地址：%s:%d\n状态：%s\n延迟：%dms\n时间：%s",
			n.Name, n.Host, n.Port, state.Status, state.LastLatencyMS, now.In(cstLoc()).Format("2006-01-02 15:04:05"),
		)
		if sendAdminNodeProbeAlert(adminID, text) {
			state.AlertedDown = false
			state.LastAlertAt = &now
			database.DB.Model(&model.NodeProbeState{}).Where("node_id = ?", n.ID).Updates(map[string]interface{}{
				"alerted_down":  false,
				"last_alert_at": now,
			})
		}
	}
}

func sendAdminNodeProbeAlert(adminID int64, text string) bool {
	bot, err := tgbotapi.NewBotAPI(config.C.Telegram.BotToken)
	if err != nil {
		log.Printf("node probe: telegram bot init failed: %v", err)
		return false
	}
	if _, err := bot.Send(tgbotapi.NewMessage(adminID, text)); err != nil {
		log.Printf("node probe: telegram alert failed: %v", err)
		return false
	}
	return true
}

func emptyDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}
