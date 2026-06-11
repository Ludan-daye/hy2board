package service

import (
	"strings"
	"testing"

	"github.com/ludandaye/hy2board/internal/config"
	"github.com/ludandaye/hy2board/internal/model"
)

func TestGeneratedRulesKeepSharedPriority(t *testing.T) {
	config.C = config.Config{}
	user := testRuleUser()
	nodes := testNodes()

	cases := map[string]string{
		"clash":        GenerateClash(user, nodes),
		"surge":        GenerateSurge(user, nodes),
		"shadowrocket": GenerateShadowrocket(user, nodes),
	}

	for name, got := range cases {
		assertOrdered(t, name, got, []string{
			"DOMAIN-SUFFIX,doubleclick.net,REJECT",
			"DOMAIN-SUFFIX,chatgpt.com,Auto",
			"DOMAIN-SUFFIX,netflix.com,Streaming",
			"DOMAIN-SUFFIX,github.com,Global",
			"DOMAIN-SUFFIX,qq.com,DIRECT",
			"GEOIP,CN,DIRECT",
		})
		if !strings.Contains(got, "DOMAIN-SUFFIX,telegram.org,Global") {
			t.Fatalf("%s rules missing telegram global rule:\n%s", name, got)
		}
		if !strings.Contains(got, "DOMAIN-SUFFIX,taobao.com,DIRECT") {
			t.Fatalf("%s rules missing taobao direct rule:\n%s", name, got)
		}
	}
}

func TestAIUsesChainWhenConfigured(t *testing.T) {
	config.C = config.Config{
		ProxyChain: config.ProxyChainConfig{
			Type:     "socks5",
			Host:     "127.0.0.1",
			Port:     1080,
			Username: "u",
			Password: "p",
		},
	}
	defer func() { config.C = config.Config{} }()

	got := GenerateClash(testRuleUser(), testNodes())
	for _, want := range []string{
		"DOMAIN-SUFFIX,chatgpt.com,AI",
		"  - name: \"JP-AI\"",
		"    dialer-proxy: JP",
		"  - name: AI",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected Clash config to contain %q:\n%s", want, got)
		}
	}
}

func TestAIFallsBackToAutoWithoutChain(t *testing.T) {
	config.C = config.Config{}

	got := GenerateClash(testRuleUser(), testNodes())
	if !strings.Contains(got, "DOMAIN-SUFFIX,chatgpt.com,Auto") {
		t.Fatalf("expected AI rule to fall back to Auto without chain:\n%s", got)
	}
	if strings.Contains(got, "  - name: AI\n") || strings.Contains(got, "-AI") {
		t.Fatalf("did not expect AI proxy group or chained proxies without chain config:\n%s", got)
	}
}

func TestGoogleScholarUsesHongKongGroupWhenHKNodesExist(t *testing.T) {
	config.C = config.Config{}
	user := testRuleUser()
	nodes := append(testNodes(), model.Node{
		Name:     "HK1-plain",
		Host:     "198.51.100.20",
		Port:     443,
		SNI:      "bing.com",
		Insecure: true,
	})

	cases := map[string]string{
		"clash":        GenerateClash(user, nodes),
		"surge":        GenerateSurge(user, nodes),
		"shadowrocket": GenerateShadowrocket(user, nodes),
	}

	for name, got := range cases {
		if !strings.Contains(got, "DOMAIN-SUFFIX,scholar.google.com,HongKong") {
			t.Fatalf("%s config should route Google Scholar to HongKong:\n%s", name, got)
		}
		if !strings.Contains(got, "HongKong") || !strings.Contains(got, "HK1-plain") {
			t.Fatalf("%s config should contain a HongKong group using the HK node:\n%s", name, got)
		}
	}
}

func TestGoogleScholarFallsBackToAutoWithoutHKNodes(t *testing.T) {
	config.C = config.Config{}
	user := testRuleUser()

	cases := map[string]string{
		"clash":        GenerateClash(user, testNodes()),
		"surge":        GenerateSurge(user, testNodes()),
		"shadowrocket": GenerateShadowrocket(user, testNodes()),
	}

	for name, got := range cases {
		if !strings.Contains(got, "DOMAIN-SUFFIX,scholar.google.com,Auto") {
			t.Fatalf("%s config should route Google Scholar to Auto without HK nodes:\n%s", name, got)
		}
		if strings.Contains(got, "HongKong =") || strings.Contains(got, "  - name: HongKong\n") {
			t.Fatalf("%s config should not create a HongKong group without HK nodes:\n%s", name, got)
		}
	}
}

func TestCustomRoutingRulesAreInsertedAfterRejectRules(t *testing.T) {
	config.C = config.Config{}
	user := testRuleUser()
	nodes := append(testNodes(), model.Node{
		Name:     "HK1-plain",
		Host:     "198.51.100.20",
		Port:     443,
		SNI:      "bing.com",
		Insecure: true,
	})
	custom := []RoutingRule{
		{Kind: RuleDomainSuffix, Value: "example.com", Policy: GroupHongKong},
		{Kind: RuleIPCIDR, Value: "203.0.113.0/24", Policy: GroupReject},
		{Kind: RuleGeoIP, Value: "US", Policy: GroupGlobal},
	}

	cases := map[string]string{
		"clash":        GenerateClashWithCustomRules(user, nodes, custom),
		"surge":        GenerateSurgeWithCustomRules(user, nodes, custom),
		"shadowrocket": GenerateShadowrocketWithCustomRules(user, nodes, custom),
	}

	for name, got := range cases {
		assertOrdered(t, name, got, []string{
			"DOMAIN-SUFFIX,doubleclick.net,REJECT",
			"DOMAIN-SUFFIX,example.com,HongKong",
			"IP-CIDR,203.0.113.0/24,REJECT",
			"GEOIP,US,Global",
			"DOMAIN-SUFFIX,chatgpt.com,Auto",
			"DOMAIN-SUFFIX,github.com,Global",
			"DOMAIN-SUFFIX,qq.com,DIRECT",
		})
	}
}

func TestCustomRoutingRulePoliciesFallbackWhenGroupsUnavailable(t *testing.T) {
	config.C = config.Config{}
	user := testRuleUser()
	custom := []RoutingRule{
		{Kind: RuleDomainSuffix, Value: "hk-only.example", Policy: GroupHongKong},
		{Kind: RuleDomainSuffix, Value: "ai-only.example", Policy: GroupAI},
	}

	got := GenerateClashWithCustomRules(user, testNodes(), custom)
	for _, want := range []string{
		"DOMAIN-SUFFIX,hk-only.example,Auto",
		"DOMAIN-SUFFIX,ai-only.example,Auto",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected fallback rule %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "  - name: HongKong\n") || strings.Contains(got, "  - name: AI\n") {
		t.Fatalf("did not expect unavailable policy groups:\n%s", got)
	}
}

func TestCustomAIRuleCreatesAIGroupWhenChainConfigured(t *testing.T) {
	config.C = config.Config{
		ProxyChain: config.ProxyChainConfig{
			Type:     "socks5",
			Host:     "127.0.0.1",
			Port:     1080,
			Username: "u",
			Password: "p",
		},
	}
	defer func() { config.C = config.Config{} }()

	user := testRuleUser()
	user.RuleAI = false
	user.ChainProxy = false
	custom := []RoutingRule{{Kind: RuleDomainSuffix, Value: "ai-only.example", Policy: GroupAI}}

	got := GenerateClashWithCustomRules(user, testNodes(), custom)
	for _, want := range []string{
		"DOMAIN-SUFFIX,ai-only.example,AI",
		"  - name: \"JP-AI\"",
		"  - name: AI",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected custom AI rule to contain %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "chatgpt.com") {
		t.Fatalf("custom AI policy should not enable built-in AI domains:\n%s", got)
	}
}

func TestAIFlagDisabledDoesNotGenerateAIRules(t *testing.T) {
	config.C = config.Config{
		ProxyChain: config.ProxyChainConfig{
			Type: "socks5",
			Host: "127.0.0.1",
			Port: 1080,
		},
	}
	defer func() { config.C = config.Config{} }()

	user := testRuleUser()
	user.RuleAI = false
	user.ChainProxy = false

	got := GenerateClash(user, testNodes())
	if strings.Contains(got, "chatgpt.com") || strings.Contains(got, "  - name: AI\n") || strings.Contains(got, "-AI") {
		t.Fatalf("did not expect AI rules or chain proxies when AI is disabled:\n%s", got)
	}
}

func TestShadowrocketGeneratesFullConfAndURIStaysPlain(t *testing.T) {
	config.C = config.Config{}
	user := testRuleUser()
	nodes := testNodes()

	conf := GenerateShadowrocket(user, nodes)
	for _, want := range []string{
		"[General]",
		"[Proxy]",
		"JP=hysteria2,203.0.113.10,443,auth=secret,udp=1,peer=bing.com,alpn=h3,skip-cert-verify=true,insecure=1",
		"[Proxy Group]",
		"[Rule]",
		"FINAL,Auto",
	} {
		if !strings.Contains(conf, want) {
			t.Fatalf("expected Shadowrocket conf to contain %q:\n%s", want, conf)
		}
	}
	for _, unexpected := range []string{" = hysteria2", "obfs-password", "obfs=salamander"} {
		if strings.Contains(conf, unexpected) {
			t.Fatalf("expected Shadowrocket conf to omit %q:\n%s", unexpected, conf)
		}
	}

	uri := GenerateURI(user, nodes)
	if !strings.HasPrefix(uri, "hysteria2://secret@203.0.113.10:443") {
		t.Fatalf("expected URI mode to remain a plain hysteria2 URI, got %q", uri)
	}
}

func TestProxyTestURLAvoidsGstaticForGeneratedConfigs(t *testing.T) {
	config.C = config.Config{}
	user := testRuleUser()
	nodes := testNodes()

	for name, got := range map[string]string{
		"clash":        GenerateClash(user, nodes),
		"surge":        GenerateSurge(user, nodes),
		"shadowrocket": GenerateShadowrocket(user, nodes),
	} {
		if strings.Contains(got, "www.gstatic.com/generate_204") {
			t.Fatalf("%s config still uses gstatic test URL:\n%s", name, got)
		}
		if !strings.Contains(got, proxyTestURL) {
			t.Fatalf("%s config missing proxy test URL %q:\n%s", name, proxyTestURL, got)
		}
	}
}

func TestV2RayNURIGetsHy2CompatibilityParams(t *testing.T) {
	config.C = config.Config{}

	got := GenerateV2RayNURI(testRuleUser(), testNodes())
	for _, want := range []string{
		"hy2://secret@203.0.113.10:443",
		"allowInsecure=1",
		"insecure=1",
		"sni=bing.com",
		"#JP",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected v2rayN URI to contain %q:\n%s", want, got)
		}
	}
}

func assertOrdered(t *testing.T, name string, got string, wants []string) {
	t.Helper()
	last := -1
	for _, want := range wants {
		idx := strings.Index(got, want)
		if idx < 0 {
			t.Fatalf("%s output missing %q:\n%s", name, want, got)
		}
		if idx < last {
			t.Fatalf("%s output has %q out of order:\n%s", name, want, got)
		}
		last = idx
	}
}

func testRuleUser() model.User {
	return model.User{
		Username:      "alice",
		Hy2Password:   "secret",
		RuleAI:        true,
		RuleStreaming: true,
		RuleChina:     true,
		RuleAdBlock:   true,
	}
}

func testNodes() []model.Node {
	return []model.Node{
		{
			Name:     "JP",
			Host:     "203.0.113.10",
			Port:     443,
			SNI:      "bing.com",
			Insecure: true,
		},
	}
}
