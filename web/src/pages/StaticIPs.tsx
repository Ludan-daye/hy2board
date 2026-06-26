import { useEffect, useState, Fragment } from "react"
import api from "@/api/client"
import { Globe, Plus, Trash2 } from "lucide-react"
import PlanEditModal, { type Plan } from "@/components/PlanEditModal"

interface StaticIPRow {
  plan_id: number
  plan_name: string
  proxy_type: string
  proxy_host: string
  proxy_port: number
  proxy_username: string
  proxy_note: string
  healthy: boolean
  last_probed_at?: string
  last_rtt_ms: number
  last_exit_ip: string
  users: { id: number; username: string; expires_at: string; traffic_24h: number }[]
  traffic_24h_tx: number
  traffic_24h_rx: number
}

function fmtBytes(n: number): string {
  if (n === 0) return "0"
  const u = ["B", "KB", "MB", "GB", "TB"]
  let i = 0
  let v = n
  while (v >= 1024 && i < u.length - 1) { v /= 1024; i++ }
  return `${v.toFixed(1)} ${u[i]}`
}

export default function StaticIPs() {
  const [rows, setRows] = useState<StaticIPRow[]>([])
  const [expanded, setExpanded] = useState<number | null>(null)
  const [editing, setEditing] = useState<Plan | null>(null)
  const [showNew, setShowNew] = useState(false)

  const load = () => api.get("/admin/static-ips").then(r => {
    // Backend marshals empty user slice as null; normalize to []
    const data: StaticIPRow[] = (r.data || []).map((row: StaticIPRow) => ({
      ...row,
      users: row.users || [],
    }))
    setRows(data)
  })
  useEffect(() => {
    load()
    const i = setInterval(load, 5000)
    return () => clearInterval(i)
  }, [])

  const del = async (r: StaticIPRow) => {
    if (!confirm(`删除独享 IP "${r.plan_name}" (${r.proxy_host}:${r.proxy_port})？`)) return
    try {
      await api.delete(`/admin/plans/${r.plan_id}`)
      if (expanded === r.plan_id) setExpanded(null)
      load()
    } catch (e: unknown) {
      const err = e as { response?: { status?: number; data?: { users_count?: number; error?: string } } }
      if (err.response?.status === 409) {
        alert(`这个独享 IP 还绑定着 ${err.response.data?.users_count || r.users.length} 个用户。先把用户换到别的 Plan，再删除。`)
      } else {
        alert(err.response?.data?.error || String(e))
      }
    }
  }

  const summary = {
    total: rows.length,
    healthy: rows.filter(r => r.healthy).length,
    unhealthy: rows.filter(r => !r.healthy).length,
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-xl font-semibold flex items-center gap-2">
          <Globe size={18} /> Static IPs
          <span className="text-xs text-zinc-500 font-normal ml-3">
            {summary.total} 个 · {summary.healthy} 健康 · {summary.unhealthy} 离线
          </span>
        </h2>
        <button
          onClick={() => setShowNew(true)}
          className="flex items-center gap-2 px-3 py-1.5 bg-clay text-white rounded-lg text-sm font-medium hover:bg-clay-hover"
        >
          <Plus size={14} /> 新增独享 IP Plan
        </button>
      </div>

      <div className="bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden">
        <table className="w-full text-sm">
          <thead className="text-xs text-zinc-500 uppercase">
            <tr>
              <th className="text-left p-3">状态</th>
              <th className="text-left p-3">IP</th>
              <th className="text-left p-3">类型</th>
              <th className="text-left p-3">备注</th>
              <th className="text-right p-3">用户</th>
              <th className="text-right p-3">24h ↑/↓</th>
              <th className="p-3"></th>
            </tr>
          </thead>
          <tbody>
            {rows.length === 0 && (
              <tr><td colSpan={7} className="p-6 text-center text-zinc-600">No IP-bearing plans yet.</td></tr>
            )}
            {rows.map(r => (
              <Fragment key={r.plan_id}>
                <tr
                  className="border-t border-zinc-800 hover:bg-zinc-800/30 cursor-pointer"
                  onClick={() => setExpanded(expanded === r.plan_id ? null : r.plan_id)}
                >
                  <td className="p-3">{r.healthy ? "🟢" : "🔴"}</td>
                  <td className="p-3 font-mono text-xs">{r.proxy_host}:{r.proxy_port}</td>
                  <td className="p-3 text-xs text-zinc-400">{r.proxy_type}</td>
                  <td className="p-3 text-xs text-zinc-300 max-w-[160px] truncate" title={r.proxy_note}>
                    {r.proxy_note || "—"}
                  </td>
                  <td className="p-3 text-right">{r.users.length}</td>
                  <td className="p-3 text-right text-xs text-zinc-400 font-mono">
                    {fmtBytes(r.traffic_24h_tx)} / {fmtBytes(r.traffic_24h_rx)}
                  </td>
                  <td className="p-3 text-right">
                    <div className="flex items-center justify-end gap-1">
                      <button
                        onClick={(e) => {
                          e.stopPropagation()
                          setEditing({
                            ID: r.plan_id,
                            name: r.plan_name,
                            traffic_limit: 0,
                            duration_days: 0,
                            node_ids: "",
                            rule_ai: false,
                            rule_streaming: false,
                            rule_china: false,
                            rule_ad_block: false,
                            auto_reset: false,
                            sort_order: 0,
                            proxy_type: r.proxy_type,
                            proxy_host: r.proxy_host,
                            proxy_port: r.proxy_port,
                            proxy_username: r.proxy_username,
                            proxy_note: r.proxy_note,
                          })
                        }}
                        className="px-2 py-1 text-xs bg-zinc-800 hover:bg-zinc-700 text-ink rounded"
                      >
                        Edit
                      </button>
                      <button
                        onClick={(e) => {
                          e.stopPropagation()
                          del(r)
                        }}
                        className="p-1.5 rounded hover:bg-zinc-700 text-zinc-500 hover:text-red-400"
                        title="删除独享 IP"
                        aria-label="删除独享 IP"
                      >
                        <Trash2 size={14} />
                      </button>
                    </div>
                  </td>
                </tr>
                {expanded === r.plan_id && (
                  <tr className="border-t border-zinc-800 bg-zinc-900/40">
                    <td colSpan={7} className="p-4 text-xs">
                      <div className="grid grid-cols-2 gap-x-6 gap-y-2 mb-3">
                        <div><span className="text-zinc-500">Plan:</span> {r.plan_name}</div>
                        <div><span className="text-zinc-500">Username:</span> <span className="font-mono">{r.proxy_username}</span></div>
                        <div><span className="text-zinc-500">Last probe:</span> {r.last_probed_at ? new Date(r.last_probed_at).toLocaleTimeString() : "未探测"}</div>
                        <div><span className="text-zinc-500">RTT / Exit IP:</span> {r.last_rtt_ms}ms <span className="font-mono ml-2">{r.last_exit_ip || "—"}</span></div>
                        <div className="col-span-2"><span className="text-zinc-500">Note:</span> {r.proxy_note || "—"}</div>
                      </div>
                      <div className="font-semibold text-zinc-400 mb-2">绑定用户 ({r.users.length})</div>
                      {r.users.length === 0 && <div className="text-zinc-600">无</div>}
                      <ul className="space-y-1">
                        {r.users.map(u => (
                          <li key={u.id} className="flex justify-between text-xs">
                            <span>👤 {u.username}</span>
                            <span className="text-zinc-500">
                              24h: {fmtBytes(u.traffic_24h)} · 到期 {u.expires_at?.startsWith("0001") ? "永不" : new Date(u.expires_at).toLocaleDateString()}
                            </span>
                          </li>
                        ))}
                      </ul>
                    </td>
                  </tr>
                )}
              </Fragment>
            ))}
          </tbody>
        </table>
      </div>

      {editing && (
        <PlanEditModal
          plan={editing}
          initialTab="ip"
          onClose={() => setEditing(null)}
          onSaved={() => { setEditing(null); load() }}
        />
      )}
      {showNew && (
        <PlanEditModal
          plan={null}
          initialTab="ip"
          onClose={() => setShowNew(false)}
          onSaved={() => { setShowNew(false); load() }}
        />
      )}
    </div>
  )
}
