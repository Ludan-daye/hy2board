package service

import (
	"fmt"

	"github.com/ludandaye/hy2board/internal/model"
	"github.com/ludandaye/hy2board/internal/util"
)

// NodeHasTrojan reports whether a node should emit a Trojan line.
func NodeHasTrojan(n model.Node) bool {
	return n.TrojanEnabled && n.TrojanPort > 0
}

// TrojanSurgeLine renders a Surge `trojan` proxy line. The name reuses VlessName
// (<node>-T) so the "TCP fallback" node is consistent across formats — VLESS in
// Clash/URI, Trojan in Surge.
func TrojanSurgeLine(u model.User, n model.Node) string {
	return fmt.Sprintf("%s = trojan, %s, %d, password=%s, sni=%s, skip-cert-verify=true",
		VlessName(n), n.Host, n.TrojanPort, util.TrojanPassword(u.Username), n.TrojanSNI)
}
