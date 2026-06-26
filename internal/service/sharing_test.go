package service

import (
	"strings"
	"testing"

	"gorm.io/gorm"

	"github.com/ludandaye/hy2board/internal/model"
)

func gormModel(id uint) gorm.Model { return gorm.Model{ID: id} }

func TestPlanSharingActions(t *testing.T) {
	users := []model.User{
		{Model: gormModel(1), Username: "alice", MaxIPs: 2},
		{Model: gormModel(2), Username: "bob", MaxIPs: 2},
	}
	snaps := []NodeSnapshot{
		{Name: "HK1", OnlineUsers: map[string]int{"alice": 1, "bob": 1}},
		{Name: "JP2", OnlineUsers: map[string]int{"alice": 1}},
	}
	// alice is over (trimmed>0) on both nodes; bob only had a blocked attempt (no kick).
	state := func(id uint) (trimmed, distinct int, blocked bool) {
		if id == 1 {
			return 1, 2, true
		}
		return 0, 2, true
	}
	acts := planSharingActions(users, snaps, state, func(uint) bool { return true })
	if len(acts) != 2 {
		t.Fatalf("want 2 actions got %d", len(acts))
	}
	var alice *sharingAction
	for i := range acts {
		if acts[i].User.Username == "alice" {
			alice = &acts[i]
		}
	}
	if alice == nil || len(alice.KickNodeIdx) != 2 || !alice.Alert {
		t.Fatalf("alice should kick 2 nodes and alert: %+v", alice)
	}
}

func TestFormatSharingAlert(t *testing.T) {
	s := formatSharingAlert("alice", 3, 2, []string{"HK1", "JP2"})
	if !strings.Contains(s, "alice") || !strings.Contains(s, "HK1") {
		t.Fatalf("alert text missing fields: %s", s)
	}
}
