package util

import (
	"crypto/sha1"
	"encoding/hex"
)

// fixed namespace UUID for hy2board VLESS identities (constant forever)
var vlessNamespace = [16]byte{0x6b, 0xa7, 0xb8, 0x12, 0x9d, 0xad, 0x11, 0xd1, 0x80, 0xb4, 0x00, 0xc0, 0x4f, 0xd4, 0x30, 0xc8}

// VlessUUID returns a deterministic RFC4122 v5 UUID for a username, so the same
// user maps to the same UUID on every node and in the panel without new storage.
func VlessUUID(username string) string {
	h := sha1.New()
	h.Write(vlessNamespace[:])
	h.Write([]byte(username))
	s := h.Sum(nil)[:16]
	s[6] = (s[6] & 0x0f) | 0x50 // version 5
	s[8] = (s[8] & 0x3f) | 0x80 // RFC4122 variant
	d := hex.EncodeToString(s)
	return d[0:8] + "-" + d[8:12] + "-" + d[12:16] + "-" + d[16:20] + "-" + d[20:32]
}
