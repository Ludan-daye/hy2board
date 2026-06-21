package util

import (
	"crypto/sha256"
	"encoding/hex"
)

// TrojanPassword is a deterministic per-user Trojan password derived from the
// username (distinct from hy2_password). Stable across nodes and the panel.
func TrojanPassword(username string) string {
	sum := sha256.Sum256([]byte("hy2board-trojan:" + username))
	return hex.EncodeToString(sum[:])
}
