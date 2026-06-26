package handler

import "testing"

func TestIPFromAddr(t *testing.T) {
	cases := map[string]string{
		"1.2.3.4:5678":      "1.2.3.4",
		"1.2.3.4":           "1.2.3.4",
		"[2001:db8::1]:443": "2001:db8::1",
		"":                  "",
	}
	for in, want := range cases {
		if got := ipFromAddr(in); got != want {
			t.Errorf("ipFromAddr(%q)=%q want %q", in, got, want)
		}
	}
}
