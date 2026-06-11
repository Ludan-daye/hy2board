package service

import (
	"sync"
	"time"

	"github.com/ludandaye/hy2board/internal/config"
	"github.com/ludandaye/hy2board/internal/model"
)

// EffectiveProxyChain returns the proxy-chain config a user should actually use
// for AI chain proxies. Priority:
//  1. Per-user override (User.Proxy* fields)
//  2. Global config (config.C.ProxyChain)
//  3. Empty + false if neither is configured.
func EffectiveProxyChain(u model.User) (config.ProxyChainConfig, bool) {
	if u.ProxyHost != "" && u.ProxyPort > 0 {
		return config.ProxyChainConfig{
			Type:     u.ProxyType,
			Host:     u.ProxyHost,
			Port:     u.ProxyPort,
			Username: u.ProxyUsername,
			Password: u.ProxyPassword,
		}, true
	}
	if config.C.HasProxyChain() {
		return config.C.ProxyChain, true
	}
	return config.ProxyChainConfig{}, false
}

// StaticIPHealth holds the most recent probe result for one Plan's proxy.
type StaticIPHealth struct {
	Healthy      bool
	LastProbedAt time.Time
	LastRTTms    int
	LastExitIP   string
	FailStreak   int // internal: consecutive failure count for debounce
}

var (
	staticIPHealthMu    sync.RWMutex
	staticIPHealthCache = make(map[uint]*StaticIPHealth)
)

// GetStaticIPHealth returns the cached health for a Plan's proxy, if any.
func GetStaticIPHealth(planID uint) (StaticIPHealth, bool) {
	staticIPHealthMu.RLock()
	defer staticIPHealthMu.RUnlock()
	h, ok := staticIPHealthCache[planID]
	if !ok {
		return StaticIPHealth{}, false
	}
	return *h, true
}

// getStaticIPHealthRaw returns a pointer for in-place updates; used by the prober only.
func getStaticIPHealthRaw(planID uint) *StaticIPHealth {
	staticIPHealthMu.Lock()
	defer staticIPHealthMu.Unlock()
	h, ok := staticIPHealthCache[planID]
	if !ok {
		h = &StaticIPHealth{}
		staticIPHealthCache[planID] = h
	}
	return h
}
