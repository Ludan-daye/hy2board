package service

import (
	"strings"
	"testing"

	"github.com/ludandaye/hy2board/internal/model"
)

func TestTrojanSurgeLine(t *testing.T) {
	u := model.User{Username: "alice"}
	n := model.Node{Name: "HK1-plain", Host: "38.47.108.14", TrojanEnabled: true, TrojanPort: 8443, TrojanSNI: "www.apple.com"}
	if !NodeHasTrojan(n) {
		t.Fatal("NodeHasTrojan should be true")
	}
	line := TrojanSurgeLine(u, n)
	// Surge-valid `trojan` type (never `vless`); name suffixed -T; tls fields present.
	if !strings.HasPrefix(line, "HK1-plain-T = trojan,") ||
		!strings.Contains(line, "38.47.108.14, 8443") ||
		!strings.Contains(line, "sni=www.apple.com") ||
		!strings.Contains(line, "skip-cert-verify=true") ||
		!strings.Contains(line, "password=") {
		t.Fatalf("bad surge trojan line: %s", line)
	}
}
