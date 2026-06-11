package service

import (
	"testing"
	"time"

	"github.com/ludandaye/hy2board/internal/model"
)

func TestBuildUserBucketsClassifiesUserAndNodeRisks(t *testing.T) {
	now := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	users := []model.User{
		{Username: "expiring", Enabled: true, ExpiresAt: now.Add(3 * 24 * time.Hour), TrafficLimit: 1000, TrafficUsed: 100},
		{Username: "near", Enabled: true, ExpiresAt: now.Add(30 * 24 * time.Hour), TrafficLimit: 1000, TrafficUsed: 810},
		{Username: "over", Enabled: true, ExpiresAt: now.Add(30 * 24 * time.Hour), TrafficLimit: 1000, TrafficUsed: 100},
		{Username: "expired", Enabled: true, ExpiresAt: now.Add(-24 * time.Hour), TrafficLimit: 1000, TrafficUsed: 100},
		{Username: "disabled", Enabled: false, ExpiresAt: now.Add(30 * 24 * time.Hour), TrafficLimit: 1000, TrafficUsed: 1000},
	}
	nodes := []model.Node{
		{Name: "ok", Host: "1.1.1.1", Port: 443, Healthy: true},
		{Name: "down", Host: "2.2.2.2", Port: 443, Healthy: false},
	}
	liveUsed := func(username string) int64 {
		if username == "over" {
			return 1200
		}
		return 0
	}

	got := BuildUserBuckets(users, nodes, now, liveUsed)

	if len(got.ExpiringSoon) != 1 || got.ExpiringSoon[0].Username != "expiring" {
		t.Fatalf("expected expiring user, got %#v", got.ExpiringSoon)
	}
	if len(got.NearLimit) != 1 || got.NearLimit[0].Username != "near" || got.NearLimit[0].PercentUsed != 81 {
		t.Fatalf("expected near-limit user with 81%%, got %#v", got.NearLimit)
	}
	if len(got.OverLimit) != 1 || got.OverLimit[0].Username != "over" || got.OverLimit[0].TrafficUsed != 1200 {
		t.Fatalf("expected live over-limit user, got %#v", got.OverLimit)
	}
	if len(got.Expired) != 1 || got.Expired[0].Username != "expired" {
		t.Fatalf("expected expired user, got %#v", got.Expired)
	}
	if len(got.Disabled) != 1 || got.Disabled[0].Username != "disabled" {
		t.Fatalf("expected disabled user, got %#v", got.Disabled)
	}
	if len(got.DeadNodes) != 1 || got.DeadNodes[0].Name != "down" {
		t.Fatalf("expected dead node, got %#v", got.DeadNodes)
	}
}
