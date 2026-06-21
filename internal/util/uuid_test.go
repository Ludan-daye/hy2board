package util

import "testing"

func TestVlessUUIDIsDeterministicAndValid(t *testing.T) {
	a := VlessUUID("ludandaye")
	b := VlessUUID("ludandaye")
	if a != b {
		t.Fatalf("not deterministic: %s vs %s", a, b)
	}
	if VlessUUID("zyk") == a {
		t.Fatalf("different users must get different uuids")
	}
	// canonical 8-4-4-4-12, version 5, RFC4122 variant
	if len(a) != 36 || a[14] != '5' || (a[19] != '8' && a[19] != '9' && a[19] != 'a' && a[19] != 'b') {
		t.Fatalf("not a v5 uuid: %s", a)
	}
}
