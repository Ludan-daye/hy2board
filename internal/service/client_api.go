package service

import (
	"sort"
	"time"

	"github.com/ludandaye/hy2board/internal/model"
)

const ClientAPIVersion = 1
const MinimumClientVersion = "0.1.0"
const LatestClientVersion = "0.1.0"

type ClientVersionInfo struct {
	AppName              string    `json:"app_name"`
	Server               string    `json:"server"`
	APIVersion           int       `json:"api_version"`
	MinimumClientVersion string    `json:"minimum_client_version"`
	LatestClientVersion  string    `json:"latest_client_version"`
	ServerTime           time.Time `json:"server_time"`
}

type ClientProfile struct {
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Enabled   bool      `json:"enabled"`
	Active    bool      `json:"active"`
	ExpiresAt time.Time `json:"expires_at"`
}

type ClientPlan struct {
	ID            uint   `json:"id,omitempty"`
	Name          string `json:"name"`
	TrafficLimit  int64  `json:"traffic_limit"`
	DurationDays  int    `json:"duration_days"`
	PriceCents    int64  `json:"price_cents"`
	RuleAI        bool   `json:"rule_ai"`
	RuleStreaming bool   `json:"rule_streaming"`
	RuleChina     bool   `json:"rule_china"`
	RuleAdBlock   bool   `json:"rule_ad_block"`
	AutoReset     bool   `json:"auto_reset"`
}

type ClientTrafficSummary struct {
	TrafficLimit int64     `json:"traffic_limit"`
	TrafficUsed  int64     `json:"traffic_used"`
	Upload       int64     `json:"upload"`
	Download     int64     `json:"download"`
	Total        int64     `json:"total"`
	SpeedTX      int64     `json:"speed_tx"`
	SpeedRX      int64     `json:"speed_rx"`
	PercentUsed  int       `json:"percent_used"`
	ExpiresAt    time.Time `json:"expires_at"`
	ExpireUnix   int64     `json:"expire_unix"`
}

type ClientNode struct {
	ID            uint       `json:"id"`
	Name          string     `json:"name"`
	Host          string     `json:"host"`
	Port          int        `json:"port"`
	SNI           string     `json:"sni"`
	Insecure      bool       `json:"insecure"`
	ObfsType      string     `json:"obfs_type,omitempty"`
	ObfsPassword  string     `json:"obfs_password,omitempty"`
	Healthy       bool       `json:"healthy"`
	ProbeStatus   string     `json:"probe_status,omitempty"`
	LastLatencyMS int        `json:"last_latency_ms,omitempty"`
	LastCheckedAt *time.Time `json:"last_checked_at,omitempty"`
	LastError     string     `json:"last_error,omitempty"`
}

type ClientNodeTraffic struct {
	Node     string `json:"node"`
	Upload   int64  `json:"upload"`
	Download int64  `json:"download"`
	Total    int64  `json:"total"`
}

type ClientTrafficHistoryEntry struct {
	SampledAt time.Time `json:"sampled_at"`
	NodeID    uint      `json:"node_id"`
	TX        int64     `json:"tx"`
	RX        int64     `json:"rx"`
}

type ClientConfigRules struct {
	AI        bool `json:"ai"`
	Streaming bool `json:"streaming"`
	China     bool `json:"china"`
	AdBlock   bool `json:"ad_block"`
}

type ClientConfigTun struct {
	Enabled bool   `json:"enabled"`
	Mode    string `json:"mode"`
}

type ClientConfig struct {
	Protocol      string            `json:"protocol"`
	Auth          string            `json:"auth"`
	DefaultNodeID uint              `json:"default_node_id"`
	Nodes         []ClientNode      `json:"nodes"`
	TUN           ClientConfigTun   `json:"tun"`
	Rules         ClientConfigRules `json:"rules"`
}

type ClientAnnouncement struct {
	ID        string    `json:"id"`
	Severity  string    `json:"severity"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ClientHelpItem struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

type ClientDiagnostics struct {
	CanConnect       bool     `json:"can_connect"`
	ReasonCodes      []string `json:"reason_codes"`
	Warnings         []string `json:"warnings,omitempty"`
	AccountActive    bool     `json:"account_active"`
	AvailableNodes   int      `json:"available_nodes"`
	UnavailableNodes int      `json:"unavailable_nodes"`
	TrafficExceeded  bool     `json:"traffic_exceeded"`
	PlanExpired      bool     `json:"plan_expired"`
	AccountDisabled  bool     `json:"account_disabled"`
}

func BuildClientVersion(now time.Time) ClientVersionInfo {
	return ClientVersionInfo{
		AppName:              "selfladder",
		Server:               "hy2board",
		APIVersion:           ClientAPIVersion,
		MinimumClientVersion: MinimumClientVersion,
		LatestClientVersion:  LatestClientVersion,
		ServerTime:           now,
	}
}

func BuildClientFeatures() map[string]bool {
	return map[string]bool{
		"hysteria2":                 true,
		"tun_required":              true,
		"third_party_subscriptions": true,
		"announcements":             true,
		"diagnostics":               true,
		"write_account_actions":     false,
		"payments":                  false,
		"device_binding":            false,
	}
}

func BuildClientProfile(u model.User) ClientProfile {
	return ClientProfile{
		Username:  u.Username,
		Email:     u.Email,
		Enabled:   u.Enabled,
		Active:    u.IsActive(),
		ExpiresAt: u.ExpiresAt,
	}
}

func BuildClientPlan(u model.User, p *model.Plan) ClientPlan {
	if p != nil && p.ID != 0 {
		return ClientPlan{
			ID:            p.ID,
			Name:          p.Name,
			TrafficLimit:  p.TrafficLimit,
			DurationDays:  p.DurationDays,
			PriceCents:    p.PriceCents,
			RuleAI:        p.RuleAI,
			RuleStreaming: p.RuleStreaming,
			RuleChina:     p.RuleChina,
			RuleAdBlock:   p.RuleAdBlock,
			AutoReset:     p.AutoReset,
		}
	}
	return ClientPlan{
		Name:          "Custom",
		TrafficLimit:  u.TrafficLimit,
		RuleAI:        u.RuleAI,
		RuleStreaming: u.RuleStreaming,
		RuleChina:     u.RuleChina,
		RuleAdBlock:   u.RuleAdBlock,
		AutoReset:     u.AutoReset,
	}
}

func BuildClientTrafficSummary(u model.User) ClientTrafficSummary {
	total := u.TrafficUsed
	upload := total / 2
	download := total - upload
	speedTX := int64(0)
	speedRX := int64(0)
	if s, ok := GetUserStat(u.Username); ok {
		liveTotal := s.TotalTX + s.TotalRX
		if liveTotal > total {
			total = liveTotal
			upload = s.TotalTX
			download = s.TotalRX
		}
		speedTX = s.SpeedTX
		speedRX = s.SpeedRX
	}
	percent := 0
	if u.TrafficLimit > 0 {
		percent = int(float64(total) * 100 / float64(u.TrafficLimit))
	}
	expireUnix := int64(0)
	if !u.ExpiresAt.IsZero() {
		expireUnix = u.ExpiresAt.Unix()
	}
	return ClientTrafficSummary{
		TrafficLimit: u.TrafficLimit,
		TrafficUsed:  u.TrafficUsed,
		Upload:       upload,
		Download:     download,
		Total:        total,
		SpeedTX:      speedTX,
		SpeedRX:      speedRX,
		PercentUsed:  percent,
		ExpiresAt:    u.ExpiresAt,
		ExpireUnix:   expireUnix,
	}
}

func BuildClientNodes(nodes []model.Node, probeStates map[uint]model.NodeProbeState) []ClientNode {
	out := make([]ClientNode, 0, len(nodes))
	for _, n := range nodes {
		item := ClientNode{
			ID:           n.ID,
			Name:         n.Name,
			Host:         n.Host,
			Port:         n.Port,
			SNI:          n.SNI,
			Insecure:     n.Insecure,
			ObfsType:     n.ObfsType,
			ObfsPassword: n.ObfsPassword,
			Healthy:      n.Healthy,
		}
		if state, ok := probeStates[n.ID]; ok {
			item.ProbeStatus = state.Status
			item.LastLatencyMS = state.LastLatencyMS
			item.LastCheckedAt = state.LastCheckedAt
			item.LastError = state.LastError
		}
		out = append(out, item)
	}
	return out
}

func BuildClientNodeTraffic(username string) []ClientNodeTraffic {
	stat, ok := GetUserStat(username)
	if !ok || len(stat.PerNode) == 0 {
		return []ClientNodeTraffic{}
	}
	out := make([]ClientNodeTraffic, 0, len(stat.PerNode))
	nodes := make([]string, 0, len(stat.PerNode))
	for node := range stat.PerNode {
		nodes = append(nodes, node)
	}
	sort.Strings(nodes)
	for _, node := range nodes {
		t := stat.PerNode[node]
		out = append(out, ClientNodeTraffic{
			Node:     node,
			Upload:   t.TX,
			Download: t.RX,
			Total:    t.TX + t.RX,
		})
	}
	return out
}

func BuildClientTrafficHistory(logs []model.TrafficLog) []ClientTrafficHistoryEntry {
	out := make([]ClientTrafficHistoryEntry, 0, len(logs))
	for _, log := range logs {
		out = append(out, ClientTrafficHistoryEntry{
			SampledAt: log.SampledAt,
			NodeID:    log.NodeID,
			TX:        log.TX,
			RX:        log.RX,
		})
	}
	return out
}

func BuildClientConfig(u model.User, nodes []model.Node, probeStates map[uint]model.NodeProbeState) ClientConfig {
	clientNodes := BuildClientNodes(nodes, probeStates)
	defaultNodeID := uint(0)
	if len(clientNodes) > 0 {
		defaultNodeID = clientNodes[0].ID
	}
	return ClientConfig{
		Protocol:      "hysteria2",
		Auth:          u.Hy2Password,
		DefaultNodeID: defaultNodeID,
		Nodes:         clientNodes,
		TUN: ClientConfigTun{
			Enabled: true,
			Mode:    "tun",
		},
		Rules: ClientConfigRules{
			AI:        u.RuleAI || u.ChainProxy,
			Streaming: u.RuleStreaming,
			China:     u.RuleChina,
			AdBlock:   u.RuleAdBlock,
		},
	}
}

func BuildClientAnnouncements(now time.Time) []ClientAnnouncement {
	return []ClientAnnouncement{
		{
			ID:        "welcome",
			Severity:  "info",
			Title:     "欢迎使用 selfladder",
			Body:      "客户端当前处于早期版本，连接问题可先使用诊断信息排查。",
			UpdatedAt: now,
		},
	}
}

func BuildClientHelp() []ClientHelpItem {
	return []ClientHelpItem{
		{ID: "udp_blocked", Title: "无法连接节点", Body: "请确认当前网络没有屏蔽 UDP，并尝试切换节点。"},
		{ID: "traffic_exceeded", Title: "流量已用尽", Body: "账号超过流量限制后将无法建立 Hy2 连接。"},
		{ID: "plan_expired", Title: "套餐已到期", Body: "账号到期后客户端会显示不可连接状态。"},
	}
}

func BuildClientDiagnostics(u model.User, availableNodes int, unavailableNodes int) ClientDiagnostics {
	reasons := make([]string, 0)
	warnings := make([]string, 0)
	accountDisabled := !u.Enabled
	planExpired := u.IsExpired()
	trafficExceeded := u.TrafficExceeded()
	if accountDisabled {
		reasons = append(reasons, "account_disabled")
	}
	if planExpired {
		reasons = append(reasons, "plan_expired")
	}
	if trafficExceeded {
		reasons = append(reasons, "traffic_exceeded")
	}
	if availableNodes == 0 {
		reasons = append(reasons, "no_nodes")
	}
	if unavailableNodes > 0 {
		warnings = append(warnings, "node_unhealthy")
	}
	return ClientDiagnostics{
		CanConnect:       len(reasons) == 0,
		ReasonCodes:      reasons,
		Warnings:         warnings,
		AccountActive:    u.IsActive(),
		AvailableNodes:   availableNodes,
		UnavailableNodes: unavailableNodes,
		TrafficExceeded:  trafficExceeded,
		PlanExpired:      planExpired,
		AccountDisabled:  accountDisabled,
	}
}
