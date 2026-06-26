package service

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ludandaye/hy2board/internal/model"
)

func TestGetNodeOnlineMap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/online" {
			t.Errorf("want /online got %s", r.URL.Path)
		}
		w.Write([]byte(`{"alice":2,"bob":1}`))
	}))
	defer srv.Close()
	m, err := GetNodeOnlineMap(model.Node{TrafficAPI: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	if m["alice"] != 2 || m["bob"] != 1 || len(m) != 2 {
		t.Fatalf("bad map: %v", m)
	}
}

func TestKickUser(t *testing.T) {
	var gotBody, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/kick" || r.Method != http.MethodPost {
			t.Errorf("want POST /kick got %s %s", r.Method, r.URL.Path)
		}
		buf := make([]byte, r.ContentLength)
		r.Body.Read(buf)
		gotBody = string(buf)
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(200)
	}))
	defer srv.Close()
	if err := KickUser(model.Node{TrafficAPI: srv.URL, TrafficSecret: "sek"}, "alice"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotBody, "alice") {
		t.Fatalf("body missing username: %q", gotBody)
	}
	if gotAuth != "sek" {
		t.Fatalf("auth header want sek got %q", gotAuth)
	}
}
