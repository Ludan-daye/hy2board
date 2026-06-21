package service

import (
	"strings"
	"testing"

	"github.com/ludandaye/hy2board/internal/model"
)

func TestVlessClashSurgeAndURILines(t *testing.T) {
	u := model.User{Username: "alice"}
	n := model.Node{Name: "HK1-plain", Host: "38.47.108.14", VlessEnabled: true, VlessPort: 443,
		RealityPubkey: "PUB", RealityShortID: "ab12", RealitySNI: "www.microsoft.com"}

	uri := VlessURILine(u, n)
	if !strings.HasPrefix(uri, "vless://") || !strings.Contains(uri, "security=reality") ||
		!strings.Contains(uri, "pbk=PUB") || !strings.Contains(uri, "sni=www.microsoft.com") ||
		!strings.Contains(uri, "@38.47.108.14:443") {
		t.Fatalf("bad vless uri: %s", uri)
	}

	clash := VlessClashBlock(u, n)
	if !strings.Contains(clash, "type: vless") || !strings.Contains(clash, "public-key: PUB") ||
		!strings.Contains(clash, "servername: www.microsoft.com") {
		t.Fatalf("bad clash block: %s", clash)
	}

	surge := VlessSurgeLine(u, n)
	if !strings.Contains(surge, "vless,") || !strings.Contains(surge, "reality-pubkey=PUB") ||
		!strings.Contains(surge, "sni=www.microsoft.com") {
		t.Fatalf("bad surge line: %s", surge)
	}

	// the display name is suffixed -T so it never collides with the HY2 node
	if !strings.Contains(uri, "HK1-plain-T") {
		t.Fatalf("vless name should be suffixed -T: %s", uri)
	}
}

func TestVlessName(t *testing.T) {
	if VlessName(model.Node{Name: "HK1-plain"}) != "HK1-plain-T" {
		t.Fatalf("VlessName should append -T")
	}
}
