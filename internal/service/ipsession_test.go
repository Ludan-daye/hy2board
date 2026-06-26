package service

import (
	"testing"
	"time"
)

func TestTouchIP(t *testing.T) {
	ipStore = map[uint]*userIPState{}
	now := time.Unix(1000, 0)
	// first two distinct IPs under limit 2 -> allowed
	if ok, n := TouchIP(1, "1.1.1.1", 2, now); !ok || n != 1 {
		t.Fatalf("ip1 want allowed,1 got %v,%d", ok, n)
	}
	if ok, n := TouchIP(1, "2.2.2.2", 2, now); !ok || n != 2 {
		t.Fatalf("ip2 want allowed,2 got %v,%d", ok, n)
	}
	// same IP again -> allowed, no count increase
	if ok, n := TouchIP(1, "1.1.1.1", 2, now); !ok || n != 2 {
		t.Fatalf("ip1 repeat want allowed,2 got %v,%d", ok, n)
	}
	// new third IP over limit -> blocked, not recorded
	if ok, n := TouchIP(1, "3.3.3.3", 2, now); ok || n != 2 {
		t.Fatalf("ip3 want blocked,2 got %v,%d", ok, n)
	}
	if !RecentlyBlocked(1, time.Minute, now) {
		t.Fatal("expected RecentlyBlocked true after a block")
	}
	// limit 0 = unlimited -> always allowed
	if ok, _ := TouchIP(2, "9.9.9.9", 0, now); !ok {
		t.Fatal("limit 0 should always allow")
	}
}

func TestWindowExpiry(t *testing.T) {
	ipStore = map[uint]*userIPState{}
	now := time.Unix(1000, 0)
	TouchIP(1, "1.1.1.1", 5, now)
	if DistinctActiveIPs(1, now) != 1 {
		t.Fatal("want 1 active")
	}
	later := now.Add(ipWindow + time.Second)
	if DistinctActiveIPs(1, later) != 0 {
		t.Fatal("want 0 active after window")
	}
}

func TestTrimOverLimit(t *testing.T) {
	ipStore = map[uint]*userIPState{}
	base := time.Unix(1000, 0)
	// 3 IPs recorded at increasing times under a high limit
	TouchIP(1, "1.1.1.1", 9, base)
	TouchIP(1, "2.2.2.2", 9, base.Add(time.Second))
	TouchIP(1, "3.3.3.3", 9, base.Add(2*time.Second))
	// now lower the limit to 2 -> trims oldest 1, keeps 2 newest
	trimmed, distinct := TrimOverLimit(1, 2, base.Add(3*time.Second))
	if trimmed != 1 || distinct != 2 {
		t.Fatalf("want trimmed1,distinct2 got %d,%d", trimmed, distinct)
	}
	// the trimmed (oldest) IP is gone; a re-auth from it is now blocked
	if ok, _ := TouchIP(1, "1.1.1.1", 2, base.Add(4*time.Second)); ok {
		t.Fatal("trimmed IP should be blocked on re-auth at limit")
	}
}

func TestShouldAlertSharing(t *testing.T) {
	ipStore = map[uint]*userIPState{}
	now := time.Unix(1000, 0)
	TouchIP(1, "1.1.1.1", 1, now) // create state
	if !ShouldAlertSharing(1, now) {
		t.Fatal("first alert should fire")
	}
	if ShouldAlertSharing(1, now.Add(time.Minute)) {
		t.Fatal("second alert within 30m should be suppressed")
	}
	if !ShouldAlertSharing(1, now.Add(alertEvery+time.Second)) {
		t.Fatal("alert after window should fire")
	}
}
