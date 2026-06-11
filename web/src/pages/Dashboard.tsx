import { useEffect, useState } from "react"
import { useNavigate } from "react-router-dom"
import api from "@/api/client"
import {
  Users, ArrowUp, ArrowDown, Wifi, Server, Activity, Shield, Globe,
  AlertTriangle, Clock, TrendingUp, Zap, Award, Ban,
} from "lucide-react"
import {
  BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer,
  PieChart, Pie, Cell, AreaChart, Area, CartesianGrid, Line, LineChart,
} from "recharts"

interface TrafficData { tx: number; rx: number }
interface NodeTraffic {
  id: number; name: string; host: string; port: number
  healthy: boolean; online: number
  traffic: Record<string, TrafficData> | null
}
interface NodeSpeedAtPoint { speed_tx: number; speed_rx: number }
interface ServerHistoryPoint {
  time: string; total_tx: number; total_rx: number
  speed_tx: number; speed_rx: number
  per_node?: Record<string, NodeSpeedAtPoint>
}
interface Stats {
  total_users: number; active_users: number
  total_nodes: number; healthy_nodes: number
  total_tx: number; total_rx: number
  nodes: NodeTraffic[]
  history?: ServerHistoryPoint[]
}
interface UserRow {
  ID: number; username: string
  traffic_limit: number; traffic_used: number
  speed_tx?: number; speed_rx?: number
  live_tx?: number; live_rx?: number
  expires_at: string; enabled: boolean
  chain_proxy: boolean
}

interface HistoryPoint { time: string; tx: number; rx: number }

const COLORS = ["#1a2b6b", "#243b8f", "#2d4ea6", "#d4a017", "#c4880b", "#8b6914"]
const STATUS_COLORS = { active: "#22c55e", expired: "#f59e0b", over: "#ef4444", disabled: "#71717a" }

function fmt(bytes: number): string {
  if (bytes === 0) return "0 B"
  const k = 1024
  const s = ["B", "KB", "MB", "GB", "TB"]
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return (bytes / Math.pow(k, i)).toFixed(1) + " " + s[i]
}

function fmtSpeed(bps: number): string {
  if (!bps || bps === 0) return "0 B/s"
  return fmt(bps) + "/s"
}

function daysUntil(dateStr: string): number {
  if (!dateStr || dateStr.startsWith("0001")) return -1
  return Math.ceil((new Date(dateStr).getTime() - Date.now()) / 86400000)
}

const tooltipStyle = {
  contentStyle: { background: "#111318", border: "1px solid rgba(255,255,255,0.08)", borderRadius: 8, fontSize: 12, color: "#fff" },
  itemStyle: { color: "#999" },
}

function StatCard({ icon: Icon, label, value, sub, accent }: {
  icon: React.ElementType; label: string; value: string | number; sub?: string; accent?: string
}) {
  const color = accent || "#1a2b6b"
  return (
    <div className="relative overflow-hidden rounded-xl p-5" style={{ background: "#111318", border: "1px solid rgba(255,255,255,0.06)" }}>
      <div className="absolute -top-8 -right-8 w-24 h-24 rounded-full opacity-20 blur-2xl" style={{ background: color }} />
      <div className="relative">
        <div className="flex items-center gap-2.5 mb-3">
          <div className="w-8 h-8 rounded-lg flex items-center justify-center" style={{ background: color + "1a" }}>
            <Icon size={14} style={{ color }} />
          </div>
          <span className="text-[11px] font-semibold tracking-[0.1em] text-zinc-500 uppercase">{label}</span>
        </div>
        <p className="text-2xl font-bold text-white">{value}</p>
        {sub && <p className="text-[11px] text-zinc-500 mt-1">{sub}</p>}
      </div>
    </div>
  )
}

function Card({ icon: Icon, title, children, right }: {
  icon: React.ElementType; title: string; children: React.ReactNode; right?: React.ReactNode
}) {
  return (
    <div className="rounded-xl p-5" style={{ background: "#111318", border: "1px solid rgba(255,255,255,0.06)" }}>
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <Icon size={14} className="text-zinc-500" />
          <span className="text-[12px] font-semibold text-zinc-400">{title}</span>
        </div>
        {right}
      </div>
      {children}
    </div>
  )
}

export default function Dashboard() {
  const navigate = useNavigate()
  const [stats, setStats] = useState<Stats | null>(null)
  const [users, setUsers] = useState<UserRow[]>([])
  const [history, setHistory] = useState<HistoryPoint[]>([])
  const [income, setIncome] = useState<{month: string; total_cents: number; count: number; new_count: number; renew_count: number}[]>([])
  const [staticIPs, setStaticIPs] = useState<{
    plan_id: number
    proxy_host: string
    proxy_note: string
    healthy: boolean
    last_rtt_ms: number
    users: unknown[]
    traffic_24h_tx: number
    traffic_24h_rx: number
  }[]>([])

  const load = () => {
    api.get("/admin/static-ips").then(r => setStaticIPs(r.data || []))
    api.get("/admin/payments/summary?n=2").then(r => setIncome(r.data || []))
    Promise.all([api.get("/admin/stats"), api.get("/admin/users")]).then(([s, u]) => {
      const sd = s.data as Stats
      setStats(sd)
      setUsers(u.data)
      if (Array.isArray(sd.history)) {
        const mapped = sd.history.slice(-30).map(p => ({
          time: new Date(p.time).toLocaleTimeString("en-US", { hour12: false, hour: "2-digit", minute: "2-digit", second: "2-digit" }),
          tx: p.speed_tx,
          rx: p.speed_rx,
        }))
        setHistory(mapped)
      }
    })
  }

  useEffect(() => { load(); const t = setInterval(load, 5000); return () => clearInterval(t) }, [])

  if (!stats) return <p className="text-zinc-500 p-8">Loading...</p>

  /* ── Derived data ── */
  const totalOnline = stats.nodes.reduce((s, n) => s + n.online, 0)

  // Alerts
  const expiringSoon = users
    .filter(u => { const d = daysUntil(u.expires_at); return d >= 0 && d <= 7 && u.enabled })
    .sort((a, b) => daysUntil(a.expires_at) - daysUntil(b.expires_at))
  const nearLimit = users
    .filter(u => u.traffic_limit > 0 && u.traffic_used / u.traffic_limit >= 0.8 && u.enabled)
    .sort((a, b) => (b.traffic_used / b.traffic_limit) - (a.traffic_used / a.traffic_limit))
  const deadNodes = stats.nodes.filter(n => !n.healthy)

  // Top 5 live speed
  const liveSpeed = users
    .map(u => ({
      name: u.username,
      up: u.speed_tx || 0,
      down: u.speed_rx || 0,
      total: (u.speed_tx || 0) + (u.speed_rx || 0),
    }))
    .filter(u => u.total > 0)
    .sort((a, b) => b.total - a.total)
    .slice(0, 5)

  // Status breakdown
  const statusCounts = { active: 0, expired: 0, over: 0, disabled: 0 }
  users.forEach(u => {
    if (!u.enabled) statusCounts.disabled++
    else if (daysUntil(u.expires_at) >= 0 && daysUntil(u.expires_at) < 0) statusCounts.expired++
    else if (u.expires_at && !u.expires_at.startsWith("0001") && new Date(u.expires_at) < new Date()) statusCounts.expired++
    else if (u.traffic_limit > 0 && u.traffic_used >= u.traffic_limit) statusCounts.over++
    else statusCounts.active++
  })
  const statusPie = [
    { name: "Active", value: statusCounts.active, color: STATUS_COLORS.active },
    { name: "Expired", value: statusCounts.expired, color: STATUS_COLORS.expired },
    { name: "Over Limit", value: statusCounts.over, color: STATUS_COLORS.over },
    { name: "Disabled", value: statusCounts.disabled, color: STATUS_COLORS.disabled },
  ].filter(x => x.value > 0)

  // Top users by cumulative traffic observed at HY2 nodes (live_tx + live_rx).
  // The db-stored `traffic_used` counter is reset on admin action and often 0.
  const liveTotal = (u: UserRow) => (u.live_tx || 0) + (u.live_rx || 0)
  const topTraffic = [...users]
    .sort((a, b) => liveTotal(b) - liveTotal(a))
    .slice(0, 6)
    .filter(u => liveTotal(u) > 0)

  // Per-node historical mini-charts
  const nodeHistory: Record<string, { time: string; val: number }[]> = {}
  if (Array.isArray(stats.history)) {
    stats.history.slice(-20).forEach(p => {
      const t = new Date(p.time).toLocaleTimeString("en-US", { hour12: false, hour: "2-digit", minute: "2-digit", second: "2-digit" })
      if (p.per_node) {
        Object.entries(p.per_node).forEach(([name, s]) => {
          if (!nodeHistory[name]) nodeHistory[name] = []
          nodeHistory[name].push({ time: t, val: (s.speed_tx || 0) + (s.speed_rx || 0) })
        })
      }
    })
  }

  // Node bar chart data
  const nodeBarData = stats.nodes.map((n) => {
    let tx = 0, rx = 0
    if (n.traffic) Object.values(n.traffic).forEach((t) => { tx += t.tx; rx += t.rx })
    return { name: n.name, upload: tx, download: rx, total: tx + rx }
  })
  const pieData = nodeBarData.filter((d) => d.total > 0).map((d) => ({ name: d.name, value: d.total }))

  const totalAlerts = expiringSoon.length + nearLimit.length + deadNodes.length

  return (
    <div>
      {/* Stats row */}
      <div className="grid grid-cols-2 lg:grid-cols-6 gap-3 mb-4">
        <StatCard icon={Users} label="Users" value={`${stats.active_users}/${stats.total_users}`} sub="active / total" accent="#1a2b6b" />
        <StatCard icon={Server} label="Nodes" value={`${stats.healthy_nodes}/${stats.total_nodes}`} sub="healthy / total" accent="#243b8f" />
        <StatCard icon={Wifi} label="Online" value={totalOnline} accent="#2d4ea6" />
        <StatCard icon={ArrowUp} label="Upload" value={fmt(stats.total_tx)} accent="#1a2b6b" />
        <StatCard icon={ArrowDown} label="Download" value={fmt(stats.total_rx)} accent="#243b8f" />
        {income.length >= 1 && (() => {
          const cur = income[income.length - 1]
          const prev = income.length >= 2 ? income[income.length - 2] : null
          const delta = prev ? cur.total_cents - prev.total_cents : 0
          const pct = prev && prev.total_cents > 0 ? (delta / prev.total_cents) * 100 : null
          const yuan = (n: number) => `¥${(n / 100).toFixed(2)}`
          return (
            <div onClick={() => navigate("/finance")} className="bg-zinc-900 border border-zinc-800 rounded-xl p-4 cursor-pointer hover:border-zinc-700">
              <div className="text-xs text-zinc-500 mb-1">本月收入</div>
              <div className="text-2xl font-semibold">{yuan(cur.total_cents)}</div>
              <div className="text-xs text-zinc-500 mt-1">
                {cur.count} 笔
                {prev && (
                  <span className={delta >= 0 ? "text-green-400 ml-2" : "text-red-400 ml-2"}>
                    · {delta >= 0 ? "+" : ""}{yuan(delta)}
                    {pct !== null && ` (${delta >= 0 ? "↑" : "↓"}${Math.abs(pct).toFixed(0)}%)`}
                  </span>
                )}
              </div>
            </div>
          )
        })()}
      </div>

      {/* Alert panel */}
      {totalAlerts > 0 && (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-3 mb-4">
          {expiringSoon.length > 0 && (
            <Card icon={Clock} title="Expiring Soon" right={<span className="text-[10px] px-1.5 py-0.5 rounded bg-amber-400/10 text-amber-400 font-semibold">{expiringSoon.length}</span>}>
              <div className="space-y-1.5">
                {expiringSoon.slice(0, 4).map(u => {
                  const d = daysUntil(u.expires_at)
                  const urgent = d <= 2
                  return (
                    <div key={u.ID} className="flex items-center justify-between py-1.5 px-2 rounded" style={{ background: urgent ? "rgba(239,68,68,0.06)" : "rgba(245,158,11,0.04)" }}>
                      <span className="text-[13px] text-white truncate">{u.username}</span>
                      <span className={"text-[11px] font-mono " + (urgent ? "text-red-400" : "text-amber-400")}>
                        {d === 0 ? "today" : d === 1 ? "tomorrow" : `${d}d`}
                      </span>
                    </div>
                  )
                })}
              </div>
            </Card>
          )}
          {nearLimit.length > 0 && (
            <Card icon={AlertTriangle} title="Near Traffic Limit" right={<span className="text-[10px] px-1.5 py-0.5 rounded bg-red-400/10 text-red-400 font-semibold">{nearLimit.length}</span>}>
              <div className="space-y-1.5">
                {nearLimit.slice(0, 4).map(u => {
                  const pct = Math.min(100, (u.traffic_used / u.traffic_limit) * 100)
                  return (
                    <div key={u.ID} className="py-1.5 px-2 rounded" style={{ background: "rgba(239,68,68,0.04)" }}>
                      <div className="flex items-center justify-between mb-1">
                        <span className="text-[13px] text-white truncate">{u.username}</span>
                        <span className="text-[11px] font-mono text-red-400">{pct.toFixed(0)}%</span>
                      </div>
                      <div className="w-full h-1 rounded-full overflow-hidden" style={{ background: "rgba(255,255,255,0.06)" }}>
                        <div className="h-full rounded-full" style={{ width: pct + "%", background: pct > 95 ? "#ef4444" : "#f59e0b" }} />
                      </div>
                    </div>
                  )
                })}
              </div>
            </Card>
          )}
          {deadNodes.length > 0 && (
            <Card icon={Ban} title="Offline Nodes" right={<span className="text-[10px] px-1.5 py-0.5 rounded bg-red-400/10 text-red-400 font-semibold">{deadNodes.length}</span>}>
              <div className="space-y-1.5">
                {deadNodes.slice(0, 4).map(n => (
                  <div key={n.id} className="flex items-center justify-between py-1.5 px-2 rounded" style={{ background: "rgba(239,68,68,0.06)" }}>
                    <div>
                      <p className="text-[13px] text-white">{n.name}</p>
                      <p className="text-[10px] text-zinc-600 font-mono">{n.host}:{n.port}</p>
                    </div>
                    <span className="relative flex h-2 w-2"><span className="relative inline-flex rounded-full h-2 w-2 bg-red-500" /></span>
                  </div>
                ))}
              </div>
            </Card>
          )}
        </div>
      )}

      {/* Row: Live speed leaderboard + Status donut + Top traffic */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-3 mb-4">
        {/* Live speed ranking */}
        <div className="lg:col-span-5">
          <Card icon={Zap} title="Live Speed Ranking" right={<span className="text-[10px] text-zinc-600 font-mono">top 5</span>}>
            {liveSpeed.length === 0 ? (
              <div className="h-[180px] flex items-center justify-center text-zinc-600 text-sm">No active traffic</div>
            ) : (
              <div className="space-y-2.5">
                {liveSpeed.map((u, i) => {
                  const maxTotal = liveSpeed[0].total || 1
                  const upPct = (u.up / maxTotal) * 100
                  const downPct = (u.down / maxTotal) * 100
                  return (
                    <div key={u.name}>
                      <div className="flex items-center justify-between text-[11px] mb-1">
                        <span className="flex items-center gap-2">
                          <span className={"inline-block w-5 text-center rounded font-bold text-[10px] " + (i === 0 ? "text-amber-400" : "text-zinc-500")}>#{i + 1}</span>
                          <span className="text-white font-medium">{u.name}</span>
                        </span>
                        <span className="font-mono text-zinc-300">{fmtSpeed(u.total)}</span>
                      </div>
                      <div className="flex h-1.5 rounded-full overflow-hidden bg-zinc-900">
                        <div className="h-full" style={{ width: upPct + "%", background: "#1a2b6b" }} />
                        <div className="h-full" style={{ width: downPct + "%", background: "#d4a017" }} />
                      </div>
                      <div className="flex gap-3 text-[10px] text-zinc-600 mt-0.5 font-mono">
                        <span className="flex items-center gap-1"><span className="w-1.5 h-1.5 rounded-sm" style={{ background: "#1a2b6b" }} />↑{fmtSpeed(u.up)}</span>
                        <span className="flex items-center gap-1"><span className="w-1.5 h-1.5 rounded-sm" style={{ background: "#d4a017" }} />↓{fmtSpeed(u.down)}</span>
                      </div>
                    </div>
                  )
                })}
              </div>
            )}
          </Card>
        </div>

        {/* Status donut */}
        <div className="lg:col-span-3">
          <Card icon={Shield} title="Account Health">
            {statusPie.length === 0 ? (
              <div className="h-[180px] flex items-center justify-center text-zinc-600 text-sm">No users</div>
            ) : (
              <>
                <ResponsiveContainer width="100%" height={140}>
                  <PieChart>
                    <Pie data={statusPie} dataKey="value" cx="50%" cy="50%" outerRadius={58} innerRadius={36} strokeWidth={0}>
                      {statusPie.map((d, i) => <Cell key={i} fill={d.color} />)}
                    </Pie>
                    <Tooltip {...tooltipStyle} />
                  </PieChart>
                </ResponsiveContainer>
                <div className="grid grid-cols-2 gap-1 mt-2">
                  {statusPie.map(d => (
                    <span key={d.name} className="flex items-center gap-1.5 text-[10px] text-zinc-400">
                      <span className="w-2 h-2 rounded-sm" style={{ background: d.color }} />
                      {d.name} <span className="text-white font-semibold">{d.value}</span>
                    </span>
                  ))}
                </div>
              </>
            )}
          </Card>
        </div>

        {/* Top traffic */}
        <div className="lg:col-span-4">
          <Card icon={Award} title="Top Traffic Users" right={<span className="text-[10px] text-zinc-600 font-mono">all-time</span>}>
            {topTraffic.length === 0 ? (
              <div className="h-[180px] flex items-center justify-center text-zinc-600 text-sm">No data</div>
            ) : (
              <div className="space-y-1.5">
                {topTraffic.map((u, i) => {
                  const uTotal = liveTotal(u)
                  const max = liveTotal(topTraffic[0]) || 1
                  const pct = (uTotal / max) * 100
                  return (
                    <div key={u.ID} className="relative">
                      <div className="absolute inset-0 rounded" style={{ width: pct + "%", background: "rgba(43, 71, 107, 0.25)" }} />
                      <div className="relative flex items-center justify-between px-2 py-1.5 text-[12px]">
                        <span className="flex items-center gap-2 truncate">
                          <span className="text-zinc-500 font-mono w-4">{i + 1}.</span>
                          <span className="text-white truncate">{u.username}</span>
                          {u.chain_proxy && <span className="text-[9px] px-1 py-0.5 rounded bg-purple-500/10 text-purple-400">AI</span>}
                        </span>
                        <span className="font-mono text-zinc-300 ml-2">{fmt(uTotal)}</span>
                      </div>
                    </div>
                  )
                })}
              </div>
            )}
          </Card>
        </div>
      </div>

      {/* Real-time traffic */}
      <div className="mb-4">
        <Card icon={Activity} title="Real-time Traffic" right={<span className="text-[10px] text-zinc-600 font-mono">3s interval</span>}>
          {history.length < 2 ? (
            <div className="h-[220px] flex items-center justify-center text-zinc-700 text-sm">Collecting data...</div>
          ) : (
            <ResponsiveContainer width="100%" height={220}>
              <AreaChart data={history}>
                <defs>
                  <linearGradient id="txG" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#1a2b6b" stopOpacity={0.5} />
                    <stop offset="95%" stopColor="#1a2b6b" stopOpacity={0} />
                  </linearGradient>
                  <linearGradient id="rxG" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#d4a017" stopOpacity={0.4} />
                    <stop offset="95%" stopColor="#d4a017" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.04)" />
                <XAxis dataKey="time" tick={{ fontSize: 10, fill: "#555" }} axisLine={false} tickLine={false} />
                <YAxis tick={{ fontSize: 10, fill: "#555" }} tickFormatter={(v) => fmt(v)} width={55} axisLine={false} tickLine={false} />
                <Tooltip {...tooltipStyle} formatter={(v: unknown) => fmtSpeed(Number(v))} />
                <Area type="monotone" dataKey="tx" stroke="#1a2b6b" fill="url(#txG)" name="Upload" strokeWidth={2} />
                <Area type="monotone" dataKey="rx" stroke="#d4a017" fill="url(#rxG)" name="Download" strokeWidth={2} />
              </AreaChart>
            </ResponsiveContainer>
          )}
          <div className="flex gap-5 mt-3">
            <span className="flex items-center gap-2 text-[11px] text-zinc-500"><span className="w-3 h-[2px] rounded" style={{ background: "#1a2b6b" }} />Upload</span>
            <span className="flex items-center gap-2 text-[11px] text-zinc-500"><span className="w-3 h-[2px] rounded" style={{ background: "#d4a017" }} />Download</span>
          </div>
        </Card>
      </div>

      {/* Per-node cards with mini sparklines */}
      <div className="mb-4">
        <Card icon={Server} title="Per-Node Live Activity">
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
            {stats.nodes.map(n => {
              const nh = nodeHistory[n.name] || []
              const latestSpeed = nh.length > 0 ? nh[nh.length - 1].val : 0
              let totalTraffic = 0
              if (n.traffic) Object.values(n.traffic).forEach(t => { totalTraffic += t.tx + t.rx })
              return (
                <div key={n.id} className="rounded-lg p-3" style={{ background: "rgba(255,255,255,0.02)", border: "1px solid rgba(255,255,255,0.04)" }}>
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center gap-2 min-w-0">
                      <span className={"w-2 h-2 rounded-full flex-shrink-0 " + (n.healthy ? "bg-green-500" : "bg-red-500")} style={n.healthy ? { boxShadow: "0 0 6px rgba(34,197,94,0.5)" } : {}} />
                      <span className="text-[13px] font-medium text-white truncate">{n.name}</span>
                    </div>
                    <span className="text-[10px] text-zinc-500">{n.online} online</span>
                  </div>
                  <div className="h-10 -mx-1 mb-1">
                    {nh.length > 1 ? (
                      <ResponsiveContainer width="100%" height="100%">
                        <LineChart data={nh}>
                          <Line type="monotone" dataKey="val" stroke={n.healthy ? "#22c55e" : "#ef4444"} strokeWidth={1.5} dot={false} />
                        </LineChart>
                      </ResponsiveContainer>
                    ) : (
                      <div className="h-full flex items-center justify-center text-[10px] text-zinc-700">—</div>
                    )}
                  </div>
                  <div className="flex items-end justify-between">
                    <div>
                      <p className="text-[10px] text-zinc-500 uppercase tracking-wider">Now</p>
                      <p className="text-[14px] font-bold text-white font-mono">{fmtSpeed(latestSpeed)}</p>
                    </div>
                    <div className="text-right">
                      <p className="text-[10px] text-zinc-500 uppercase tracking-wider">Total</p>
                      <p className="text-[12px] text-zinc-300 font-mono">{fmt(totalTraffic)}</p>
                    </div>
                  </div>
                </div>
              )
            })}
          </div>
        </Card>
      </div>

      {/* Static IP widget */}
      {staticIPs.length > 0 && (
        <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-4 mb-4">
          <div className="flex items-center justify-between mb-3">
            <h3 className="text-sm font-semibold text-white flex items-center gap-2">
              🌐 Static IPs
            </h3>
            <a href="/static-ips" className="text-xs text-zinc-400 hover:text-white">
              管理 →
            </a>
          </div>
          <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-2">
            {staticIPs.map(ip => (
              <div
                key={ip.plan_id}
                className="bg-zinc-950 border border-zinc-800 rounded p-3 text-xs"
              >
                <div className="flex items-center gap-1 mb-1">
                  <span>{ip.healthy ? "🟢" : "🔴"}</span>
                  <span className="text-zinc-300 truncate" title={ip.proxy_note || ip.proxy_host}>
                    {(ip.proxy_note || ip.proxy_host).slice(0, 8)}
                  </span>
                </div>
                <div className="font-mono text-[10px] text-zinc-500">{ip.proxy_host}</div>
                <div className="text-zinc-400 mt-1">{(ip.users || []).length} user{(ip.users || []).length === 1 ? "" : "s"}</div>
                <div className="text-zinc-500">{ip.healthy ? `${ip.last_rtt_ms}ms` : "timeout"}</div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Node comparison + Traffic distribution */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-3 mb-4">
        <div className="lg:col-span-2">
          <Card icon={TrendingUp} title="Node Traffic Comparison">
            <ResponsiveContainer width="100%" height={200}>
              <BarChart data={nodeBarData} barGap={4}>
                <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.04)" />
                <XAxis dataKey="name" tick={{ fontSize: 11, fill: "#666" }} axisLine={false} tickLine={false} />
                <YAxis tick={{ fontSize: 10, fill: "#555" }} tickFormatter={(v) => fmt(v)} width={60} axisLine={false} tickLine={false} />
                <Tooltip {...tooltipStyle} formatter={(v: unknown) => fmt(Number(v))} />
                <Bar dataKey="upload" fill="#1a2b6b" radius={[4, 4, 0, 0]} name="Upload" />
                <Bar dataKey="download" fill="#d4a017" radius={[4, 4, 0, 0]} name="Download" />
              </BarChart>
            </ResponsiveContainer>
          </Card>
        </div>
        <Card icon={Globe} title="Traffic Distribution">
          {pieData.length === 0 ? (
            <div className="h-[200px] flex items-center justify-center text-zinc-700 text-sm">No data</div>
          ) : (
            <>
              <ResponsiveContainer width="100%" height={160}>
                <PieChart>
                  <Pie data={pieData} dataKey="value" cx="50%" cy="50%" outerRadius={60} innerRadius={35} strokeWidth={0}>
                    {pieData.map((_, i) => <Cell key={i} fill={COLORS[i % COLORS.length]} />)}
                  </Pie>
                  <Tooltip {...tooltipStyle} formatter={(v: unknown) => fmt(Number(v))} />
                </PieChart>
              </ResponsiveContainer>
              <div className="flex flex-wrap gap-x-3 gap-y-1 mt-2 justify-center">
                {pieData.map((d, i) => (
                  <span key={d.name} className="flex items-center gap-1.5 text-[11px] text-zinc-500">
                    <span className="w-2 h-2 rounded-sm" style={{ background: COLORS[i % COLORS.length] }} />
                    {d.name}
                  </span>
                ))}
              </div>
            </>
          )}
        </Card>
      </div>
    </div>
  )
}
