package service

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/ludandaye/hy2board/internal/model"
)

const proxyTestURL = "http://cp.cloudflare.com/generate_204"

func GenerateSurge(user model.User, nodes []model.Node) string {
	return GenerateSurgeWithCustomRules(user, nodes, nil)
}

func GenerateSurgeWithCustomRules(user model.User, nodes []model.Node, customRules []RoutingRule) string {
	var lines []string
	pc, hasChainConfig := EffectiveProxyChain(user)
	rs := BuildRuleSet(user, customRules...)
	hasChain := rs.HasAI && hasChainConfig
	aiPolicy := GroupAuto
	if hasChain {
		aiPolicy = GroupAI
	}
	hongKongNames := hongKongNodeNames(nodes)
	hongKongPolicy := GroupAuto
	if len(hongKongNames) > 0 {
		hongKongPolicy = GroupHongKong
	}

	lines = append(lines, "[General]")
	lines = append(lines, "loglevel = notify")
	lines = append(lines, "skip-proxy = 127.0.0.1, 192.168.0.0/16, 10.0.0.0/8, 172.16.0.0/12, localhost, *.local")
	lines = append(lines, "proxy-test-url = "+proxyTestURL)
	lines = append(lines, "test-timeout = 3")
	lines = append(lines, "")

	lines = append(lines, "[Proxy]")
	names := make([]string, len(nodes))
	chainNames := make([]string, 0)
	vlessNames := make([]string, 0)

	for i, n := range nodes {
		proxy := fmt.Sprintf("%s = hysteria2, %s, %d, password=%s, skip-cert-verify=%t, sni=%s",
			n.Name, n.Host, n.Port, user.Hy2Password, n.Insecure, n.SNI)
		if n.ObfsType != "" {
			proxy += fmt.Sprintf(", obfs=%s, obfs-password=%s", n.ObfsType, n.ObfsPassword)
		}
		lines = append(lines, proxy)
		names[i] = n.Name
		// VLESS not emitted for Surge: Surge has no `vless` proxy type (it would
		// error "Unknown proxy type: vless" and break the whole config). Surge users
		// get HY2 only; the VLESS fallback is delivered via the Clash/URI formats.

		if hasChain {
			chainName := n.Name + "-AI"
			chainProxy := fmt.Sprintf("%s = %s, %s, %d, username=%s, password=%s, underlying-proxy=%s",
				chainName, pc.Type, pc.Host, pc.Port, pc.Username, pc.Password, n.Name)
			lines = append(lines, chainProxy)
			chainNames = append(chainNames, chainName)
		}
	}
	lines = append(lines, "")

	names = append(names, vlessNames...)
	lines = append(lines, "[Proxy Group]")
	lines = append(lines, fmt.Sprintf("%s = url-test, %s, url=%s, interval=300", GroupAuto, strings.Join(names, ", "), proxyTestURL))
	lines = append(lines, fmt.Sprintf("%s = select, %s", GroupManual, strings.Join(names, ", ")))
	if len(hongKongNames) > 0 {
		lines = append(lines, fmt.Sprintf("%s = url-test, %s, url=%s, interval=300", GroupHongKong, strings.Join(hongKongNames, ", "), proxyTestURL))
	}
	if rs.HasAI && hasChain {
		lines = append(lines, fmt.Sprintf("%s = url-test, %s, url=%s, interval=300", GroupAI, strings.Join(chainNames, ", "), proxyTestURL))
	}
	if rs.HasStreaming {
		lines = append(lines, fmt.Sprintf("%s = url-test, %s, url=%s, interval=300", GroupStreaming, strings.Join(names, ", "), proxyTestURL))
	}
	if rs.HasGlobal {
		lines = append(lines, fmt.Sprintf("%s = url-test, %s, url=%s, interval=300", GroupGlobal, strings.Join(names, ", "), proxyTestURL))
	}
	lines = append(lines, "")

	lines = append(lines, "[Rule]")
	for _, rule := range rs.Lines(aiPolicy, hongKongPolicy) {
		lines = append(lines, rule)
	}
	lines = append(lines, "FINAL,"+GroupAuto)

	return strings.Join(lines, "\n")
}

func GenerateClash(user model.User, nodes []model.Node) string {
	return GenerateClashWithCustomRules(user, nodes, nil)
}

func GenerateClashWithCustomRules(user model.User, nodes []model.Node, customRules []RoutingRule) string {
	var lines []string
	pc, hasChainConfig := EffectiveProxyChain(user)
	rs := BuildRuleSet(user, customRules...)
	hasChain := rs.HasAI && hasChainConfig
	aiPolicy := GroupAuto
	if hasChain {
		aiPolicy = GroupAI
	}
	hongKongNames := hongKongNodeNames(nodes)
	hongKongPolicy := GroupAuto
	if len(hongKongNames) > 0 {
		hongKongPolicy = GroupHongKong
	}

	lines = append(lines, "mixed-port: 7890")
	lines = append(lines, "allow-lan: false")
	lines = append(lines, "mode: rule")
	lines = append(lines, "log-level: info")
	lines = append(lines, "")
	lines = append(lines, "proxies:")

	names := make([]string, len(nodes))
	chainNames := make([]string, 0)
	vlessNames := make([]string, 0)

	for i, n := range nodes {
		lines = append(lines, fmt.Sprintf("  - name: \"%s\"", n.Name))
		lines = append(lines, "    type: hysteria2")
		lines = append(lines, fmt.Sprintf("    server: %s", n.Host))
		lines = append(lines, fmt.Sprintf("    port: %d", n.Port))
		lines = append(lines, fmt.Sprintf("    password: \"%s\"", user.Hy2Password))
		lines = append(lines, fmt.Sprintf("    sni: %s", n.SNI))
		lines = append(lines, fmt.Sprintf("    skip-cert-verify: %t", n.Insecure))
		if n.ObfsType != "" {
			lines = append(lines, fmt.Sprintf("    obfs: %s", n.ObfsType))
			lines = append(lines, fmt.Sprintf("    obfs-password: %s", n.ObfsPassword))
		}
		lines = append(lines, "")
		names[i] = n.Name
		if NodeHasVless(n) {
			lines = append(lines, VlessClashBlock(user, n))
			lines = append(lines, "")
			vlessNames = append(vlessNames, VlessName(n))
		}

		if hasChain {
			chainName := n.Name + "-AI"
			lines = append(lines, fmt.Sprintf("  - name: \"%s\"", chainName))
			lines = append(lines, fmt.Sprintf("    type: %s", pc.Type))
			lines = append(lines, fmt.Sprintf("    server: %s", pc.Host))
			lines = append(lines, fmt.Sprintf("    port: %d", pc.Port))
			lines = append(lines, fmt.Sprintf("    username: %s", pc.Username))
			lines = append(lines, fmt.Sprintf("    password: %s", pc.Password))
			lines = append(lines, fmt.Sprintf("    dialer-proxy: %s", n.Name))
			lines = append(lines, "")
			chainNames = append(chainNames, chainName)
		}
	}

	names = append(names, vlessNames...)
	lines = append(lines, "proxy-groups:")
	lines = append(lines, "  - name: "+GroupAuto)
	lines = append(lines, "    type: url-test")
	lines = append(lines, "    proxies:")
	for _, name := range names {
		lines = append(lines, fmt.Sprintf("      - \"%s\"", name))
	}
	lines = append(lines, "    url: "+proxyTestURL)
	lines = append(lines, "    interval: 300")

	lines = append(lines, "  - name: "+GroupManual)
	lines = append(lines, "    type: select")
	lines = append(lines, "    proxies:")
	for _, name := range names {
		lines = append(lines, fmt.Sprintf("      - \"%s\"", name))
	}

	if len(hongKongNames) > 0 {
		lines = append(lines, "  - name: "+GroupHongKong)
		lines = append(lines, "    type: url-test")
		lines = append(lines, "    proxies:")
		for _, name := range hongKongNames {
			lines = append(lines, fmt.Sprintf("      - \"%s\"", name))
		}
		lines = append(lines, "    url: "+proxyTestURL)
		lines = append(lines, "    interval: 300")
	}

	if rs.HasAI && hasChain {
		lines = append(lines, "  - name: "+GroupAI)
		lines = append(lines, "    type: url-test")
		lines = append(lines, "    proxies:")
		for _, name := range chainNames {
			lines = append(lines, fmt.Sprintf("      - \"%s\"", name))
		}
		lines = append(lines, "    url: "+proxyTestURL)
		lines = append(lines, "    interval: 300")
	}

	if rs.HasStreaming {
		lines = append(lines, "  - name: "+GroupStreaming)
		lines = append(lines, "    type: url-test")
		lines = append(lines, "    proxies:")
		for _, name := range names {
			lines = append(lines, fmt.Sprintf("      - \"%s\"", name))
		}
		lines = append(lines, "    url: "+proxyTestURL)
		lines = append(lines, "    interval: 300")
	}

	if rs.HasGlobal {
		lines = append(lines, "  - name: "+GroupGlobal)
		lines = append(lines, "    type: url-test")
		lines = append(lines, "    proxies:")
		for _, name := range names {
			lines = append(lines, fmt.Sprintf("      - \"%s\"", name))
		}
		lines = append(lines, "    url: "+proxyTestURL)
		lines = append(lines, "    interval: 300")
	}

	lines = append(lines, "")

	lines = append(lines, "rules:")
	for _, rule := range rs.Lines(aiPolicy, hongKongPolicy) {
		lines = append(lines, fmt.Sprintf("  - %s", rule))
	}
	lines = append(lines, "  - MATCH,"+GroupAuto)

	return strings.Join(lines, "\n")
}

func GenerateShadowrocket(user model.User, nodes []model.Node) string {
	return GenerateShadowrocketWithCustomRules(user, nodes, nil)
}

func GenerateShadowrocketNodes(user model.User, nodes []model.Node) string {
	lines := make([]string, 0, len(nodes))
	for _, n := range nodes {
		lines = append(lines, shadowrocketProxyLine(user, n))
	}
	return strings.Join(lines, "\n")
}

func GenerateShadowrocketWithCustomRules(user model.User, nodes []model.Node, customRules []RoutingRule) string {
	var lines []string
	pc, hasChainConfig := EffectiveProxyChain(user)
	rs := BuildRuleSet(user, customRules...)
	hasChain := rs.HasAI && hasChainConfig
	aiPolicy := GroupAuto
	if hasChain {
		aiPolicy = GroupAI
	}
	hongKongNames := hongKongNodeNames(nodes)
	hongKongPolicy := GroupAuto
	if len(hongKongNames) > 0 {
		hongKongPolicy = GroupHongKong
	}

	lines = append(lines, "[General]")
	lines = append(lines, "bypass-system = true")
	lines = append(lines, "skip-proxy = 127.0.0.1, 192.168.0.0/16, 10.0.0.0/8, 172.16.0.0/12, localhost, *.local")
	lines = append(lines, "proxy-test-url = "+proxyTestURL)
	lines = append(lines, "test-timeout = 3")
	lines = append(lines, "")

	lines = append(lines, "[Proxy]")
	names := make([]string, len(nodes))
	chainNames := make([]string, 0)
	vlessNames := make([]string, 0)
	for i, n := range nodes {
		lines = append(lines, shadowrocketProxyLine(user, n))
		names[i] = n.Name
		// VLESS not wired for Shadowrocket: its .conf uses key=value syntax, not
		// Surge's. Shadowrocket users get the VLESS node via the URI format instead.

		if hasChain {
			chainName := n.Name + "-AI"
			chainProxy := fmt.Sprintf("%s = %s, %s, %d, username=%s, password=%s, underlying-proxy=%s",
				chainName, pc.Type, pc.Host, pc.Port, pc.Username, pc.Password, n.Name)
			lines = append(lines, chainProxy)
			chainNames = append(chainNames, chainName)
		}
	}
	lines = append(lines, "")

	names = append(names, vlessNames...)
	lines = append(lines, "[Proxy Group]")
	lines = append(lines, fmt.Sprintf("%s = url-test, %s, url=%s, interval=300", GroupAuto, strings.Join(names, ", "), proxyTestURL))
	lines = append(lines, fmt.Sprintf("%s = select, %s", GroupManual, strings.Join(names, ", ")))
	if len(hongKongNames) > 0 {
		lines = append(lines, fmt.Sprintf("%s = url-test, %s, url=%s, interval=300", GroupHongKong, strings.Join(hongKongNames, ", "), proxyTestURL))
	}
	if rs.HasAI && hasChain {
		lines = append(lines, fmt.Sprintf("%s = url-test, %s, url=%s, interval=300", GroupAI, strings.Join(chainNames, ", "), proxyTestURL))
	}
	if rs.HasStreaming {
		lines = append(lines, fmt.Sprintf("%s = url-test, %s, url=%s, interval=300", GroupStreaming, strings.Join(names, ", "), proxyTestURL))
	}
	if rs.HasGlobal {
		lines = append(lines, fmt.Sprintf("%s = url-test, %s, url=%s, interval=300", GroupGlobal, strings.Join(names, ", "), proxyTestURL))
	}
	lines = append(lines, "")

	lines = append(lines, "[Rule]")
	for _, rule := range rs.Lines(aiPolicy, hongKongPolicy) {
		lines = append(lines, rule)
	}
	lines = append(lines, "FINAL,"+GroupAuto)

	return strings.Join(lines, "\n")
}

func shadowrocketProxyLine(user model.User, n model.Node) string {
	parts := []string{
		fmt.Sprintf("%s=hysteria2", n.Name),
		n.Host,
		fmt.Sprintf("%d", n.Port),
		"auth=" + user.Hy2Password,
	}
	if n.ObfsType != "" {
		parts = append(parts, "obfsParam="+n.ObfsPassword)
	}
	parts = append(parts, "udp=1")
	if n.SNI != "" {
		parts = append(parts, "peer="+n.SNI)
	}
	parts = append(parts, "alpn=h3")
	if n.Insecure {
		parts = append(parts, "skip-cert-verify=true", "insecure=1")
	}
	return strings.Join(parts, ",")
}

func GenerateURI(user model.User, nodes []model.Node) string {
	return generateURI(user, nodes, "hysteria2", false)
}

func GenerateV2RayNURI(user model.User, nodes []model.Node) string {
	return generateURI(user, nodes, "hy2", true)
}

func generateURI(user model.User, nodes []model.Node, scheme string, includeAllowInsecure bool) string {
	var uris []string
	for _, n := range nodes {
		u := url.URL{
			Scheme: scheme,
			User:   url.User(user.Hy2Password),
			Host:   fmt.Sprintf("%s:%d", n.Host, n.Port),
		}
		q := u.Query()
		q.Set("sni", n.SNI)
		if n.Insecure {
			q.Set("insecure", "1")
			if includeAllowInsecure {
				q.Set("allowInsecure", "1")
			}
		}
		if n.ObfsType != "" {
			q.Set("obfs", n.ObfsType)
			q.Set("obfs-password", n.ObfsPassword)
		}
		u.RawQuery = q.Encode()
		u.Fragment = n.Name
		uris = append(uris, u.String())
		if NodeHasVless(n) {
			uris = append(uris, VlessURILine(user, n))
		}
	}
	return strings.Join(uris, "\n")
}

func hongKongNodeNames(nodes []model.Node) []string {
	names := make([]string, 0)
	for _, n := range nodes {
		if IsHongKongNodeName(n.Name) {
			names = append(names, n.Name)
		}
	}
	return names
}

func IsHongKongNodeName(name string) bool {
	upper := strings.ToUpper(name)
	compact := strings.NewReplacer(" ", "", "-", "", "_", "").Replace(upper)
	return strings.Contains(name, "香港") ||
		strings.Contains(compact, "HONGKONG") ||
		strings.Contains(compact, "HK")
}
