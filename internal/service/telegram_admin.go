package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/ludandaye/hy2board/internal/config"
)

type TelegramStatus struct {
	Enabled               bool   `json:"enabled"`
	BotConfigured         bool   `json:"bot_configured"`
	GroupRegistered       bool   `json:"group_registered"`
	GroupChatID           int64  `json:"group_chat_id"`
	AdminUsername         string `json:"admin_username"`
	AdminBound            bool   `json:"admin_bound"`
	AdminTelegramID       int64  `json:"admin_telegram_id"`
	DailyPostTime         string `json:"daily_post_time"`
	WeeklyLeaderboardTime string `json:"weekly_leaderboard_time"`
}

func GetTelegramStatus() TelegramStatus {
	groupID := getGroupChatID()
	status := TelegramStatus{
		Enabled:               config.C.Telegram.Enabled,
		BotConfigured:         strings.TrimSpace(config.C.Telegram.BotToken) != "",
		GroupRegistered:       groupID != 0,
		GroupChatID:           groupID,
		AdminUsername:         strings.TrimSpace(config.C.Admin.Username),
		DailyPostTime:         "每天 09:00 Asia/Shanghai",
		WeeklyLeaderboardTime: "每周日 20:00 Asia/Shanghai",
	}
	if adminID, ok := getAdminTelegramID(); ok {
		status.AdminBound = true
		status.AdminTelegramID = adminID
	}
	return status
}

func SendTestAdminNewMemberNotice(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !config.C.HasTelegram() {
		return fmt.Errorf("telegram not configured")
	}
	adminID, ok := getAdminTelegramID()
	if !ok {
		return fmt.Errorf("admin telegram is not bound")
	}

	bot, err := tgbotapi.NewBotAPI(config.C.Telegram.BotToken)
	if err != nil {
		return err
	}

	groupLabel := "未登记"
	if groupID := getGroupChatID(); groupID != 0 {
		groupLabel = fmt.Sprintf("%d", groupID)
	}
	text := fmt.Sprintf(
		"🧪 新成员进群通知测试\n\n"+
			"群组：%s\n"+
			"昵称：测试新朋友\n"+
			"用户名：@test_user\n"+
			"Telegram ID：100000000\n"+
			"时间：%s\n\n"+
			"如果你收到这条，说明“新人入群 → 私发管理员”这条链路是通的。",
		groupLabel,
		time.Now().In(cstLoc()).Format("2006-01-02 15:04:05"),
	)

	if _, err := bot.Send(tgbotapi.NewMessage(adminID, text)); err != nil {
		return err
	}
	return ctx.Err()
}
