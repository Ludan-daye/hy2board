import { useEffect, useState } from "react"
import api from "@/api/client"
import { Plus, Package, Trash2, Edit2 } from "lucide-react"
import PlanEditModal from "@/components/PlanEditModal"
import type { Plan } from "@/components/PlanEditModal"

interface PlanRow extends Plan {
  users_count?: number
}

function fmtBytes(n: number): string {
  if (n === 0) return "∞"
  const units = ["B", "KB", "MB", "GB", "TB"]
  let i = 0
  let v = n
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return v.toFixed(0) + " " + units[i]
}

function fmtRules(p: PlanRow): string {
  const parts: string[] = []
  if (p.rule_ai) parts.push("AI")
  if (p.rule_streaming) parts.push("Stream")
  if (p.rule_china) parts.push("CN")
  if (p.rule_ad_block) parts.push("Ad")
  return parts.join("+") || "—"
}

export default function Plans() {
  const [rows, setRows] = useState<PlanRow[]>([])
  const [editing, setEditing] = useState<PlanRow | null>(null)
  const [showNew, setShowNew] = useState(false)
  const [expanded, setExpanded] = useState<number | null>(null)

  const load = () => api.get("/admin/plans").then((r) => setRows(r.data))
  useEffect(() => {
    load()
  }, [])

  const del = async (r: PlanRow) => {
    if (!confirm(`Delete plan "${r.name}"?`)) return
    try {
      await api.delete(`/admin/plans/${r.ID}`)
      load()
    } catch (e: unknown) {
      const err = e as { response?: { status?: number; data?: { users_count?: number } } }
      if (err.response?.status === 409) {
        alert(`Plan is in use by ${err.response.data?.users_count} users. Reassign them first.`)
      } else {
        alert(String(e))
      }
    }
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-xl font-semibold">Plans</h2>
        <button
          onClick={() => setShowNew(true)}
          className="flex items-center gap-2 px-3 py-1.5 bg-clay text-white rounded-lg text-sm font-medium hover:bg-clay-hover"
        >
          <Plus size={14} /> Add Plan
        </button>
      </div>
      <div className="bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden">
        <table className="w-full text-sm">
          <thead className="text-xs text-zinc-500 uppercase">
            <tr>
              <th className="text-left p-3">Name</th>
              <th className="text-right p-3">Traffic</th>
              <th className="text-right p-3">Duration</th>
              <th className="text-left p-3">Nodes</th>
              <th className="text-left p-3">Rules</th>
              <th className="text-left p-3">IP</th>
              <th className="text-left p-3">备注</th>
              <th className="text-right p-3">Users</th>
              <th className="p-3"></th>
            </tr>
          </thead>
          <tbody>
            {rows.map((r) => (
              <>
                <tr
                  key={r.ID}
                  className="border-t border-zinc-800 hover:bg-zinc-800/30 cursor-pointer"
                  onClick={() => setExpanded(expanded === r.ID ? null : r.ID!)}
                >
                  <td className="p-3">
                    <span className="flex items-center gap-2">
                      <Package size={14} className="text-zinc-500" />
                      {r.name}
                    </span>
                  </td>
                  <td className="p-3 text-right font-mono">{fmtBytes(r.traffic_limit)}</td>
                  <td className="p-3 text-right font-mono">{r.duration_days}d</td>
                  <td className="p-3 font-mono text-xs text-zinc-400">{r.node_ids || "all"}</td>
                  <td className="p-3 text-xs text-zinc-400">{fmtRules(r)}</td>
                  <td className="p-3 font-mono text-xs text-zinc-300">
                    {r.proxy_host ? `${r.proxy_host}:${r.proxy_port}` : <span className="text-zinc-600">共享</span>}
                  </td>
                  <td className="p-3 text-xs text-zinc-400 max-w-[140px] truncate" title={r.proxy_note || ""}>
                    {r.proxy_note || "—"}
                  </td>
                  <td className="p-3 text-right">{r.users_count ?? 0}</td>
                  <td className="p-3 text-right space-x-2">
                    <button
                      onClick={(e) => { e.stopPropagation(); setEditing(r) }}
                      className="p-1.5 rounded hover:bg-zinc-700 text-zinc-400 hover:text-ink"
                    >
                      <Edit2 size={14} />
                    </button>
                    <button
                      onClick={(e) => { e.stopPropagation(); del(r) }}
                      className="p-1.5 rounded hover:bg-zinc-700 text-zinc-400 hover:text-red-400"
                    >
                      <Trash2 size={14} />
                    </button>
                  </td>
                </tr>
                {expanded === r.ID && (
                  <tr className="border-t border-zinc-800 bg-zinc-900/40">
                    <td colSpan={9} className="p-4 text-xs">
                      <div className="grid grid-cols-2 gap-x-6 gap-y-2">
                        <div><span className="text-zinc-500">Type:</span> <span className="font-mono">{r.proxy_type || "—"}</span></div>
                        <div><span className="text-zinc-500">Username:</span> <span className="font-mono">{r.proxy_username || "—"}</span></div>
                        <div><span className="text-zinc-500">Host:Port:</span> <span className="font-mono">{r.proxy_host ? `${r.proxy_host}:${r.proxy_port}` : "—"}</span></div>
                        <div><span className="text-zinc-500">Password:</span> <span className="font-mono">{r.proxy_host ? "(已设置)" : "(未设置)"}</span></div>
                        <div className="col-span-2"><span className="text-zinc-500">Note:</span> <span>{r.proxy_note || "—"}</span></div>
                      </div>
                      <div className="mt-3 text-zinc-500">
                        点击 ✏️ 编辑详细信息（包含改密码）
                      </div>
                    </td>
                  </tr>
                )}
              </>
            ))}
          </tbody>
        </table>
      </div>
      {editing && <PlanEditModal plan={editing} onClose={() => setEditing(null)} onSaved={load} />}
      {showNew && <PlanEditModal plan={null} onClose={() => setShowNew(false)} onSaved={load} />}
    </div>
  )
}
