package service

import "testing"

func TestParseVlessStats(t *testing.T) {
	body := []byte(`{"alice":{"tx":100,"rx":200},"bob":{"tx":5,"rx":0}}`)
	m, err := parseVlessStats(body)
	if err != nil || m["alice"].TX != 100 || m["alice"].RX != 200 || m["bob"].TX != 5 {
		t.Fatalf("bad parse: %#v err=%v", m, err)
	}
}
