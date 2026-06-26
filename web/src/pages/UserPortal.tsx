import { copyToClipboard } from "@/lib/clipboard"
import { useEffect, useState } from "react"
import api from "@/api/client"
import { ArrowUp, ArrowDown, Copy, Check, LogOut, Server, Clock, Shield, Activity, Download } from "lucide-react"
import { PieChart, Pie, Cell, Tooltip, ResponsiveContainer, BarChart, Bar, XAxis, YAxis, CartesianGrid, AreaChart, Area } from "recharts"

interface NodeTraffic { node: string; tx: number; rx: number }
interface UserInfo {
  username: string; sub_token: string
  traffic_limit: number; traffic_used: number
  traffic_tx: number; traffic_rx: number
  speed_tx?: number; speed_rx?: number
  expires_at: string; enabled: boolean; active: boolean
  nodes: NodeTraffic[]
  history?: { time: string; speed_tx: number; speed_rx: number }[]
}
interface HistoryPoint { time: string; tx: number; rx: number }


function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B"
  const k = 1024
  const sizes = ["B", "KB", "MB", "GB", "TB"]
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return (bytes / Math.pow(k, i)).toFixed(1) + " " + sizes[i]
}

export default function UserPortal() {
  const [info, setInfo] = useState<UserInfo | null>(null)
  const [copied, setCopied] = useState("")
  const [history, setHistory] = useState<HistoryPoint[]>([])

  const load = () => {
    api.get("/user/me").then((res) => {
      const d = res.data as UserInfo
      setInfo(d)
      if (Array.isArray(d.history)) {
        const mapped: HistoryPoint[] = d.history.slice(-30).map(p => ({
          time: new Date(p.time).toLocaleTimeString("en-US", { hour12: false, hour: "2-digit", minute: "2-digit", second: "2-digit" }),
          tx: p.speed_tx,
          rx: p.speed_rx,
        }))
        setHistory(mapped)
      }
    })
  }

  useEffect(() => {
    load()
    const timer = setInterval(load, 10000)
    return () => clearInterval(timer)
  }, [])

  const logout = () => { localStorage.removeItem("token"); localStorage.removeItem("role"); window.location.href = "/login" }
  const copyText = (text: string, label: string) => { copyToClipboard(text); setCopied(label); setTimeout(() => setCopied(""), 2000) }

  if (!info) return <div className="min-h-screen bg-surface flex items-center justify-center text-zinc-500">Loading...</div>

  const subBase = window.location.origin + "/api/sub/" + info.sub_token
  const trafficTotal = info.traffic_tx + info.traffic_rx
  const trafficPercent = info.traffic_limit > 0 ? Math.min((trafficTotal / info.traffic_limit) * 100, 100) : 0
  const daysLeft = info.expires_at && !info.expires_at.startsWith("0001") ? Math.max(0, Math.ceil((new Date(info.expires_at).getTime() - Date.now()) / 86400000)) : -1

  const pieData = [{ name: "Upload", value: info.traffic_tx }, { name: "Download", value: info.traffic_rx }]
  const nodeBarData = (info.nodes || []).map(n => ({ name: n.node, upload: n.tx, download: n.rx }))


  return (
    <div className="min-h-screen bg-surface text-ink">
      <div className="max-w-3xl mx-auto p-6">
        {/* Header */}
        <div className="flex items-center justify-between mb-6">
          <div>
            <h1 className="text-xl font-semibold">hy2board</h1>
            <p className="text-sm text-zinc-500">Welcome, {info.username}</p>
          </div>
          <div className="flex items-center gap-3">
            <a
              href="/downloads"
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium text-clay bg-clay/10 border border-clay/20 hover:bg-clay/20 transition-colors"
            >
              <Download size={12} />
              客户端下载
            </a>
            <a
              href="/status/"
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium text-green-400 bg-green-400/10 border border-green-400/20 hover:bg-green-400/20 transition-colors"
            >
              <Activity size={12} />
              Service Status
            </a>
            <span className="flex items-center gap-1.5 text-xs text-zinc-500"><span className="w-1.5 h-1.5 rounded-full bg-green-400 animate-pulse" />Live</span>
            <button onClick={logout} className="flex items-center gap-2 text-sm text-zinc-400 hover:text-ink"><LogOut size={14} /></button>
          </div>
        </div>

        {/* Status + Gauge */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
          {/* Account status */}
          <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-5">
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center gap-2">
                <Shield size={16} className={info.active ? "text-green-400" : "text-red-400"} />
                <span className={"text-sm font-medium " + (info.active ? "text-green-400" : "text-red-400")}>{info.active ? "Active" : "Inactive"}</span>
              </div>
              <div className="flex items-center gap-1.5 text-zinc-500 text-xs">
                <Clock size={12} />
                {daysLeft >= 0 ? daysLeft + " days left" : "No expiry"}
              </div>
            </div>

            {/* Traffic progress */}
            <div className="mb-3">
              <div className="flex justify-between text-sm mb-1.5">
                <span className="text-zinc-400">Traffic</span>
                <span className="font-medium">{formatBytes(trafficTotal)}{info.traffic_limit > 0 ? " / " + formatBytes(info.traffic_limit) : ""}</span>
              </div>
              {info.traffic_limit > 0 && (
                <div className="w-full h-3 bg-zinc-800 rounded-full overflow-hidden">
                  <div className={"h-full rounded-full transition-all duration-500 " + (trafficPercent > 90 ? "bg-gradient-to-r from-red-600 to-red-400" : trafficPercent > 70 ? "bg-gradient-to-r from-yellow-600 to-yellow-400" : "bg-gradient-to-r from-clay to-clay-hover")} style={{ width: trafficPercent + "%" }} />
                </div>
              )}
            </div>
            <div className="flex gap-6 text-sm">
              <span className="flex items-center gap-1.5 text-zinc-400"><ArrowUp size={13} className="text-clay" />{formatBytes(info.traffic_tx)}</span>
              <span className="flex items-center gap-1.5 text-zinc-400"><ArrowDown size={13} className="text-green-400" />{formatBytes(info.traffic_rx)}</span>
            </div>
          </div>

          {/* Upload/Download pie */}
          <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-5">
            <h3 className="text-sm font-medium text-zinc-400 mb-2">Upload vs Download</h3>
            {trafficTotal === 0 ? (
              <div className="h-32 flex items-center justify-center text-zinc-600 text-sm">No traffic yet</div>
            ) : (
              <div className="flex items-center">
                <ResponsiveContainer width="50%" height={130}>
                  <PieChart>
                    <Pie data={pieData} dataKey="value" cx="50%" cy="50%" outerRadius={50} innerRadius={30} strokeWidth={0}>
                      <Cell fill="#C96442" />
                      <Cell fill="#3F8A4D" />
                    </Pie>
                    <Tooltip contentStyle={{ background: "#FAF9F5", border: "1px solid #E5E0D6", borderRadius: 8, fontSize: 12 }} formatter={(v) => formatBytes(Number(v))} />
                  </PieChart>
                </ResponsiveContainer>
                <div className="space-y-2">
                  <div className="flex items-center gap-2 text-sm"><span className="w-3 h-3 rounded bg-clay" /><span className="text-zinc-400">Upload</span><span className="font-medium">{formatBytes(info.traffic_tx)}</span></div>
                  <div className="flex items-center gap-2 text-sm"><span className="w-3 h-3 rounded bg-green-500" /><span className="text-zinc-400">Download</span><span className="font-medium">{formatBytes(info.traffic_rx)}</span></div>
                  <div className="text-xs text-zinc-500 mt-1">Ratio: 1:{info.traffic_tx > 0 ? (info.traffic_rx / info.traffic_tx).toFixed(0) : "∞"}</div>
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Real-time speed chart */}
        <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-5 mb-4">
          <h3 className="text-sm font-medium text-zinc-400 mb-3">Real-time Speed</h3>
          {history.length < 2 ? (
            <div className="h-40 flex items-center justify-center text-zinc-600 text-sm">Collecting data...</div>
          ) : (
            <ResponsiveContainer width="100%" height={160}>
              <AreaChart data={history}>
                <defs>
                  <linearGradient id="utxG" x1="0" y1="0" x2="0" y2="1"><stop offset="5%" stopColor="#C96442" stopOpacity={0.3} /><stop offset="95%" stopColor="#C96442" stopOpacity={0} /></linearGradient>
                  <linearGradient id="urxG" x1="0" y1="0" x2="0" y2="1"><stop offset="5%" stopColor="#3F8A4D" stopOpacity={0.3} /><stop offset="95%" stopColor="#3F8A4D" stopOpacity={0} /></linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="#E5E0D6" />
                <XAxis dataKey="time" tick={{ fontSize: 9, fill: "#6B6862" }} />
                <YAxis tick={{ fontSize: 9, fill: "#6B6862" }} tickFormatter={(v) => formatBytes(v)} width={55} />
                <Tooltip contentStyle={{ background: "#FAF9F5", border: "1px solid #E5E0D6", borderRadius: 8, fontSize: 11 }} formatter={(v) => formatBytes(Number(v))} />
                <Area type="monotone" dataKey="tx" stroke="#C96442" fill="url(#utxG)" name="Upload" strokeWidth={2} />
                <Area type="monotone" dataKey="rx" stroke="#3F8A4D" fill="url(#urxG)" name="Download" strokeWidth={2} />
              </AreaChart>
            </ResponsiveContainer>
          )}
        </div>

        {/* Per-node bar chart */}
        {nodeBarData.length > 0 && (
          <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-5 mb-4">
            <h3 className="text-sm font-medium text-zinc-400 mb-3">Traffic by Node</h3>
            <ResponsiveContainer width="100%" height={180}>
              <BarChart data={nodeBarData} barGap={2}>
                <CartesianGrid strokeDasharray="3 3" stroke="#E5E0D6" />
                <XAxis dataKey="name" tick={{ fontSize: 11, fill: "#6B6862" }} />
                <YAxis tick={{ fontSize: 9, fill: "#6B6862" }} tickFormatter={(v) => formatBytes(v)} width={55} />
                <Tooltip contentStyle={{ background: "#FAF9F5", border: "1px solid #E5E0D6", borderRadius: 8, fontSize: 11 }} formatter={(v) => formatBytes(Number(v))} />
                <Bar dataKey="upload" fill="#C96442" radius={[4, 4, 0, 0]} name="Upload" />
                <Bar dataKey="download" fill="#3F8A4D" radius={[4, 4, 0, 0]} name="Download" />
              </BarChart>
            </ResponsiveContainer>
          </div>
        )}

        {/* Subscription Links */}
        <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-5 mb-4">
          <h3 className="text-sm font-medium text-zinc-400 mb-3">Subscription Links</h3>
          <div className="space-y-2">
            {[
              { label: "Universal (URI)", url: subBase },
              { label: "Clash / mihomo", url: subBase + "?format=clash" },
              { label: "Surge", url: subBase + "?format=surge" },
              { label: "Shadowrocket full config", url: subBase + "?format=shadowrocket-conf" },
              { label: "V2Ray / V2RayN / NekoBox", url: subBase + "?format=v2ray" },
            ].map(({ label, url }) => (
              <div key={label} className="flex items-center justify-between bg-surface rounded-lg p-3 border border-zinc-800">
                <div className="min-w-0 flex-1 mr-3">
                  <p className="text-sm font-medium">{label}</p>
                  <p className="text-xs text-zinc-500 truncate">{url}</p>
                </div>
                <button onClick={() => copyText(url, label)} className="flex-shrink-0 flex items-center gap-1 px-3 py-1.5 bg-zinc-800 rounded-lg text-xs text-zinc-300 hover:bg-zinc-700">
                  {copied === label ? <><Check size={12} /> Copied</> : <><Copy size={12} /> Copy</>}
                </button>
              </div>
            ))}
          </div>
          <a
            href="/downloads"
            target="_blank"
            rel="noopener noreferrer"
            className="mt-3 flex items-center justify-between bg-clay/10 border border-clay/20 hover:bg-clay/20 rounded-lg p-3 transition-colors"
          >
            <div className="flex items-center gap-2 text-sm text-clay">
              <Download size={14} />
              <span>没有客户端？点击下载推荐的代理软件</span>
            </div>
            <span className="text-xs text-clay">→</span>
          </a>
        </div>

        {/* Node list */}
        {info.nodes && info.nodes.length > 0 && (
          <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-5">
            <h3 className="text-sm font-medium text-zinc-400 mb-3">Connected Nodes</h3>
            <div className="space-y-2">
              {info.nodes.map((n, i) => (
                <div key={i} className="flex items-center justify-between bg-surface rounded-lg p-3 border border-zinc-800">
                  <div className="flex items-center gap-2">
                    <Server size={14} className="text-zinc-500" />
                    <span className="text-sm font-medium">{n.node}</span>
                  </div>
                  <div className="flex gap-4 text-xs text-zinc-400">
                    <span className="flex items-center gap-1"><ArrowUp size={10} className="text-clay" />{formatBytes(n.tx)}</span>
                    <span className="flex items-center gap-1"><ArrowDown size={10} className="text-green-400" />{formatBytes(n.rx)}</span>
                    <span className="font-medium text-ink">{formatBytes(n.tx + n.rx)}</span>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
