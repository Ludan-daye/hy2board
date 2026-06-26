package service

import (
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/ludandaye/hy2board/internal/config"
	"github.com/ludandaye/hy2board/internal/database"
	"github.com/ludandaye/hy2board/internal/model"
)

type sharingAction struct {
	User        model.User
	KickNodeIdx []int // indices into the snapshot slice where this user is online
	Distinct    int
	Alert       bool
}

// planSharingActions is the pure core: given limited users, the current node snapshots, a
// per-user state function (trimmed/distinct/blocked) and an alert-gate, decide who to kick
// where and whether to alert. A user is acted on if they were trimmed (over limit) or had a
// recent blocked attempt. Kicks target only the nodes where they're currently online.
func planSharingActions(
	users []model.User,
	snaps []NodeSnapshot,
	state func(userID uint) (trimmed, distinct int, blocked bool),
	shouldAlert func(userID uint) bool,
) []sharingAction {
	var out []sharingAction
	for _, u := range users {
		trimmed, distinct, blocked := state(u.ID)
		if trimmed == 0 && !blocked {
			continue
		}
		act := sharingAction{User: u, Distinct: distinct}
		if trimmed > 0 {
			for i := range snaps {
				if snaps[i].OnlineUsers[u.Username] > 0 {
					act.KickNodeIdx = append(act.KickNodeIdx, i)
				}
			}
		}
		act.Alert = shouldAlert(u.ID)
		out = append(out, act)
	}
	return out
}

func formatSharingAlert(username string, distinct, limit int, kicked []string) string {
	b := &strings.Builder{}
	fmt.Fprintf(b, "⚠️ 共享告警\n用户：%s\n当前不同IP：%d（上限 %d）\n", username, distinct, limit)
	if len(kicked) > 0 {
		fmt.Fprintf(b, "已踢节点：%s\n", strings.Join(kicked, ", "))
	} else {
		b.WriteString("有新IP尝试接入被拦截（疑似共享）\n")
	}
	return b.String()
}

func formatSharingUserNotice(limit int) string {
	return fmt.Sprintf(
		"⚠️ 安全提醒\n检测到你的账号从多个网络同时使用，已超过套餐上限（%d 个）。多余连接已断开。\n如非本人操作，请尽快修改密码。",
		limit,
	)
}

// enforceSharing runs each refresh cycle: trim/kick over-limit users, alert admin, notify user.
func enforceSharing(nodes []model.Node, snaps []NodeSnapshot) {
	var users []model.User
	if err := database.DB.Where("max_ips > 0").Find(&users).Error; err != nil {
		return
	}
	now := time.Now()
	state := func(id uint) (int, int, bool) {
		trimmed, distinct := TrimOverLimit(id, ipLimitFor(users, id), now)
		return trimmed, distinct, RecentlyBlocked(id, 90*time.Second, now)
	}
	for _, act := range planSharingActions(users, snaps, state, func(id uint) bool { return ShouldAlertSharing(id, now) }) {
		var kicked []string
		for _, idx := range act.KickNodeIdx {
			if err := KickUser(nodes[idx], act.User.Username); err == nil {
				kicked = append(kicked, snaps[idx].Name)
			}
		}
		if act.Alert {
			notifyAdminTG(formatSharingAlert(act.User.Username, act.Distinct, act.User.MaxIPs, kicked))
			notifyUserTG(act.User.TelegramID, formatSharingUserNotice(act.User.MaxIPs))
		}
	}
}

func ipLimitFor(users []model.User, id uint) int {
	for _, u := range users {
		if u.ID == id {
			return u.MaxIPs
		}
	}
	return 0
}

// notifyAdminTG sends to the bound admin (private), never the customer group.
func notifyAdminTG(text string) {
	if !config.C.HasTelegram() {
		return
	}
	adminID, ok := getAdminTelegramID()
	if !ok {
		return
	}
	bot, err := tgbotapi.NewBotAPI(config.C.Telegram.BotToken)
	if err != nil {
		return
	}
	bot.Send(tgbotapi.NewMessage(adminID, text))
}

func notifyUserTG(tgID int64, text string) {
	if tgID == 0 || !config.C.HasTelegram() {
		return
	}
	bot, err := tgbotapi.NewBotAPI(config.C.Telegram.BotToken)
	if err != nil {
		return
	}
	bot.Send(tgbotapi.NewMessage(tgID, text))
}
