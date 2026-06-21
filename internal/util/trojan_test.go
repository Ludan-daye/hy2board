package util

import "testing"

func TestTrojanPasswordDeterministicAndDistinct(t *testing.T) {
	a := TrojanPassword("ludandaye")
	if a != TrojanPassword("ludandaye") {
		t.Fatal("not deterministic")
	}
	if a == TrojanPassword("zyk") {
		t.Fatal("different users must differ")
	}
	if len(a) != 64 { // sha256 hex
		t.Fatalf("want 64 hex chars, got %d", len(a))
	}
}
