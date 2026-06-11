package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/net/proxy"

	"github.com/ludandaye/hy2board/internal/database"
	"github.com/ludandaye/hy2board/internal/model"
)

// StartStaticIPProber probes each IP-bearing Plan every interval.
// Health flips to false only after 2 consecutive failures (debounce).
func StartStaticIPProber(interval time.Duration) {
	tick := func() {
		var plans []model.Plan
		database.DB.Where("proxy_host <> ''").Find(&plans)
		if len(plans) == 0 {
			return
		}

		sem := make(chan struct{}, 8)
		var wg sync.WaitGroup
		for _, p := range plans {
			p := p
			wg.Add(1)
			sem <- struct{}{}
			go func() {
				defer wg.Done()
				defer func() { <-sem }()
				probeStaticIP(p)
			}()
		}
		wg.Wait()
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

// ProbeOnce runs a single probe for a specific plan asynchronously. Call after
// Plan create/update so the UI shows healthy state within seconds instead of
// waiting up to interval for the next tick.
func ProbeOnce(planID uint) {
	go func() {
		var p model.Plan
		if err := database.DB.First(&p, planID).Error; err != nil {
			return
		}
		if p.ProxyHost == "" {
			return
		}
		probeStaticIP(p)
	}()
}

// probeStaticIP performs a SOCKS5 handshake + GET https://api.ipify.org
// through the proxy. Updates the in-memory health cache.
func probeStaticIP(p model.Plan) {
	h := getStaticIPHealthRaw(p.ID)
	h.LastProbedAt = time.Now()

	if p.ProxyType != "socks5" {
		h.Healthy = false
		h.LastExitIP = "unsupported proxy_type: " + p.ProxyType
		return
	}

	addr := fmt.Sprintf("%s:%d", p.ProxyHost, p.ProxyPort)
	var auth *proxy.Auth
	if p.ProxyUsername != "" {
		auth = &proxy.Auth{User: p.ProxyUsername, Password: p.ProxyPassword}
	}

	dialer, err := proxy.SOCKS5("tcp", addr, auth, &net.Dialer{Timeout: 5 * time.Second})
	if err != nil {
		recordFailure(h, fmt.Sprintf("socks5 dialer error: %v", err))
		return
	}

	transport := &http.Transport{
		Dial:                  dialer.Dial,
		ResponseHeaderTimeout: 5 * time.Second,
	}
	client := &http.Client{Transport: transport, Timeout: 10 * time.Second}

	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.ipify.org?format=json", nil)
	resp, err := client.Do(req)
	rtt := int(time.Since(start) / time.Millisecond)
	if err != nil {
		recordFailure(h, fmt.Sprintf("ipify req: %v", err))
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var ipResp struct {
		IP string `json:"ip"`
	}
	_ = json.Unmarshal(body, &ipResp)

	h.Healthy = true
	h.FailStreak = 0
	h.LastRTTms = rtt
	h.LastExitIP = ipResp.IP
	log.Printf("static-ip probe: plan=%s host=%s healthy rtt=%dms exit=%s",
		p.Name, p.ProxyHost, rtt, ipResp.IP)
}

func recordFailure(h *StaticIPHealth, reason string) {
	h.FailStreak++
	h.LastExitIP = ""
	h.LastRTTms = 0
	if h.FailStreak >= 2 {
		h.Healthy = false
	}
	log.Printf("static-ip probe: failure streak=%d reason=%s", h.FailStreak, reason)
}
