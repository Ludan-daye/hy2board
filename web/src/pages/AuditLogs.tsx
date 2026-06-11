import { useEffect, useState } from "react"
import api from "@/api/client"
import { ScrollText, RefreshCw } from "lucide-react"

interface AuditLog {
  ID: number
  CreatedAt: string
  actor: string
  action: string
  entity: string
  entity_id: number
  entity_name: string
  detail: string
  ip: string
}

const actionLabels: Record<string, string> = {
  "node.create": "Create node",
  "node.update": "Update node",
  "node.delete": "Delete node",
  "user.create": "Create user",
  "user.update": "Update user",
  "user.delete": "Delete user",
  "user.reset_sub_token": "Reset subscription",
  "user.toggle": "Toggle user",
  "user.reset_traffic": "Reset traffic",
  "user.toggle_chain_proxy": "Toggle AI route",
  "user.renew": "Renew user",
  "plan.create": "Create plan",
  "plan.update": "Update plan",
  "plan.delete": "Delete plan",
  "plan.apply_to_user": "Apply plan",
  "plan.set_proxy_password": "Set proxy password",
  "payment.update": "Update payment",
  "payment.delete": "Delete payment",
  "cost.create": "Create cost",
  "cost.update": "Update cost",
  "cost.delete": "Delete cost",
  "telegram.test_admin_notice": "Test admin notice",
  "telegram.test_daily_post": "Test daily post",
  "telegram.announce_activity": "Activity announcement",
}

function formatDetail(detail: string) {
  if (!detail || detail === "null") return "-"
  try {
    const parsed = JSON.parse(detail)
    return JSON.stringify(parsed, null, 2)
  } catch {
    return detail
  }
}

function entityLabel(row: AuditLog) {
  const name = row.entity_name || (row.entity_id ? `#${row.entity_id}` : "-")
  return `${row.entity}${row.entity_id ? ` #${row.entity_id}` : ""} · ${name}`
}

export default function AuditLogs() {
  const [rows, setRows] = useState<AuditLog[]>([])
  const [total, setTotal] = useState(0)
  const [entity, setEntity] = useState("")
  const [action, setAction] = useState("")
  const [actor, setActor] = useState("")
  const [loading, setLoading] = useState(false)

  const load = async () => {
    setLoading(true)
    try {
      const params = new URLSearchParams({ limit: "120" })
      if (entity) params.set("entity", entity)
      if (action) params.set("action", action)
      if (actor) params.set("actor", actor)
      const r = await api.get(`/admin/audit-logs?${params}`)
      setRows(r.data.items || [])
      setTotal(r.data.total || 0)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { load() }, [entity, action])

  const actions = Array.from(new Set(rows.map(r => r.action).concat(Object.keys(actionLabels)))).sort()

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-xl font-semibold flex items-center gap-2">
          <ScrollText size={18} /> Audit Logs
          <span className="text-xs text-zinc-500 font-normal ml-3">{total} records</span>
        </h2>
        <button
          onClick={load}
          className="flex items-center gap-2 px-3 py-1.5 bg-white text-black rounded-lg text-sm font-medium hover:bg-zinc-200 disabled:opacity-60"
          disabled={loading}
        >
          <RefreshCw size={14} className={loading ? "animate-spin" : ""} /> Refresh
        </button>
      </div>

      <div className="flex flex-wrap gap-2 mb-3">
        <select value={entity} onChange={e => setEntity(e.target.value)} className="px-3 py-2 bg-black border border-zinc-700 rounded text-sm text-white">
          <option value="">All entities</option>
          <option value="node">Node</option>
          <option value="user">User</option>
          <option value="plan">Plan</option>
          <option value="payment">Payment</option>
          <option value="cost">Cost</option>
          <option value="telegram">Telegram</option>
        </select>
        <select value={action} onChange={e => setAction(e.target.value)} className="px-3 py-2 bg-black border border-zinc-700 rounded text-sm text-white">
          <option value="">All actions</option>
          {actions.map(a => <option key={a} value={a}>{actionLabels[a] || a}</option>)}
        </select>
        <input
          value={actor}
          onChange={e => setActor(e.target.value)}
          onKeyDown={e => { if (e.key === "Enter") load() }}
          placeholder="Actor"
          className="px-3 py-2 bg-black border border-zinc-700 rounded text-sm text-white"
        />
        <button onClick={() => { setActor(""); setAction(""); setEntity("") }} className="px-3 py-2 text-sm text-zinc-400 hover:text-white">Clear</button>
      </div>

      <div className="bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden">
        <table className="w-full text-sm">
          <thead className="text-xs text-zinc-500 uppercase">
            <tr>
              <th className="text-left p-3">Time</th>
              <th className="text-left p-3">Actor</th>
              <th className="text-left p-3">Action</th>
              <th className="text-left p-3">Target</th>
              <th className="text-left p-3">IP</th>
              <th className="text-left p-3">Detail</th>
            </tr>
          </thead>
          <tbody>
            {rows.length === 0 && <tr><td colSpan={6} className="p-6 text-center text-zinc-600">No audit records yet.</td></tr>}
            {rows.map(row => (
              <tr key={row.ID} className="border-t border-zinc-800 align-top">
                <td className="p-3 font-mono text-xs text-zinc-400 whitespace-nowrap">{new Date(row.CreatedAt).toLocaleString()}</td>
                <td className="p-3 text-zinc-200">{row.actor || "-"}</td>
                <td className="p-3">
                  <span className="px-2 py-0.5 rounded bg-blue-500/10 text-blue-300 text-xs">{actionLabels[row.action] || row.action}</span>
                </td>
                <td className="p-3 text-zinc-300">{entityLabel(row)}</td>
                <td className="p-3 font-mono text-xs text-zinc-500">{row.ip || "-"}</td>
                <td className="p-3">
                  <pre className="max-w-[360px] whitespace-pre-wrap break-words text-[11px] leading-relaxed text-zinc-500">{formatDetail(row.detail)}</pre>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
