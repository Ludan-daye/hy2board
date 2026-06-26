import { useEffect, useState } from "react"
import { Download, ExternalLink, Apple, Smartphone, Monitor, Terminal } from "lucide-react"

interface DownloadItem {
  name: string
  platform: "windows" | "android" | "macos" | "ios" | "linux"
  filename?: string
  external_url?: string
  homepage?: string
  note?: string
  available: boolean
  size_bytes?: number
}

const platformLabels: Record<string, { label: string; icon: any; cls: string }> = {
  windows: { label: "Windows",  icon: Monitor,    cls: "text-sky-400" },
  android: { label: "Android",  icon: Smartphone, cls: "text-green-400" },
  macos:   { label: "macOS",    icon: Apple,      cls: "text-zinc-300" },
  ios:     { label: "iOS",      icon: Apple,      cls: "text-zinc-300" },
  linux:   { label: "Linux",    icon: Terminal,   cls: "text-amber-400" },
}

function fmtBytes(n?: number): string {
  if (!n) return ""
  const u = ["B", "KB", "MB", "GB"]
  let i = 0
  let v = n
  while (v >= 1024 && i < u.length - 1) { v /= 1024; i++ }
  return `${v.toFixed(1)} ${u[i]}`
}

export default function Downloads() {
  const [items, setItems] = useState<DownloadItem[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetch("/api/downloads")
      .then(r => r.json())
      .then(data => { setItems(data || []); setLoading(false) })
      .catch(() => setLoading(false))
  }, [])

  const grouped: Record<string, DownloadItem[]> = {}
  for (const it of items) {
    grouped[it.platform] = grouped[it.platform] || []
    grouped[it.platform].push(it)
  }

  const order = ["windows", "macos", "ios", "android", "linux"]

  return (
    <div className="min-h-screen bg-surface text-ink">
      <header className="border-b border-zinc-800 px-6 py-4 flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold flex items-center gap-2">
            <Download size={20} /> Client Downloads
          </h1>
          <p className="text-xs text-zinc-500 mt-1">下载推荐的代理客户端，支持 Hysteria 2 协议</p>
        </div>
        <a href="/" className="text-xs text-zinc-500 hover:text-ink">← 返回</a>
      </header>

      <main className="max-w-5xl mx-auto p-6">
        <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-4 mb-6 text-sm text-zinc-400">
          <p className="font-medium text-zinc-300 mb-1">📌 使用提示</p>
          <ul className="list-disc list-inside space-y-1 text-xs">
            <li><span className="text-zinc-300">订阅链接</span>：登录 admin 后台 → Users 页面 → 找到对应用户 → 点 🔗 复制订阅地址</li>
            <li><span className="text-zinc-300">客户端选哪个</span>：Windows/Android 选 NekoBox 或 V2RayN；macOS 选 V2Box 或 ClashX Meta；iOS 选 Shadowrocket / Stash</li>
            <li><span className="text-zinc-300">订阅格式</span>：v2ray 系列用 <code className="bg-surface px-1 rounded">?format=v2ray</code>；Clash/Mihomo 用 <code className="bg-surface px-1 rounded">?format=clash</code>；Surge/Loon 用 <code className="bg-surface px-1 rounded">?format=surge</code>；Shadowrocket 用 <code className="bg-surface px-1 rounded">?format=shadowrocket-conf</code></li>
          </ul>
        </div>

        {loading && <div className="text-center text-zinc-500">Loading...</div>}

        {!loading && order.filter(p => grouped[p]?.length).map(p => {
          const meta = platformLabels[p]
          const Icon = meta.icon
          return (
            <section key={p} className="mb-8">
              <h2 className={`text-sm font-medium mb-3 flex items-center gap-2 ${meta.cls}`}>
                <Icon size={16} /> {meta.label}
              </h2>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                {grouped[p].map((it, i) => (
                  <div key={i} className="bg-zinc-900 border border-zinc-800 rounded-xl p-4 flex flex-col">
                    <div className="flex items-start justify-between mb-2">
                      <div>
                        <p className="font-medium">{it.name}</p>
                        {it.note && <p className="text-xs text-zinc-500 mt-1">{it.note}</p>}
                      </div>
                      {it.size_bytes ? <span className="text-xs text-zinc-600 font-mono">{fmtBytes(it.size_bytes)}</span> : null}
                    </div>
                    <div className="mt-auto flex gap-2 pt-3">
                      {it.external_url ? (
                        <a href={it.external_url} target="_blank" rel="noreferrer"
                           className="flex items-center justify-center gap-1.5 flex-1 px-3 py-2 bg-clay text-white rounded-lg text-xs font-medium hover:bg-clay-hover">
                          <ExternalLink size={12} /> App Store
                        </a>
                      ) : it.available ? (
                        <a href={`/dl/${it.filename}`}
                           className="flex items-center justify-center gap-1.5 flex-1 px-3 py-2 bg-clay text-white rounded-lg text-xs font-medium hover:bg-clay-hover">
                          <Download size={12} /> 下载 ({fmtBytes(it.size_bytes)})
                        </a>
                      ) : (
                        <button disabled
                                className="flex-1 px-3 py-2 bg-zinc-800 text-zinc-600 rounded-lg text-xs cursor-not-allowed">
                          待管理员上传
                        </button>
                      )}
                      {it.homepage && (
                        <a href={it.homepage} target="_blank" rel="noreferrer"
                           className="flex items-center justify-center px-3 py-2 bg-zinc-800 hover:bg-zinc-700 text-zinc-300 rounded-lg text-xs"
                           title="项目主页">
                          <ExternalLink size={12} />
                        </a>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            </section>
          )
        })}
      </main>

      <footer className="border-t border-zinc-800 px-6 py-4 text-center text-xs text-zinc-600">
        所有下载文件托管在本服务器；iOS 客户端需在对应区 App Store 购买。
      </footer>
    </div>
  )
}
