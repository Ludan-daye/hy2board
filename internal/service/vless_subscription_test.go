package service

import (
	"strings"
	"testing"

	"github.com/ludandaye/hy2board/internal/model"
)

func TestVlessClashSurgeAndURILines(t *testing.T) {
	u := model.User{Username: "alice"}
	// all-digit short-id + leading-"-" pubkey: both must survive YAML/URI encoding.
	n := model.Node{Name: "HK1-plain", Host: "38.47.108.14", VlessEnabled: true, VlessPort: 443,
		RealityPubkey: "-PUBkey", RealityShortID: "07204836", RealitySNI: "www.microsoft.com"}

	uri := VlessURILine(u, n)
	if !strings.HasPrefix(uri, "vless://") || !strings.Contains(uri, "security=reality") ||
		!strings.Contains(uri, "sid=07204836") || !strings.Contains(uri, "sni=www.microsoft.com") ||
		!strings.Contains(uri, "@38.47.108.14:443") {
		t.Fatalf("bad vless uri: %s", uri)
	}

	clash := VlessClashBlock(u, n)
	// short-id/public-key/servername MUST be quoted: an all-digit short-id would
	// otherwise parse as a YAML number, and a leading-"-" pubkey could mis-parse.
	if !strings.Contains(clash, "type: vless") ||
		!strings.Contains(clash, `short-id: "07204836"`) ||
		!strings.Contains(clash, `public-key: "-PUBkey"`) ||
		!strings.Contains(clash, `servername: "www.microsoft.com"`) {
		t.Fatalf("bad clash block: %s", clash)
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
