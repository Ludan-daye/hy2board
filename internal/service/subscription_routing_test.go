package service

import (
	"strings"
	"testing"

	"github.com/ludandaye/hy2board/internal/config"
	"github.com/ludandaye/hy2board/internal/model"
)

func routingNodes() []model.Node {
	return []model.Node{{Name: "SG1", Host: "9.9.9.9", Port: 443, SNI: "bing.com", Insecure: true}}
}

func TestStaticVsNonStaticRouting(t *testing.T) {
	config.C.ProxyChain = config.ProxyChainConfig{Type: "socks5", Host: "1.1.1.1", Port: 8022, Username: "us", Password: "up"}
	config.C.ProxyChainDefault = config.ProxyChainConfig{Type: "socks5", Host: "2.2.2.2", Port: 8022, Username: "tw", Password: "tp"}

	// non-static user with rule_ai ON must get NO -AI leak, no -TW
	non := model.User{Hy2Password: "p", RuleAI: true, RuleChina: true, ChainProxy: false}
	c := GenerateClash(non, routingNodes())
	if strings.Contains(c, "SG1-AI") || strings.Contains(c, "1.1.1.1") {
		t.Fatal("non-static user leaked the US -AI proxy")
	}
	if strings.Contains(c, "SG1-TW") {
		t.Fatal("non-static user should have no -TW")
	}

	// static user: US -AI present + Taiwan -TW present + Auto over -TW
	st := model.User{Hy2Password: "p", RuleAI: true, RuleChina: true, ChainProxy: true}
	c2 := GenerateClash(st, routingNodes())
	if !strings.Contains(c2, "SG1-AI") || !strings.Contains(c2, "1.1.1.1") {
		t.Fatal("static user missing US AI chain")
	}
	if !strings.Contains(c2, "SG1-TW") || !strings.Contains(c2, "2.2.2.2") {
		t.Fatal("static user missing Taiwan default chain")
	}
	// Auto group should list the -TW node (Taiwan pool), not raw SG1
	autoIdx := strings.Index(c2, "name: "+GroupAuto)
	if autoIdx < 0 {
		t.Fatal("no Auto group")
	}
	seg := c2[autoIdx : autoIdx+200]
	if !strings.Contains(seg, "SG1-TW") {
		t.Fatalf("Auto group should use Taiwan pool: %s", seg)
	}
}
