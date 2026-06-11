package service

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	qrcode "github.com/skip2/go-qrcode"

	"github.com/ludandaye/hy2board/internal/config"
	"github.com/ludandaye/hy2board/internal/database"
	"github.com/ludandaye/hy2board/internal/model"
)

// requirePrivate returns true if the message is in a private chat.
// If not, it sends a guard message and returns false.
func requirePrivate(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) bool {
	if msg.Chat.Type == "private" {
		return true
	}
	reply(bot, msg, fmt.Sprintf("🔒 此命令涉及个人信息，请私聊我 @%s 使用。", bot.Self.UserName))
	return false
}

// cstLoc returns Asia/Shanghai location, falling back to UTC+8 fixed offset if
// tzdata is not installed in the container.
func cstLoc() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	return loc
}

// ─────────────────────────── settings helpers ────────────────────────────────

const keyGroupChatID = "telegram_group_chat_id"

func getSetting(key string) string {
	var s model.Setting
	if err := database.DB.Where("key = ?", key).First(&s).Error; err != nil {
		return ""
	}
	return s.Value
}

func setSetting(key, val string) error {
	return database.DB.Save(&model.Setting{Key: key, Value: val}).Error
}

func getGroupChatID() int64 {
	s := getSetting(keyGroupChatID)
	if s == "" {
		return 0
	}
	var id int64
	fmt.Sscanf(s, "%d", &id)
	return id
}

// ─────────────────────────── Chinese UI text ─────────────────────────────────

const welcomeZH = `嗨！我是小卤蛋的 hy2 小助手 🤖💙

🔧 账号类（私聊）
/link <用户名> <密码> — 绑定账号
/unlink — 解除绑定
/sub — 订阅 + 二维码 📡
/quota — 流量 / 到期日 📊
/成就 — 徽章墙 🏅

💎 查套餐
/plans — 所有套餐

🎮 互动娱乐
/签到 — 每日打卡 📅
/一签 — 今日运势 🎴
/笑话 — 来一发段子 😂
/骰子 — 摇 1-6 🎲

⚙️ 群管理
/register_group — 群内注册公告频道（管理员）

📅 每天 09:00 播报节点状态
🏆 每周日 20:00 本周流量王
💬 群里 @ 我随时呼出帮助`

// ─────────────────────────── daily tips pool ─────────────────────────────────

var dailyTips = []string{
	"💡 连不上？先 /quota 看流量是否用完。",
	"💡 Clash 记得打开 TUN 模式，速度更稳。",
	"💡 手机用 Shadowrocket 直接扫 /sub 二维码最快。",
	"💡 自动重置流量：从绑定那天起每 30 天一个周期。",
	"💡 节点离线会自动从订阅里剔除，不用你操作。",
	"💡 Clash 里 manually 选节点可锁定某个线路。",
	"💡 订阅链接别分享，可在面板一键轮换 token。",
	"💡 AI 请求走住宅链路，随便选节点都能用。",
	"💡 国内域名默认直连，刷抖音/B站不消耗流量。",
	"💡 有疑问群里 @ 管理员，一般当天回复。",
	"💡 桌面端推荐 Clash Verge，移动端推荐 Shadowrocket。",
	"💡 一个订阅可以同时在多台设备上用，节点不冲突。",
}

var serviceHighlights = []string{
	"🚀 基于 Hysteria 2 协议（QUIC/UDP），延迟低、抗干扰能力强，比老协议快 30%+",
	"🤖 ChatGPT / Claude / Gemini 走专用住宅代理链，原生解锁不卡，写代码聊天丝滑",
	"🎬 流媒体线路覆盖 Netflix / Disney+ / HBO Max / Hulu，4K 不转圈",
	"🏠 中国大陆域名默认直连，刷 B 站 / 抖音 / 微信完全不走梯子也不扣流量",
	"🔀 多节点智能切换，某个节点掉线自动跳过，用户全程无感知",
	"🛡 订阅 token 一键轮换，泄漏不用慌",
	"📱 一套订阅全平台通吃：Clash / Mihomo / Surge / Shadowrocket",
	"📊 实时面板监控每个节点、每个用户的流量和速度",
	"💎 套餐灵活：试用 / 月付 / 年付 / 无限流量 任选",
	"⚡ 面板操作友好，续期改密码重置流量都是一键",
	"🎯 AI / 流媒体 / 国内直连 / 广告拦截 四大规则包可按需开关",
	"🔐 HY2 协议原生 TLS 伪装，比 SS/VMess 更抗探测",
}

// ─────────────────────────── fortune pool (30 items) ─────────────────────────

var fortuneList = []string{
	"🎴 上上签 · 今日宜：多刷 AI 对话 · 忌：分享订阅链接",
	"🎴 上签 · 今日宜：升级套餐 · 忌：熬夜下载",
	"🎴 上签 · 今日宜：看 Netflix 高清 · 忌：手动改节点",
	"🎴 中签 · 今日宜：Clash 自动模式 · 忌：到处切换",
	"🎴 中签 · 今日宜：看 YouTube 4K · 忌：后台下载大文件",
	"🎴 中签 · 今日宜：和 AI 写代码 · 忌：开太多标签页",
	"🎴 中签 · 今日宜：远程办公 · 忌：用公共 WiFi",
	"🎴 中签 · 今日宜：订阅续费 · 忌：分享密码",
	"🎴 中签 · 今日宜：学习新技能 · 忌：沉迷短视频",
	"🎴 吉 · 今日宜：和群友聊天 · 忌：装高冷",
	"🎴 吉 · 今日宜：/quota 看看自己流量 · 忌：用超",
	"🎴 吉 · 今日宜：开启流媒体规则 · 忌：一直用默认",
	"🎴 吉 · 今日宜：分享技术心得 · 忌：做伸手党",
	"🎴 吉 · 今日宜：夜深看片 · 忌：被老板发现",
	"🎴 吉 · 今日宜：@ 管理员反馈问题 · 忌：默默忍受",
	"🎴 平 · 今日宜：正常工作 · 忌：乱点广告",
	"🎴 平 · 今日宜：喝点水 · 忌：坐太久",
	"🎴 平 · 今日宜：写点代码 · 忌：bug 赶不完",
	"🎴 平 · 今日宜：/sub 更新订阅 · 忌：用老链接",
	"🎴 小凶 · 今日宜：关闭不用的标签 · 忌：占用带宽",
	"🎴 小凶 · 今日宜：早点休息 · 忌：通宵追剧",
	"🎴 小凶 · 今日宜：节制上网 · 忌：超流量额度",
	"🎴 小凶 · 今日宜：谨慎点外链 · 忌：信陌生邮件",
	"🎴 大吉 · 今日宜：面试顺利 · 忌：网络卡顿（放心我们不会）",
	"🎴 大吉 · 今日宜：表白 · 忌：用前任的账号",
	"🎴 大吉 · 今日宜：中奖 · 忌：不相信运气",
	"🎴 大吉 · 今日宜：写完 PPT · 忌：被 AI 卡住",
	"🎴 大凶 · 今日宜：低调 · 忌：吹牛 · 但 hy2 稳定不会让你掉链子",
	"🎴 吉 · 今日宜：给群里发条消息 · 忌：潜水一整天",
	"🎴 吉 · 今日宜：给管理员点个赞 · 忌：白嫖不说声谢谢",
}

// ─────────────────────────── joke pool (30 items) ────────────────────────────

var jokeList = []string{
	"😂 为什么 Hysteria 这么快？因为它用 UDP，没那么多规矩 🤷",
	"😂 程序员的浪漫：用 AI 聊天时关掉 VPN 看它崩成什么样。",
	"😂 我家猫最喜欢看的视频也要走代理，因为它只认 Netflix。",
	"😂 调试 bug 比连 VPN 还慢——是因为 bug 会反连你。",
	"😂 Clash 规则：所有我不认识的域名都走代理——谨慎但真实。",
	"😂 网线只是一根绳子，直到它连上了好节点。",
	"😂 今天你对我爱理不理，明天 ChatGPT 让你高攀不起。",
	"😂 运维三件套：重启、重连、重新配订阅。",
	"😂 VPN 稳定的秘诀：选个好面板 + 一个好朋友（我）。",
	"😂 面试官：说说 QUIC 的优势。我：它能让 Netflix 不卡。",
	"😂 老板问我为什么下载那么多——答：我在备份代码。",
	"😂 我没有社交恐惧症，我只是不喜欢网络延迟。",
	"😂 用代理的三大快乐：省、快、稳。可惜老板不让报销。",
	"😂 GitHub 上克隆仓库最快的时候，就是你终于配好节点之后。",
	"😂 代码写不出来的时候，我就去看看节点监控——看别人流量都 TB 级，突然就有动力了。",
	"😂 和 AI 聊天有三种境界：问它问题、反驳它、教它写 Go。",
	"😂 什么叫真朋友？会帮你配 Clash 规则的那种。",
	"😂 今日新闻：某用户订阅 token 被分享，管理员一键轮换，友谊翻车。",
	"😂 以前我以为 ping 低就是最快，后来才知道有 Hysteria 这种东西。",
	"😂 Surge 和 Clash 的区别？一个是贵的，一个是免费的……但规则其实差不多。",
	"😂 节点离线了怎么办？答：换一个。bug 解决不了怎么办？答：换一个程序员。",
	"😂 程序员的浪漫：在 nginx 配置里写情书。",
	"😂 早安！今天的网速祝你也像火箭一样快。",
	"😂 别问我怎么选节点，Clash 里有 Auto，都有 Auto 了还手动个啥。",
	"😂 我的流量就像钱包——每月初富得流油，月底拮据。",
	"😂 什么时候最幸福？/quota 发现还剩一半。",
	"😂 传统艺能：配了 VPN 先测速，测完就忘了。",
	"😂 我跟我朋友说你要用 HY2，他以为是某种饮料。",
	"😂 AI 聊天要走住宅代理，不然它说你是机器人——我才是机器人啊！",
	"😂 有些 bug 会自己消失，有些节点也会——所以我们有自动切换。",
}

// ─────────────────────────── bot lifecycle ───────────────────────────────────

// StartTelegramBot launches the bot in a goroutine. No-op if the token is
// empty or config.Telegram.Enabled is false.
func StartTelegramBot() {
	if !config.C.HasTelegram() {
		log.Printf("telegram: disabled (no token or enabled=false)")
		return
	}
	bot, err := tgbotapi.NewBotAPI(config.C.Telegram.BotToken)
	if err != nil {
		log.Printf("telegram: init failed: %v", err)
		return
	}
	log.Printf("telegram: bot authorized as @%s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30
	updates := bot.GetUpdatesChan(u)

	go func() {
		for update := range updates {
			if update.Message == nil {
				continue
			}
			if len(update.Message.NewChatMembers) > 0 {
				go handleNewMembers(bot, update.Message)
				continue
			}
			// Telegram only flags ASCII commands as bot_command entities,
			// so /签到 / /笑话 etc. fail msg.IsCommand(). Detect "/" prefix manually.
			if strings.HasPrefix(strings.TrimSpace(update.Message.Text), "/") {
				go handleCommand(bot, update.Message)
				continue
			}
			// Non-command: only respond if THIS bot was mentioned
			if isBotMentioned(update.Message, bot.Self.UserName) {
				go handleMention(bot, update.Message)
			}
		}
	}()
}

// parseCommandText returns (cmd, args) from a message starting with "/", handling
// non-ASCII command names and "@botname" suffixes that Telegram's IsCommand()
// helper misses.
func parseCommandText(text, botUsername string) (string, string) {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "/") {
		return "", ""
	}
	rest := text[1:]
	fields := strings.SplitN(rest, " ", 2)
	cmdPart := fields[0]
	args := ""
	if len(fields) == 2 {
		args = strings.TrimSpace(fields[1])
	}
	if at := strings.Index(cmdPart, "@"); at >= 0 {
		// /cmd@botname → keep only "cmd"
		cmdPart = cmdPart[:at]
	}
	return cmdPart, args
}

// ─────────────────────────── command router ──────────────────────────────────

func handleCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("telegram: panic in handler: %v", r)
		}
	}()

	cmd := msg.Command()
	args := strings.TrimSpace(msg.CommandArguments())
	// Telegram doesn't tag non-ASCII commands as bot_command entities, so
	// msg.Command() returns "" for /签到 etc. Fall back to manual parsing.
	if cmd == "" {
		cmd, args = parseCommandText(msg.Text, bot.Self.UserName)
	}
	tgID := msg.From.ID

	switch cmd {
	case "start", "help":
		reply(bot, msg, welcomeZH)
	case "link":
		handleLink(bot, msg, tgID, args)
	case "unlink", "解绑":
		handleUnlink(bot, msg, tgID)
	case "sub":
		handleSub(bot, msg, tgID)
	case "quota":
		handleQuota(bot, msg, tgID)
	case "register_group":
		handleRegisterGroup(bot, msg, tgID)
	case "签到", "checkin":
		handleCheckin(bot, msg, tgID)
	case "一签", "fortune":
		handleFortune(bot, msg, tgID)
	case "笑话", "joke":
		handleJoke(bot, msg)
	case "骰子", "dice":
		handleDice(bot, msg)
	case "成就", "achievement":
		handleAchievements(bot, msg, tgID)
	case "plans", "套餐":
		handlePlans(bot, msg)
	default:
		reply(bot, msg, "未知命令，输入 /help 查看可用命令。")
	}
}

// ─────────────────────────── /link ───────────────────────────────────────────

func handleLink(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, tgID int64, args string) {
	if !requirePrivate(bot, msg) {
		return
	}
	parts := strings.Fields(args)
	if len(parts) != 2 {
		reply(bot, msg, "用法：/link <用户名> <密码>")
		return
	}
	username, password := parts[0], parts[1]

	var user model.User
	if database.DB.Where("username = ?", username).First(&user).Error != nil {
		reply(bot, msg, "账号不存在。")
		return
	}
	if user.LoginPassword == "" || !user.CheckLoginPassword(password) {
		reply(bot, msg, "用户名或密码错误。")
		return
	}

	// Prevent hijacking: if this tgID is already bound to another user, refuse.
	var other model.User
	if err := database.DB.Where("telegram_id = ? AND id <> ?", tgID, user.ID).First(&other).Error; err == nil {
		reply(bot, msg, fmt.Sprintf("此 Telegram 账号已绑定到用户 %s，请联系管理员解绑。", other.Username))
		return
	}

	if err := database.DB.Model(&user).Update("telegram_id", tgID).Error; err != nil {
		reply(bot, msg, "保存失败："+err.Error())
		return
	}
	reply(bot, msg, fmt.Sprintf("✅ 已绑定到 %s。试试 /sub 或 /quota。", user.Username))
}

// ─────────────────────────── /unlink ─────────────────────────────────────────

func handleUnlink(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, tgID int64) {
	if !requirePrivate(bot, msg) {
		return
	}
	user, err := findUserByTelegramID(tgID)
	if err != nil {
		reply(bot, msg, "你当前没有绑定任何账号。")
		return
	}
	if err := database.DB.Model(&user).Update("telegram_id", 0).Error; err != nil {
		reply(bot, msg, "解绑失败："+err.Error())
		return
	}
	reply(bot, msg, fmt.Sprintf("✅ 已解绑账号 %s。\n\n如需重新绑定，发送 /link <用户名> <密码>", user.Username))
}

// ─────────────────────────── /sub ────────────────────────────────────────────

func handleSub(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, tgID int64) {
	if !requirePrivate(bot, msg) {
		return
	}
	user, err := findUserByTelegramID(tgID)
	if err != nil {
		reply(bot, msg, "你还没绑定，请先发送 /link <用户名> <密码>。")
		return
	}
	if !user.IsActive() {
		reply(bot, msg, "订阅不可用（已停用 / 过期 / 超流量）。")
		return
	}

	base := "https://vpn.linkbyfree.com/api/sub/" + user.SubToken
	text := fmt.Sprintf(
		"📡 订阅链接（点击复制）\n\n🔗 URI:\n%s\n\n🌐 Clash:\n%s?format=clash\n\n🌊 Surge:\n%s?format=surge\n\n🚀 Shadowrocket:\n%s?format=shadowrocket-conf",
		base, base, base, base,
	)
	reply(bot, msg, text)

	// Send QR for the URI (universal)
	png, err := qrcode.Encode(base, qrcode.Medium, 512)
	if err == nil {
		photo := tgbotapi.NewPhoto(msg.Chat.ID, tgbotapi.FileBytes{Name: "qr.png", Bytes: png})
		photo.Caption = "用任何 Hysteria 2 客户端扫码导入"
		if _, err := bot.Send(photo); err != nil {
			log.Printf("telegram: send QR failed: %v", err)
		}
	}
}

// ─────────────────────────── /quota ──────────────────────────────────────────

func handleQuota(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, tgID int64) {
	if !requirePrivate(bot, msg) {
		return
	}
	user, err := findUserByTelegramID(tgID)
	if err != nil {
		reply(bot, msg, "你还没绑定，请先发送 /link <用户名> <密码>。")
		return
	}

	// Live traffic from cache
	liveTX, liveRX := int64(0), int64(0)
	if s, ok := GetUserStat(user.Username); ok {
		liveTX, liveRX = s.TotalTX, s.TotalRX
	}

	statusZH := "活跃"
	switch {
	case !user.Enabled:
		statusZH = "已禁用"
	case user.IsExpired():
		statusZH = "已过期"
	case user.TrafficExceeded():
		statusZH = "流量超限"
	}

	expiry := "永不过期"
	daysLeft := ""
	if !user.ExpiresAt.IsZero() {
		expiry = user.ExpiresAt.Format("2006-01-02")
		d := int(time.Until(user.ExpiresAt).Hours() / 24)
		daysLeft = fmt.Sprintf("（剩 %d 天）", d)
	}

	limit := "不限"
	if user.TrafficLimit > 0 {
		limit = humanBytes(user.TrafficLimit)
	}

	var rules []string
	if user.RuleAI || user.ChainProxy {
		rules = append(rules, "AI代理")
	}
	if user.RuleStreaming {
		rules = append(rules, "流媒体")
	}
	if user.RuleChina {
		rules = append(rules, "国内直连")
	}
	if user.RuleAdBlock {
		rules = append(rules, "广告拦截")
	}
	ruleStr := "（未启用）"
	if len(rules) > 0 {
		ruleStr = strings.Join(rules, ", ")
	}

	text := fmt.Sprintf(
		"👤 用户: %s\n📊 状态: %s\n📦 流量: 已用 %s（↑%s ↓%s） / 上限 %s\n⏰ 到期: %s %s\n🧭 规则: %s",
		user.Username, statusZH,
		humanBytes(liveTX+liveRX), humanBytes(liveTX), humanBytes(liveRX), limit,
		expiry, daysLeft, ruleStr,
	)
	reply(bot, msg, text)
}

// ─────────────────────────── /register_group ─────────────────────────────────

func handleRegisterGroup(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, tgID int64) {
	if msg.Chat.Type != "group" && msg.Chat.Type != "supergroup" {
		reply(bot, msg, "此命令只能在群聊中使用。")
		return
	}
	user, err := findUserByTelegramID(tgID)
	if err != nil {
		reply(bot, msg, "你还没绑定账号，请先私聊 bot 发送 /link。")
		return
	}
	if err := setSetting(keyGroupChatID, fmt.Sprintf("%d", msg.Chat.ID)); err != nil {
		reply(bot, msg, "保存失败："+err.Error())
		return
	}
	reply(bot, msg, fmt.Sprintf(
		"✅ 本群（%s）已登记为公告频道。由 %s 绑定。\n每天早上 09:00 将推送节点状态 + 小贴士。",
		msg.Chat.Title, user.Username,
	))
}

// ─────────────────────────── /签到 ───────────────────────────────────────────

func handleCheckin(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, tgID int64) {
	user, err := findUserByTelegramID(tgID)
	if err != nil {
		reply(bot, msg, "请先私聊我绑定账号：/link <用户名> <密码>")
		return
	}

	loc := cstLoc()
	today := time.Now().In(loc).Format("2006-01-02")
	yesterday := time.Now().In(loc).Add(-24 * time.Hour).Format("2006-01-02")

	var c model.Checkin
	err = database.DB.Where("user_id = ?", user.ID).First(&c).Error
	isNew := err != nil
	if isNew {
		c = model.Checkin{UserID: user.ID, Streak: 1, TotalCheckIns: 1}
		c.LastCheckIn, _ = time.ParseInLocation("2006-01-02", today, loc)
		database.DB.Create(&c)
		reply(bot, msg, fmt.Sprintf("🎉 签到成功！\n连续 %d 天 · 总签到 %d 次\n(第一次签到，明天继续！)", c.Streak, c.TotalCheckIns))
		return
	}

	last := c.LastCheckIn.In(loc).Format("2006-01-02")
	if last == today {
		reply(bot, msg, fmt.Sprintf("☕ 今天已经签过啦\n连续 %d 天 · 总签到 %d 次\n明天 00:00 后再来～", c.Streak, c.TotalCheckIns))
		return
	}
	if last == yesterday {
		c.Streak++
	} else {
		c.Streak = 1
	}
	c.TotalCheckIns++
	c.LastCheckIn, _ = time.ParseInLocation("2006-01-02", today, loc)
	database.DB.Save(&c)

	// Check for 30-day milestone reward
	milestoneMsg := ""
	if c.Streak >= 30 && c.Streak%30 == 0 && c.Streak > c.LastRewardedStreak {
		milestoneMsg = fmt.Sprintf("\n\n🎁 达到 %d 天连续签到！自动赠送 30 天 VPN！群里有公告 🎉", c.Streak)
		awardMonthlyFreeSubscription(bot, &c, user)
	}

	// Compute rank
	var higher int64
	database.DB.Model(&model.Checkin{}).Where("streak > ?", c.Streak).Count(&higher)
	rank := higher + 1

	emoji := "✨"
	if c.Streak >= 30 {
		emoji = "🌟"
	} else if c.Streak >= 7 {
		emoji = "⭐"
	}

	reply(bot, msg, fmt.Sprintf("%s 签到成功！\n连续 %d 天 · 总签到 %d 次\n群内排名：第 %d 位%s", emoji, c.Streak, c.TotalCheckIns, rank, milestoneMsg))
}

// awardMonthlyFreeSubscription extends ExpiresAt by 30d, marks milestone rewarded,
// and sends celebration messages (group + private).
func awardMonthlyFreeSubscription(bot *tgbotapi.BotAPI, c *model.Checkin, user *model.User) {
	base := time.Now()
	if user.ExpiresAt.After(base) {
		base = user.ExpiresAt
	}
	newExpiry := base.AddDate(0, 0, 30)

	if err := database.DB.Model(user).Update("expires_at", newExpiry).Error; err != nil {
		log.Printf("telegram: reward expiry update failed for %s: %v", user.Username, err)
		return
	}

	// Record milestone on Checkin row so we don't re-reward the same streak
	database.DB.Model(c).Update("last_rewarded_streak", c.Streak)

	// Public celebration in group
	groupID := getGroupChatID()
	if groupID != 0 {
		groupMsg := tgbotapi.NewMessage(groupID, fmt.Sprintf(
			"🎉🎊 恭喜 @%s 连续签到 %d 天！\n"+
				"🎁 奖励：VPN 服务自动延长 30 天免费！\n"+
				"新到期日：%s\n\n"+
				"👏 给 TA 点个赞，下一个就是你！",
			user.Username, c.Streak, newExpiry.Format("2006-01-02"),
		))
		if _, err := bot.Send(groupMsg); err != nil {
			log.Printf("telegram: group reward msg failed: %v", err)
		}
	}

	log.Printf("telegram: awarded 30d free to %s (streak=%d, new_expiry=%s)",
		user.Username, c.Streak, newExpiry.Format("2006-01-02"))
}

// ─────────────────────────── /一签 ───────────────────────────────────────────

func handleFortune(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, tgID int64) {
	loc := cstLoc()
	today := time.Now().In(loc).Format("2006-01-02")
	// Deterministic hash
	seed := fmt.Sprintf("%d-%s", tgID, today)
	var h uint32
	for _, c := range seed {
		h = h*31 + uint32(c)
	}
	idx := int(h) % len(fortuneList)
	if idx < 0 {
		idx = -idx
	}
	reply(bot, msg, fortuneList[idx]+"\n\n（每日一签，明天再来）")
}

// ─────────────────────────── /笑话 ───────────────────────────────────────────

func handleJoke(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	reply(bot, msg, jokeList[rand.Intn(len(jokeList))])
}

// ─────────────────────────── /骰子 ───────────────────────────────────────────

func handleDice(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	dice := tgbotapi.NewDice(msg.Chat.ID)
	if _, err := bot.Send(dice); err != nil {
		log.Printf("telegram: dice send failed: %v", err)
	}
}

// ─────────────────────────── /成就 ───────────────────────────────────────────

func handleAchievements(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, tgID int64) {
	if !requirePrivate(bot, msg) {
		return
	}
	user, err := findUserByTelegramID(tgID)
	if err != nil {
		reply(bot, msg, "请先私聊我绑定账号：/link <用户名> <密码>")
		return
	}

	var totalBytes int64
	if s, ok := GetUserStat(user.Username); ok {
		totalBytes = s.TotalTX + s.TotalRX
	}

	var checkin model.Checkin
	database.DB.Where("user_id = ?", user.ID).First(&checkin)

	GB := int64(1024 * 1024 * 1024)
	subscribedDays := int(time.Since(user.CreatedAt).Hours() / 24)

	achievements := []struct {
		earned      bool
		icon, label string
	}{
		{true, "🎯", "首次订阅"},
		{totalBytes >= GB, "🥉", "累计 1 GB"},
		{totalBytes >= 10*GB, "🥈", "累计 10 GB"},
		{totalBytes >= 100*GB, "🥇", "累计 100 GB"},
		{totalBytes >= 500*GB, "🚀", "累计 500 GB"},
		{user.TrafficLimit == 0, "💎", "无限流量套餐"},
		{user.RuleAI && user.RuleStreaming && user.RuleChina && user.RuleAdBlock, "🌈", "全规则开启"},
		{checkin.Streak >= 7, "⭐", "连续签到 7 天"},
		{checkin.Streak >= 30, "🌟", "连续签到 30 天"},
		{subscribedDays >= 30, "📅", "订阅满 30 天"},
		{subscribedDays >= 365, "🏆", "订阅满 365 天"},
	}

	earned := []string{}
	locked := []string{}
	for _, a := range achievements {
		if a.earned {
			earned = append(earned, fmt.Sprintf("%s %s", a.icon, a.label))
		} else {
			locked = append(locked, fmt.Sprintf("🔒 %s", a.label))
		}
	}

	text := fmt.Sprintf("🏅 %s 的成就卡片\n\n", user.Username)
	text += "✅ 已解锁 (" + fmt.Sprintf("%d", len(earned)) + ")\n"
	if len(earned) > 0 {
		text += strings.Join(earned, "\n")
	} else {
		text += "（无）"
	}
	text += "\n\n🔓 待解锁 (" + fmt.Sprintf("%d", len(locked)) + ")\n"
	if len(locked) > 0 {
		text += strings.Join(locked, "\n")
	} else {
		text += "🎊 全部完成，你是最强 hy2 用户！"
	}
	reply(bot, msg, text)
}

// ─────────────────────────── new member welcome ──────────────────────────────

func handleNewMembers(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	defer func() { recover() }()
	// Only in the registered group
	groupID := getGroupChatID()
	if groupID == 0 || msg.Chat.ID != groupID {
		return
	}
	for _, m := range msg.NewChatMembers {
		if m.IsBot {
			continue
		}
		name := m.FirstName
		if name == "" {
			name = m.UserName
		}
		if name == "" {
			name = "新朋友"
		}
		text := fmt.Sprintf(
			"🎉✨ 欢迎新朋友 %s！\n\n"+
				"我是小卤蛋的 hy2 小助手 🤖💙\n"+
				"负责账号自助、订阅发放、群里唠嗑\n\n"+
				"━━━━━━━━━━━━━━\n"+
				"🚀 三步上手\n"+
				"1️⃣ 私聊我 @%s\n"+
				"2️⃣ /link <用户名> <密码> 绑定账号\n"+
				"3️⃣ /sub 拿订阅 + 二维码 📡\n"+
				"━━━━━━━━━━━━━━\n\n"+
				"🔧 账号类（私聊用）\n"+
				"· /sub — 获取订阅链接 + 二维码\n"+
				"· /quota — 查流量 / 到期日\n"+
				"· /成就 — 查看我的徽章墙 🏅\n"+
				"· /unlink — 解除当前 Telegram 绑定\n\n"+
				"💎 查套餐（任意地方）\n"+
				"· /plans — 查看所有套餐 · 开通联系管理员\n\n"+
				"🎮 互动 & 娱乐（任意地方）\n"+
				"· /签到 — 每日打卡，看连续天数和排行榜 📅\n"+
				"· /一签 — 今日运势 🎴\n"+
				"· /笑话 — 技术宅段子一发 😂\n"+
				"· /骰子 — 摇个 1-6 🎲\n\n"+
				"━━━━━━━━━━━━━━\n"+
				"📅 每天 09:00 我会在群里播报节点状态 + 小贴士\n"+
				"🏆 每周日 20:00 公布本周流量王\n"+
				"💬 群里随时 @ 我呼出帮助\n"+
				"🙋 有问题请 @ 管理员",
			name, bot.Self.UserName,
		)
		m2 := tgbotapi.NewMessage(msg.Chat.ID, text)
		if _, err := bot.Send(m2); err != nil {
			log.Printf("telegram: welcome send failed: %v", err)
		}
		notifyAdminNewMember(bot, msg, m)
	}
}

func notifyAdminNewMember(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, member tgbotapi.User) {
	adminID, ok := getAdminTelegramID()
	if !ok {
		log.Printf("telegram: new member admin notify skipped — admin telegram_id not bound")
		return
	}

	displayName := strings.TrimSpace(strings.Join([]string{member.FirstName, member.LastName}, " "))
	if displayName == "" {
		displayName = "未设置昵称"
	}
	username := "无"
	if member.UserName != "" {
		username = "@" + member.UserName
	}
	joinedAt := time.Now().In(cstLoc()).Format("2006-01-02 15:04:05")
	text := fmt.Sprintf(
		"🚪 新成员进群\n\n"+
			"群组：%s\n"+
			"昵称：%s\n"+
			"用户名：%s\n"+
			"Telegram ID：%d\n"+
			"时间：%s\n\n"+
			"需要确认身份的话，可以直接去群里看一眼。",
		msg.Chat.Title,
		displayName,
		username,
		member.ID,
		joinedAt,
	)
	notice := tgbotapi.NewMessage(adminID, text)
	if _, err := bot.Send(notice); err != nil {
		log.Printf("telegram: new member admin notify failed: %v", err)
	}
}

// ─────────────────────────── @mention helpers ────────────────────────────────

func isBotMentioned(msg *tgbotapi.Message, botUsername string) bool {
	// Only relevant in group chats (private chats always go through commands).
	if msg.Chat.Type != "group" && msg.Chat.Type != "supergroup" {
		return false
	}
	needle := "@" + strings.ToLower(botUsername)
	// Plain @BotUsername in text or caption — that's the only true positive.
	// Don't trigger on @other_user (entity.Type == "mention" matches any @username).
	if strings.Contains(strings.ToLower(msg.Text), needle) {
		return true
	}
	if strings.Contains(strings.ToLower(msg.Caption), needle) {
		return true
	}
	// "text_mention" entities target a specific User without @username in the text.
	// Match only when the mentioned user IS the bot.
	for _, ent := range msg.Entities {
		if ent.Type == "text_mention" && ent.User != nil &&
			strings.EqualFold(ent.User.UserName, botUsername) {
			return true
		}
	}
	for _, ent := range msg.CaptionEntities {
		if ent.Type == "text_mention" && ent.User != nil &&
			strings.EqualFold(ent.User.UserName, botUsername) {
			return true
		}
	}
	return false
}

func handleMention(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	defer func() { recover() }()
	text := "嗨 👋 我是小卤蛋的 hy2 小助手 🤖💙\n" +
		"很高兴认识你！下面是我能做的事：\n\n" +
		"━━━━━━━━━━━━━━\n" +
		"🔐 账号自助（私聊我）\n" +
		"不打扰管理员，直接绑定账号、取订阅、查流量、看到期日，一气呵成。\n" +
		"· /link · /unlink · /sub · /quota · /成就\n\n" +
		"💎 套餐查看\n" +
		"随时在哪都能 /plans 看所有套餐详情。\n\n" +
		"🌅 每日节点播报\n" +
		"每天 09:00 在群里推送：节点健康、实时在线、24 小时流量、昨日流量王 + 一条使用贴士。\n\n" +
		"🏆 本周流量王\n" +
		"每周日 20:00 公布近 7 天流量前 3 名 🥇🥈🥉。\n\n" +
		"🎮 陪你唠嗑\n" +
		"· /签到 连续打卡看排名 📅\n" +
		"· /一签 今日运势 🎴\n" +
		"· /笑话 技术宅段子 😂\n" +
		"· /骰子 摇个 1-6 🎲\n\n" +
		"🏅 成就系统\n" +
		"私聊我发 /成就 查看自己的徽章墙（1 GB、100 GB、签到 7 天、订阅满 1 年……）。\n" +
		"━━━━━━━━━━━━━━\n\n" +
		"👉 点我头像开私聊 @" + bot.Self.UserName + " 最方便\n" +
		"💬 群里随时 @ 我都可以呼出本介绍\n" +
		"🙋 有问题请 @ 管理员"
	m := tgbotapi.NewMessage(msg.Chat.ID, text)
	m.ReplyToMessageID = msg.MessageID
	if _, err := bot.Send(m); err != nil {
		log.Printf("telegram: mention reply failed: %v", err)
	}
}

// ─────────────────────────── daily scheduled post ────────────────────────────

func formatDailyPost() string {
	snap := GetTrafficSnapshot()
	buckets := GetUserBuckets()

	healthy, total := 0, len(snap)
	online := 0
	for _, n := range snap {
		if n.Healthy {
			healthy++
		}
		online += n.Online
	}

	// Count active vs total users
	var allUsers []model.User
	database.DB.Find(&allUsers)
	activeUsers, totalUsers := 0, len(allUsers)
	for _, u := range allUsers {
		if u.IsActive() {
			activeUsers++
		}
	}

	// traffic_logs are cumulative counter snapshots. Use ordered deltas instead
	// of raw SUM; otherwise every snapshot is counted again.
	tx24, rx24, _ := AggregateTrafficDelta(database.DB, time.Now().Add(-24*time.Hour), time.Now())

	// Yesterday's traffic king (feature F)
	loc := cstLoc()
	nowLoc := time.Now().In(loc)
	startY := time.Date(nowLoc.Year(), nowLoc.Month(), nowLoc.Day()-1, 0, 0, 0, 0, loc)
	endY := startY.Add(24 * time.Hour)
	topRows, _ := TrafficDeltas(database.DB, startY, endY, 1)
	yesterdayKing := ""
	if len(topRows) > 0 && topRows[0].Total > 0 {
		yesterdayKing = fmt.Sprintf("\n🏆 昨日流量王：%s %s", topRows[0].Username, humanBytes(topRows[0].Total))
	}

	nodeStatus := "✅ 全线正常"
	if dead := len(buckets.DeadNodes); dead > 0 {
		nodeStatus = fmt.Sprintf("⚠️ %d 个节点离线", dead)
	}

	tip := dailyTips[rand.Intn(len(dailyTips))]
	highlight := serviceHighlights[rand.Intn(len(serviceHighlights))]

	return fmt.Sprintf(
		"🌅 早安！hy2board 今日播报\n\n"+
			"📡 节点状态：%d/%d 健康 · %s\n"+
			"🟢 实时在线：%d 连接\n"+
			"📊 24 小时流量：↑ %s · ↓ %s%s\n"+
			"👥 活跃用户：%d / %d\n\n"+
			"━━━━━━━━━━━━━━\n"+
			"✨ 今日亮点\n\n%s\n"+
			"━━━━━━━━━━━━━━\n"+
			"💡 小贴士\n\n%s\n\n"+
			"━━━━━━━━━━━━━━\n"+
			"⚡ 还没上车？私聊我 /plans 看套餐 💎\n"+
			"📊 老朋友 /quota 查流量 · /sub 拿订阅",
		healthy, total, nodeStatus,
		online,
		humanBytes(tx24), humanBytes(rx24), yesterdayKing,
		activeUsers, totalUsers,
		highlight, tip,
	)
}

// StartDailyPoster schedules a daily post at 09:00 Asia/Shanghai.
func StartDailyPoster() {
	go func() {
		loc := cstLoc()
		for {
			now := time.Now().In(loc)
			next := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, loc)
			if !next.After(now) {
				next = next.Add(24 * time.Hour)
			}
			time.Sleep(time.Until(next))
			sendDailyPost()
		}
	}()
}

func sendDailyPost() {
	groupID := getGroupChatID()
	if groupID == 0 {
		log.Printf("telegram: daily post skipped — no group registered")
		return
	}
	if !config.C.HasTelegram() {
		return
	}
	bot, err := tgbotapi.NewBotAPI(config.C.Telegram.BotToken)
	if err != nil {
		return
	}
	text := formatDailyPost()
	if _, err := bot.Send(tgbotapi.NewMessage(groupID, text)); err != nil {
		log.Printf("telegram: daily post send failed: %v", err)
	}
}

// SendTestDailyPost sends a test daily post to the registered group immediately.
func SendTestDailyPost(ctx context.Context) error {
	groupID := getGroupChatID()
	if groupID == 0 {
		return fmt.Errorf("no group registered")
	}
	bot, err := tgbotapi.NewBotAPI(config.C.Telegram.BotToken)
	if err != nil {
		return err
	}
	_, err = bot.Send(tgbotapi.NewMessage(groupID, "🧪 测试播报\n\n"+formatDailyPost()))
	return err
}

// ─────────────────────────── weekly leaderboard (feature E) ──────────────────

// StartWeeklyLeaderboard fires every Sunday 20:00 Asia/Shanghai.
func StartWeeklyLeaderboard() {
	go func() {
		loc := cstLoc()
		for {
			now := time.Now().In(loc)
			// Find next Sunday 20:00 CST
			daysUntilSunday := (7 - int(now.Weekday())) % 7
			if daysUntilSunday == 0 && now.Hour() >= 20 {
				daysUntilSunday = 7
			}
			next := time.Date(now.Year(), now.Month(), now.Day()+daysUntilSunday, 20, 0, 0, 0, loc)
			time.Sleep(time.Until(next))
			sendWeeklyLeaderboard()
		}
	}()
}

func sendWeeklyLeaderboard() {
	groupID := getGroupChatID()
	if groupID == 0 || !config.C.HasTelegram() {
		return
	}

	rows, err := TrafficDeltas(database.DB, time.Now().Add(-7*24*time.Hour), time.Now(), 3)
	if err != nil {
		log.Printf("telegram: weekly leaderboard query failed: %v", err)
		return
	}

	bot, err := tgbotapi.NewBotAPI(config.C.Telegram.BotToken)
	if err != nil {
		return
	}

	text := "🏆 本周流量王（最近 7 天）\n\n"
	medals := []string{"🥇", "🥈", "🥉"}
	if len(rows) == 0 {
		text += "（本周暂无流量记录）"
	} else {
		for i, r := range rows {
			text += fmt.Sprintf("%s %s — %s\n", medals[i], r.Username, humanBytes(r.Total))
		}
	}
	text += "\n👉 想上榜？/sub 获取订阅，/plans 升级套餐"

	bot.Send(tgbotapi.NewMessage(groupID, text))
}

// ─────────────────────────── activity announcement ───────────────────────────

const keyAnnouncementMsgID = "activity_announcement_message_id"

const activityAnnouncementZH = `📣 每月签到活动上线！

🎯 活动规则
连续签到满 30 天，免费送 30 天 VPN 服务！
（每达到 30 天 / 60 天 / 90 天…都自动奖励 30 天）

🗓 如何参与
每天私聊我 @%s 发送 /签到
连续不间断即可累积天数

💡 小贴士
· 签到时间：每天 00:00 (北京时间) 刷新
· 断签后重新计数，不用担心，从 1 开始
· 奖励自动发放到你的订阅到期日
· 在群里 @ 我或私聊 /成就 查看自己的进度

🏁 第一个达成 30 天的朋友，群里等你！`

func PostAndPinActivity() error {
	if !config.C.HasTelegram() {
		return fmt.Errorf("telegram not configured")
	}
	groupID := getGroupChatID()
	if groupID == 0 {
		return fmt.Errorf("no group registered")
	}
	bot, err := tgbotapi.NewBotAPI(config.C.Telegram.BotToken)
	if err != nil {
		return err
	}

	text := fmt.Sprintf(activityAnnouncementZH, bot.Self.UserName)
	sent, err := bot.Send(tgbotapi.NewMessage(groupID, text))
	if err != nil {
		return fmt.Errorf("send: %w", err)
	}

	// Pin the message
	pinCfg := tgbotapi.PinChatMessageConfig{
		ChatID:              groupID,
		MessageID:           sent.MessageID,
		DisableNotification: false,
	}
	if _, err := bot.Request(pinCfg); err != nil {
		log.Printf("telegram: pin failed (continuing): %v", err)
	}

	setSetting(keyAnnouncementMsgID, fmt.Sprintf("%d", sent.MessageID))
	return nil
}

// ─────────────────────────── /plans ──────────────────────────────────────────

func handlePlans(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	var plans []model.Plan
	if err := database.DB.Order("sort_order asc, id asc").Find(&plans).Error; err != nil {
		reply(bot, msg, "查询失败："+err.Error())
		return
	}
	if len(plans) == 0 {
		reply(bot, msg, "暂无可选套餐，请联系管理员。")
		return
	}

	var sb strings.Builder
	sb.WriteString("💎 套餐列表\n\n")
	for _, p := range plans {
		limit := "无限流量"
		if p.TrafficLimit > 0 {
			limit = humanBytes(p.TrafficLimit)
		}
		dur := fmt.Sprintf("%d 天", p.DurationDays)
		if p.DurationDays == 0 {
			dur = "永久"
		}

		rules := []string{}
		if p.RuleAI {
			rules = append(rules, "AI")
		}
		if p.RuleStreaming {
			rules = append(rules, "流媒体")
		}
		if p.RuleChina {
			rules = append(rules, "国内直连")
		}
		if p.RuleAdBlock {
			rules = append(rules, "广告拦截")
		}
		ruleStr := "—"
		if len(rules) > 0 {
			ruleStr = strings.Join(rules, " · ")
		}

		reset := "手动"
		if p.AutoReset {
			reset = "每月自动重置"
		}

		sb.WriteString(fmt.Sprintf("📦 %s\n", p.Name))
		sb.WriteString(fmt.Sprintf("   流量：%s\n", limit))
		sb.WriteString(fmt.Sprintf("   周期：%s\n", dur))
		sb.WriteString(fmt.Sprintf("   规则：%s\n", ruleStr))
		sb.WriteString(fmt.Sprintf("   重置：%s\n\n", reset))
	}
	sb.WriteString("💬 开通或询价请联系管理员。")
	reply(bot, msg, sb.String())
}

// ─────────────────────────── helpers ─────────────────────────────────────────

func findUserByTelegramID(tgID int64) (*model.User, error) {
	var user model.User
	if err := database.DB.Where("telegram_id = ?", tgID).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func getAdminTelegramID() (int64, bool) {
	username := strings.TrimSpace(config.C.Admin.Username)
	if username == "" {
		return 0, false
	}
	var user model.User
	if err := database.DB.Where("username = ? AND telegram_id <> 0", username).First(&user).Error; err != nil {
		return 0, false
	}
	return user.TelegramID, true
}

func humanBytes(b int64) string {
	if b == 0 {
		return "0 B"
	}
	units := []string{"B", "KB", "MB", "GB", "TB"}
	i := 0
	v := float64(b)
	for v >= 1024 && i < len(units)-1 {
		v /= 1024
		i++
	}
	return fmt.Sprintf("%.2f %s", v, units[i])
}

func reply(bot *tgbotapi.BotAPI, src *tgbotapi.Message, text string) {
	m := tgbotapi.NewMessage(src.Chat.ID, text)
	m.ReplyToMessageID = src.MessageID
	if _, err := bot.Send(m); err != nil {
		log.Printf("telegram: send reply failed: %v", err)
	}
}
