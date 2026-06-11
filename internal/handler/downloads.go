package handler

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// downloadEntry describes one client app shown on /downloads.
// Filename is the expected file in /app/downloads/. ExternalURL is set for
// iOS apps (App Store links — can't be self-hosted).
type downloadEntry struct {
	Name        string `json:"name"`
	Platform    string `json:"platform"` // windows | android | macos | ios | linux
	Filename    string `json:"filename,omitempty"`
	ExternalURL string `json:"external_url,omitempty"`
	Homepage    string `json:"homepage,omitempty"`
	Note        string `json:"note,omitempty"`
	Available   bool   `json:"available"` // true if the file exists in /app/downloads/
	SizeBytes   int64  `json:"size_bytes,omitempty"`
}

var downloadCatalog = []downloadEntry{
	// Windows — Clash Verge first (recommended)
	{Name: "Clash Verge", Platform: "windows", Filename: "clash-verge-windows.exe", Homepage: "https://github.com/clash-verge-rev/clash-verge-rev", Note: "★ 推荐：Mihomo 内核 GUI，订阅选 Clash 格式"},
	{Name: "V2RayN", Platform: "windows", Filename: "v2rayn-windows.zip", Homepage: "https://github.com/2dust/v2rayN", Note: "经典 Windows 客户端，支持 hysteria2"},
	{Name: "NekoBox", Platform: "windows", Filename: "nekobox-windows.zip", Homepage: "https://github.com/MatsuriDayo/nekoray", Note: "通用代理工具，多协议支持"},
	{Name: "Hiddify", Platform: "windows", Filename: "hiddify-windows.exe", Homepage: "https://hiddify.com", Note: "现代化 GUI，开箱即用"},
	{Name: "Karing", Platform: "windows", Filename: "karing-windows.exe", Homepage: "https://karing.app", Note: "基于 sing-box 的现代客户端"},

	// Android — CMFA first (Clash Verge has no Android version)
	{Name: "Clash Meta (CMFA)", Platform: "android", Filename: "clash-meta-android.apk", Homepage: "https://github.com/MetaCubeX/ClashMetaForAndroid", Note: "★ Android 推荐：Clash Verge 没 Android 版，用这个（同 Mihomo 内核）"},
	{Name: "NekoBox", Platform: "android", Filename: "nekobox-android.apk", Homepage: "https://github.com/MatsuriDayo/NekoBoxForAndroid", Note: "Android 上的多协议代理"},
	{Name: "V2RayNG", Platform: "android", Filename: "v2rayng-android.apk", Homepage: "https://github.com/2dust/v2rayNG", Note: "老牌 V2Ray Android 客户端"},
	{Name: "Hiddify", Platform: "android", Filename: "hiddify-android.apk", Homepage: "https://hiddify.com", Note: "Hiddify Android"},
	{Name: "Karing", Platform: "android", Filename: "karing-android.apk", Homepage: "https://karing.app", Note: "基于 sing-box 的现代客户端"},

	// macOS — Clash Verge first (recommended)
	{Name: "Clash Verge (Apple Silicon)", Platform: "macos", Filename: "clash-verge-macos-arm.dmg", Homepage: "https://github.com/clash-verge-rev/clash-verge-rev", Note: "★ 推荐：M1/M2/M3 Mac，Mihomo 内核"},
	{Name: "Clash Verge (Intel)", Platform: "macos", Filename: "clash-verge-macos-intel.dmg", Homepage: "https://github.com/clash-verge-rev/clash-verge-rev", Note: "★ 推荐：Intel 芯片 Mac，Mihomo 内核"},
	{Name: "V2Box", Platform: "macos", Filename: "v2box-macos.dmg", Homepage: "https://v2box.app", Note: "Mac 上的免费多协议客户端"},
	{Name: "Hiddify", Platform: "macos", Filename: "hiddify-macos.dmg", Homepage: "https://hiddify.com", Note: "Hiddify macOS"},
	{Name: "ClashX Meta", Platform: "macos", Filename: "clashx-meta-macos.zip", Homepage: "https://github.com/MetaCubeX/ClashX.Meta", Note: "Mihomo 内核，订阅链接选 Clash 格式"},
	{Name: "Karing", Platform: "macos", Filename: "karing-macos.dmg", Homepage: "https://karing.app", Note: "基于 sing-box 的现代客户端"},

	// Linux — Clash Verge first (recommended)
	{Name: "Clash Verge", Platform: "linux", Filename: "clash-verge-linux.deb", Homepage: "https://github.com/clash-verge-rev/clash-verge-rev", Note: "★ 推荐：Debian/Ubuntu .deb 包，Mihomo 内核"},
	{Name: "Hiddify", Platform: "linux", Filename: "hiddify-linux.AppImage", Homepage: "https://hiddify.com", Note: "Hiddify Linux AppImage"},
	{Name: "NekoBox", Platform: "linux", Filename: "nekobox-linux.tar.gz", Homepage: "https://github.com/MatsuriDayo/nekoray", Note: "NekoBox Linux"},

	// iOS — App Store only (can't self-host)
	{Name: "Shadowrocket", Platform: "ios", ExternalURL: "https://apps.apple.com/app/id932747118", Note: "需美区/海外 Apple ID 购买，¥2.99 美元一次性"},
	{Name: "Stash", Platform: "ios", ExternalURL: "https://apps.apple.com/app/id1596063349", Note: "美观的 Mihomo 客户端，订阅选 Clash 格式"},
	{Name: "Surge", Platform: "ios", ExternalURL: "https://apps.apple.com/app/id1442620678", Note: "高级用户首选，价格较高"},
	{Name: "Loon", Platform: "ios", ExternalURL: "https://apps.apple.com/app/id1373567447", Note: "类似 Surge，订阅选 Surge 格式"},
}

const downloadDir = "/app/downloads"

// ListDownloads is public (no auth). Returns the catalog with availability +
// size for each self-hosted file.
func ListDownloads(c *gin.Context) {
	out := make([]downloadEntry, 0, len(downloadCatalog))
	for _, e := range downloadCatalog {
		if e.Filename != "" {
			info, err := os.Stat(filepath.Join(downloadDir, e.Filename))
			if err == nil && !info.IsDir() {
				e.Available = true
				e.SizeBytes = info.Size()
			}
		} else if e.ExternalURL != "" {
			e.Available = true // iOS App Store link is always "available"
		}
		out = append(out, e)
	}
	c.JSON(http.StatusOK, out)
}
