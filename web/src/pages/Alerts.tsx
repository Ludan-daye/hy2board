import { useEffect, useState } from "react"
import { Link } from "react-router-dom"
import api from "@/api/client"
import { AlertTriangle, Ban, BellRing, Clock3, RefreshCw, ServerCrash, TrendingUp, Users } from "lucide-react"

interface UserBrief {
  id: number
  username: string
  traffic_used: number
  traffic_limit: number
  expires_at: string
  days_until: number
  percent_used: number
}

interface NodeBrief {
  id: number
  name: string
  host: string
  port: number
  healthy: boolean
}

interface Buckets {
  expiring_soon?: UserBrief[]
  near_limit?: UserBrief[]
  over_limit?: UserBrief[]
  expired?: UserBrief[]
  disabled?: UserBrief[]
  dead_nodes?: NodeBrief[]
  updated_at?: string
}

const emptyBuckets: Required<Buckets> = {
  expiring_soon: [],
  near_limit: [],
  over_limit: [],
  expired: [],
  disabled: [],
  dead_nodes: [],
  updated_at: "",
}

function fmtBytes(bytes: number) {
  if (!bytes) return "0 B"
  const units = ["B", "KB", "MB", "GB", "TB"]
  let v = bytes
  let i = 0
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return `${v.toFixed(1)} ${units[i]}`
}

function fmtDate(value: string) {
  if (!value || value.startsWith("0001")) return "Never"
  return new Date(value).toLocaleDateString()
}

function StatCard({ icon: Icon, label, value, tone }: { icon: any; label: string; value: number; tone: string }) {
  return (
    <div className={`bg-zinc-900 border rounded-lg p-4 ${tone}`}>
      <div className="flex items-center justify-between text-xs mb-3">
        <span>{label}</span>
        <Icon size={15} />
      </div>
      <div className="text-2xl font-semibold text-ink">{value}</div>
    </div>
  )
}

function UserSection({ title, icon: Icon, rows, tone, empty }: { title: string; icon: any; rows: UserBrief[]; tone: string; empty: string }) {
  return (
    <section className="bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden">
      <div className="px-4 py-3 border-b border-zinc-800 flex items-center justify-between">
        <h3 className="text-sm font-medium text-ink flex items-center gap-2">
          <Icon size={15} className={tone} /> {title}
        </h3>
        <span className="text-xs text-zinc-500">{rows.length}</span>
      </div>
      {rows.length === 0 ? (
        <div className="px-4 py-6 text-sm text-zinc-600">{empty}</div>
      ) : (
        <div className="divide-y divide-zinc-800">
          {rows.map((u) => (
            <Link to={`/users?search=${encodeURIComponent(u.username)}`} key={`${title}-${u.id}`} className="block px-4 py-3 hover:bg-surface/[0.03]">
              <div className="flex items-center justify-between gap-4">
                <div>
                  <div className="text-sm font-medium text-zinc-100">{u.username}</div>
                  <div className="text-xs text-zinc-500 mt-1">
                    到期 {fmtDate(u.expires_at)} · {u.days_until >= 0 ? `剩 ${u.days_until} 天` : "已过期"}
                  </div>
                </div>
                <div className="text-right shrink-0">
                  <div className="text-sm text-zinc-200">{u.percent_used}%</div>
                  <div className="text-xs text-zinc-500 mt-1">{fmtBytes(u.traffic_used)} / {u.traffic_limit ? fmtBytes(u.traffic_limit) : "Unlimited"}</div>
                </div>
              </div>
            </Link>
          ))}
        </div>
      )}
    </section>
  )
}

function NodeSection({ rows }: { rows: NodeBrief[] }) {
  return (
    <section className="bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden">
      <div className="px-4 py-3 border-b border-zinc-800 flex items-center justify-between">
        <h3 className="text-sm font-medium text-ink flex items-center gap-2">
          <ServerCrash size={15} className="text-red-400" /> Down Nodes
        </h3>
        <span className="text-xs text-zinc-500">{rows.length}</span>
      </div>
      {rows.length === 0 ? (
        <div className="px-4 py-6 text-sm text-zinc-600">节点都在线，先喝口水。</div>
      ) : (
        <div className="divide-y divide-zinc-800">
          {rows.map((n) => (
            <Link to="/nodes" key={n.id} className="block px-4 py-3 hover:bg-surface/[0.03]">
              <div className="flex items-center justify-between gap-4">
                <div>
                  <div className="text-sm font-medium text-zinc-100">{n.name}</div>
                  <div className="text-xs text-zinc-500 mt-1">{n.host}:{n.port}</div>
                </div>
                <span className="text-xs px-2 py-1 rounded bg-red-500/10 text-red-300 border border-red-500/20">DOWN</span>
              </div>
            </Link>
          ))}
        </div>
      )}
    </section>
  )
}

export default function Alerts() {
  const [buckets, setBuckets] = useState<Required<Buckets>>(emptyBuckets)
  const [loading, setLoading] = useState(false)

  const load = async () => {
    setLoading(true)
    try {
      const r = await api.get("/admin/buckets")
      setBuckets({
        expiring_soon: r.data.expiring_soon || [],
        near_limit: r.data.near_limit || [],
        over_limit: r.data.over_limit || [],
        expired: r.data.expired || [],
        disabled: r.data.disabled || [],
        dead_nodes: r.data.dead_nodes || [],
        updated_at: r.data.updated_at || "",
      })
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
    const timer = setInterval(load, 30000)
    return () => clearInterval(timer)
  }, [])

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-xl font-semibold flex items-center gap-2">
          <BellRing size={18} /> Alerts
          <span className="text-xs text-zinc-500 font-normal ml-3">
            {buckets.updated_at ? `updated ${new Date(buckets.updated_at).toLocaleTimeString()}` : "loading"}
          </span>
        </h2>
        <button
          onClick={load}
          disabled={loading}
          className="flex items-center gap-2 px-3 py-1.5 bg-clay text-white rounded-lg text-sm font-medium hover:bg-clay-hover disabled:opacity-60"
        >
          <RefreshCw size={14} className={loading ? "animate-spin" : ""} /> Refresh
        </button>
      </div>

      <div className="grid grid-cols-2 xl:grid-cols-6 gap-3 mb-4">
        <StatCard icon={Clock3} label="Expiring" value={buckets.expiring_soon.length} tone="border-amber-500/20 text-amber-300" />
        <StatCard icon={AlertTriangle} label="Expired" value={buckets.expired.length} tone="border-red-500/20 text-red-300" />
        <StatCard icon={TrendingUp} label="Near Limit" value={buckets.near_limit.length} tone="border-clay/20 text-clay" />
        <StatCard icon={Ban} label="Over Limit" value={buckets.over_limit.length} tone="border-red-500/20 text-red-300" />
        <StatCard icon={Users} label="Disabled" value={buckets.disabled.length} tone="border-zinc-700 text-zinc-400" />
        <StatCard icon={ServerCrash} label="Down Nodes" value={buckets.dead_nodes.length} tone="border-red-500/20 text-red-300" />
      </div>

      <div className="grid grid-cols-1 xl:grid-cols-2 gap-4">
        <UserSection title="Expiring Soon" icon={Clock3} rows={buckets.expiring_soon} tone="text-amber-300" empty="7 天内没有用户到期。" />
        <UserSection title="Expired" icon={AlertTriangle} rows={buckets.expired} tone="text-red-400" empty="没有已过期用户。" />
        <UserSection title="Near Traffic Limit" icon={TrendingUp} rows={buckets.near_limit} tone="text-clay" empty="没有 80% 以上流量用户。" />
        <UserSection title="Over Traffic Limit" icon={Ban} rows={buckets.over_limit} tone="text-red-400" empty="没有超流量用户。" />
        <UserSection title="Disabled Users" icon={Users} rows={buckets.disabled} tone="text-zinc-400" empty="没有停用用户。" />
        <NodeSection rows={buckets.dead_nodes} />
      </div>
    </div>
  )
}
