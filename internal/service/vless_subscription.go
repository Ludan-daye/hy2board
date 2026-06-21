package service

import (
	"fmt"
	"net/url"

	"github.com/ludandaye/hy2board/internal/model"
	"github.com/ludandaye/hy2board/internal/util"
)

// VlessName is the display name of a node's VLESS entry. The "-T" (TCP) suffix
// keeps it distinct from the HY2 proxy of the same node name.
func VlessName(n model.Node) string { return n.Name + "-T" }

// NodeHasVless reports whether a node is configured to emit a VLESS line.
func NodeHasVless(n model.Node) bool {
	return n.VlessEnabled && n.VlessPort > 0 && n.RealityPubkey != ""
}

// VlessURILine renders a vless://...reality URI (v2rayN / Shadowrocket / generic).
// Built via net/url so node name / SNI / keys are correctly escaped.
func VlessURILine(u model.User, n model.Node) string {
	uri := url.URL{
		Scheme: "vless",
		User:   url.User(util.VlessUUID(u.Username)),
		Host:   fmt.Sprintf("%s:%d", n.Host, n.VlessPort),
	}
	q := uri.Query()
	q.Set("encryption", "none")
	q.Set("security", "reality")
	q.Set("sni", n.RealitySNI)
	q.Set("fp", "chrome")
	q.Set("pbk", n.RealityPubkey)
	q.Set("sid", n.RealityShortID)
	q.Set("flow", "xtls-rprx-vision")
	q.Set("type", "tcp")
	uri.RawQuery = q.Encode()
	uri.Fragment = VlessName(n)
	return uri.String()
}

// VlessClashBlock renders a Clash/mihomo proxy entry (YAML list item).
func VlessClashBlock(u model.User, n model.Node) string {
	uuid := util.VlessUUID(u.Username)
	// String values are quoted so an all-digit short-id isn't parsed as a YAML
	// number and a leading-"-" public-key isn't mis-parsed.
	return fmt.Sprintf(`  - name: "%s"
    type: vless
    server: %s
    port: %d
    uuid: "%s"
    network: tcp
    tls: true
    udp: true
    flow: xtls-rprx-vision
    servername: "%s"
    client-fingerprint: chrome
    reality-opts:
      public-key: "%s"
      short-id: "%s"`, VlessName(n), n.Host, n.VlessPort, uuid, n.RealitySNI, n.RealityPubkey, n.RealityShortID)
}

// VlessSurgeLine renders a Surge proxy line.
func VlessSurgeLine(u model.User, n model.Node) string {
	uuid := util.VlessUUID(u.Username)
	return fmt.Sprintf("%s = vless, %s, %d, username=%s, tls=true, sni=%s, reality-pubkey=%s, reality-short-id=%s, flow=xtls-rprx-vision",
		VlessName(n), n.Host, n.VlessPort, uuid, n.RealitySNI, n.RealityPubkey, n.RealityShortID)
}
