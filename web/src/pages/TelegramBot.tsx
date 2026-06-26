import { useEffect, useState } from "react"
import api from "@/api/client"
import { AlertTriangle, Bot, CalendarClock, CheckCircle2, Megaphone, RefreshCw, Send, ShieldCheck, Users } from "lucide-react"

interface TelegramStatus {
  enabled: boolean
  bot_configured: boolean
  group_registered: boolean
  group_chat_id: number
  admin_username: string
  admin_bound: boolean
  admin_telegram_id: number
  daily_post_time: string
  weekly_leaderboard_time: string
}

const emptyStatus: TelegramStatus = {
  enabled: false,
  bot_configured: false,
  group_registered: false,
  group_chat_id: 0,
  admin_username: "",
  admin_bound: false,
  admin_telegram_id: 0,
  daily_post_time: "",
  weekly_leaderboard_time: "",
}

function StatePill({ ok, label }: { ok: boolean; label: string }) {
  return (
    <span className={`inline-flex items-center gap-1.5 px-2 py-1 rounded-md border text-xs ${
      ok ? "bg-green-500/10 text-green-300 border-green-500/20" : "bg-amber-500/10 text-amber-300 border-amber-500/20"
    }`}>
      {ok ? <CheckCircle2 size={13} /> : <AlertTriangle size={13} />}
      {label}
    </span>
  )
}

function InfoTile({ icon: Icon, title, value, ok }: { icon: any; title: string; value: string; ok: boolean }) {
  return (
    <div className="bg-zinc-900 border border-zinc-800 rounded-lg p-4 min-h-[116px]">
      <div className="flex items-center justify-between mb-4">
        <div className="w-9 h-9 rounded-lg flex items-center justify-center bg-ink/5 text-zinc-300">
          <Icon size={16} />
        </div>
        <StatePill ok={ok} label={ok ? "OK" : "待处理"} />
      </div>
      <div className="text-xs text-zinc-500 mb-1">{title}</div>
      <div className="text-sm font-medium text-zinc-100 break-words">{value}</div>
    </div>
  )
}

export default function TelegramBot() {
  const [status, setStatus] = useState<TelegramStatus>(emptyStatus)
  const [loading, setLoading] = useState(false)
  const [busy, setBusy] = useState("")
  const [message, setMessage] = useState("")
  const [error, setError] = useState("")

  const load = async () => {
    setLoading(true)
    try {
      const r = await api.get("/admin/tg/status")
      setStatus(r.data || emptyStatus)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { load() }, [])

  const runAction = async (key: string, endpoint: string, success: string) => {
    setBusy(key)
    setMessage("")
    setError("")
    try {
      await api.post(endpoint)
      setMessage(success)
      load()
    } catch (e: any) {
      setError(e.response?.data?.error || e.message || "Request failed")
    } finally {
      setBusy("")
    }
  }

  const botReady = status.enabled && status.bot_configured
  const allReady = botReady && status.group_registered && status.admin_bound

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-xl font-semibold flex items-center gap-2">
          <Bot size={18} /> Telegram Bot
          <span className="text-xs text-zinc-500 font-normal ml-3">{allReady ? "ready" : "needs attention"}</span>
        </h2>
        <button
          onClick={load}
          disabled={loading}
          className="flex items-center gap-2 px-3 py-1.5 bg-clay text-white rounded-lg text-sm font-medium hover:bg-clay-hover disabled:opacity-60"
        >
          <RefreshCw size={14} className={loading ? "animate-spin" : ""} /> Refresh
        </button>
      </div>

      {message && <div className="mb-4 px-4 py-2 bg-green-500/10 border border-green-500/20 rounded-lg text-green-300 text-sm">{message}</div>}
      {error && <div className="mb-4 px-4 py-2 bg-red-500/10 border border-red-500/20 rounded-lg text-red-300 text-sm">{error}</div>}

      <div className="grid grid-cols-1 lg:grid-cols-4 gap-3 mb-4">
        <InfoTile icon={Bot} title="Bot" value={botReady ? "Enabled / token configured" : "Check config.yaml"} ok={botReady} />
        <InfoTile icon={ShieldCheck} title="Admin DM" value={status.admin_bound ? `${status.admin_username} · ${status.admin_telegram_id}` : `${status.admin_username || "admin"} not linked`} ok={status.admin_bound} />
        <InfoTile icon={Users} title="Group" value={status.group_registered ? `${status.group_chat_id}` : "No group registered"} ok={status.group_registered} />
        <InfoTile icon={CalendarClock} title="Schedule" value={`${status.daily_post_time || "-"} / ${status.weekly_leaderboard_time || "-"}`} ok={botReady} />
      </div>

      <div className="grid grid-cols-1 xl:grid-cols-[1fr_360px] gap-4">
        <div className="bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden">
          <div className="px-4 py-3 border-b border-zinc-800 flex items-center justify-between">
            <div>
              <h3 className="text-sm font-medium text-ink">Actions</h3>
              <p className="text-xs text-zinc-500 mt-0.5">Run live Telegram checks from the admin panel.</p>
            </div>
            <StatePill ok={allReady} label={allReady ? "链路完整" : "需要补配置"} />
          </div>
          <div className="divide-y divide-zinc-800">
            <button
              onClick={() => runAction("admin", "/admin/tg/test-admin-notice", "Admin test notice sent")}
              disabled={busy !== "" || !botReady || !status.admin_bound}
              className="w-full px-4 py-4 flex items-center justify-between text-left hover:bg-surface/[0.03] disabled:opacity-40"
            >
              <span className="flex items-center gap-3">
                <Send size={16} className="text-clay" />
                <span>
                  <span className="block text-sm text-zinc-100">Send admin DM test</span>
                  <span className="block text-xs text-zinc-500 mt-0.5">New-member private alert path</span>
                </span>
              </span>
              <span className="text-xs text-zinc-500">{busy === "admin" ? "Sending..." : "Run"}</span>
            </button>

            <button
              onClick={() => runAction("daily", "/admin/tg/test-post", "Daily test post sent")}
              disabled={busy !== "" || !botReady || !status.group_registered}
              className="w-full px-4 py-4 flex items-center justify-between text-left hover:bg-surface/[0.03] disabled:opacity-40"
            >
              <span className="flex items-center gap-3">
                <CalendarClock size={16} className="text-green-300" />
                <span>
                  <span className="block text-sm text-zinc-100">Send daily test post</span>
                  <span className="block text-xs text-zinc-500 mt-0.5">Group node status broadcast</span>
                </span>
              </span>
              <span className="text-xs text-zinc-500">{busy === "daily" ? "Sending..." : "Run"}</span>
            </button>

            <button
              onClick={() => runAction("activity", "/admin/tg/announce-activity", "Activity announcement sent and pinned")}
              disabled={busy !== "" || !botReady || !status.group_registered}
              className="w-full px-4 py-4 flex items-center justify-between text-left hover:bg-surface/[0.03] disabled:opacity-40"
            >
              <span className="flex items-center gap-3">
                <Megaphone size={16} className="text-clay" />
                <span>
                  <span className="block text-sm text-zinc-100">Post activity announcement</span>
                  <span className="block text-xs text-zinc-500 mt-0.5">Monthly check-in reward notice</span>
                </span>
              </span>
              <span className="text-xs text-zinc-500">{busy === "activity" ? "Sending..." : "Run"}</span>
            </button>
          </div>
        </div>

        <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-4">
          <h3 className="text-sm font-medium text-ink mb-3">Setup</h3>
          <div className="space-y-4 text-sm">
            <div>
              <div className="text-xs text-zinc-500 mb-1">Admin binding</div>
              <code className="block text-xs bg-surface border border-zinc-800 rounded-lg px-3 py-2 text-zinc-300 break-words">/link {status.admin_username || "admin"} &lt;面板密码&gt;</code>
            </div>
            <div>
              <div className="text-xs text-zinc-500 mb-1">Group registration</div>
              <code className="block text-xs bg-surface border border-zinc-800 rounded-lg px-3 py-2 text-zinc-300">/register_group</code>
            </div>
            <div className="pt-2 border-t border-zinc-800 text-xs leading-relaxed text-zinc-500">
              Bot 必须在群里，且能看到新成员事件。管理员账号绑定后，新人入群会自动私发给管理员。
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
