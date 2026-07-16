package service

import (
	"testing"

	"github.com/ludandaye/hy2board/internal/config"
	"github.com/ludandaye/hy2board/internal/model"
)

func TestEffectiveChains(t *testing.T) {
	config.C.ProxyChain = config.ProxyChainConfig{Type: "socks5", Host: "1.1.1.1", Port: 8022}        // US/AI
	config.C.ProxyChainDefault = config.ProxyChainConfig{Type: "socks5", Host: "2.2.2.2", Port: 8022} // Taiwan

	// non-static user: no chains at all
	u := model.User{ChainProxy: false}
	if _, ok := EffectiveAIChain(u); ok {
		t.Fatal("non-static must have no AI chain")
	}
	if _, ok := EffectiveDefaultChain(u); ok {
		t.Fatal("non-static must have no default chain")
	}

	// static user: AI=US global, default=Taiwan global
	s := model.User{ChainProxy: true}
	if ai, ok := EffectiveAIChain(s); !ok || ai.Host != "1.1.1.1" {
		t.Fatalf("static AI want 1.1.1.1 got %v ok=%v", ai.Host, ok)
	}
	if d, ok := EffectiveDefaultChain(s); !ok || d.Host != "2.2.2.2" {
		t.Fatalf("static default want 2.2.2.2 got %v ok=%v", d.Host, ok)
	}

	// static user with per-user ProxyHost override -> default uses the override, AI unchanged
	o := model.User{ChainProxy: true, ProxyType: "socks5", ProxyHost: "9.9.9.9", ProxyPort: 1080}
	if d, ok := EffectiveDefaultChain(o); !ok || d.Host != "9.9.9.9" {
		t.Fatalf("override default want 9.9.9.9 got %v", d.Host)
	}
	if ai, _ := EffectiveAIChain(o); ai.Host != "1.1.1.1" {
		t.Fatal("override must not change AI chain")
	}
}
