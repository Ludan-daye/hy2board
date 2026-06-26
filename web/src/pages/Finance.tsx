import { useEffect, useState } from "react"
import api from "@/api/client"
import { LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid } from "recharts"
import { Wallet, Download, Trash2, Edit2, Plus, ReceiptText } from "lucide-react"

interface Bucket {
  month: string
  total_cents: number
  cost_cents: number
  profit_cents: number
  count: number
  cost_count: number
  new_count: number
  renew_count: number
}
interface Row {
  ID: number; user_id: number; username: string
  plan_id: number; plan_name: string
  amount_cents: number; days_added: number; kind: string
  note: string; operator: string; paid_at: string; created_at: string
}
interface CostRow {
  ID: number
  name: string
  category: string
  amount_cents: number
  note: string
  operator: string
  incurred_at: string
  created_at: string
}

const yuan = (c: number) => `¥${(c / 100).toFixed(2)}`
const dateToISO = (date: string) => `${date}T00:00:00Z`

export default function Finance() {
  const [buckets, setBuckets] = useState<Bucket[]>([])
  const [rows, setRows] = useState<Row[]>([])
  const [costs, setCosts] = useState<CostRow[]>([])
  const [total, setTotal] = useState(0)
  const [costTotal, setCostTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [costPage, setCostPage] = useState(1)
  const size = 50
  const [from, setFrom] = useState("")
  const [to, setTo] = useState("")
  const [editing, setEditing] = useState<Row | null>(null)
  const [editingCost, setEditingCost] = useState<CostRow | null>(null)
  const [showNewCost, setShowNewCost] = useState(false)

  const load = () => {
    api.get(`/admin/payments/summary?n=12`).then(r => setBuckets(r.data || []))
    const params = new URLSearchParams({ page: String(page), size: String(size) })
    if (from) params.set("from", from)
    if (to)   params.set("to", to)
    api.get(`/admin/payments?${params}`).then(r => { setRows(r.data.items || []); setTotal(r.data.total || 0) })
    const costParams = new URLSearchParams({ page: String(costPage), size: String(size) })
    if (from) costParams.set("from", from)
    if (to)   costParams.set("to", to)
    api.get(`/admin/costs?${costParams}`).then(r => { setCosts(r.data.items || []); setCostTotal(r.data.total || 0) })
  }
  useEffect(() => { load() }, [page, costPage, from, to])

  const cur = buckets[buckets.length - 1]
  const stats = cur ? {
    total: cur.total_cents, cost: cur.cost_cents || 0, profit: cur.profit_cents ?? (cur.total_cents - (cur.cost_cents || 0)), count: cur.count,
    avg: cur.count > 0 ? cur.total_cents / cur.count : 0,
    newC: cur.new_count, renewC: cur.renew_count, costCount: cur.cost_count || 0,
  } : null

  const exportCSV = async () => {
    const params = new URLSearchParams()
    if (from) params.set("from", from)
    if (to)   params.set("to", to)
    const r = await api.get(`/admin/payments.csv?${params}`, { responseType: 'blob' })
    const url = URL.createObjectURL(r.data)
    const a = document.createElement('a')
    a.href = url
    a.download = `payments-${new Date().toISOString().slice(0,10)}.csv`
    a.click()
    URL.revokeObjectURL(url)
  }

  const del = async (id: number) => {
    if (!confirm("删除这笔记录？(软删除)")) return
    await api.delete(`/admin/payments/${id}`)
    load()
  }

  const delCost = async (id: number) => {
    if (!confirm("删除这笔成本？(软删除)")) return
    await api.delete(`/admin/costs/${id}`)
    load()
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-xl font-semibold flex items-center gap-2">
          <Wallet size={18} /> 财务
        </h2>
        <div className="flex items-center gap-2">
          <button onClick={() => setShowNewCost(true)} className="flex items-center gap-2 px-3 py-1.5 bg-zinc-800 text-ink border border-zinc-700 rounded-lg text-sm font-medium hover:bg-zinc-700">
            <Plus size={14} /> 成本
          </button>
          <button onClick={exportCSV} className="flex items-center gap-2 px-3 py-1.5 bg-clay text-white rounded-lg text-sm font-medium hover:bg-clay-hover">
            <Download size={14} /> Export CSV
          </button>
        </div>
      </div>

      {stats && (
        <div className="grid grid-cols-1 md:grid-cols-4 gap-3 mb-6">
          <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-4">
            <div className="text-xs text-zinc-500 mb-1">本月收入</div>
            <div className="text-2xl font-semibold">{yuan(stats.total)}</div>
          </div>
          <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-4">
            <div className="text-xs text-zinc-500 mb-1">本月成本</div>
            <div className="text-2xl font-semibold text-red-300">{yuan(stats.cost)}</div>
          </div>
          <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-4">
            <div className="text-xs text-zinc-500 mb-1">本月利润</div>
            <div className={`text-2xl font-semibold ${stats.profit >= 0 ? "text-green-300" : "text-red-300"}`}>{yuan(stats.profit)}</div>
          </div>
          <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-4">
            <div className="text-xs text-zinc-500 mb-1">笔数</div>
            <div className="text-sm">收款 <b>{stats.count}</b> · 成本 <b>{stats.costCount}</b></div>
            <div className="text-xs text-zinc-500 mt-1">新购 {stats.newC} · 续费 {stats.renewC}</div>
          </div>
        </div>
      )}

      <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-4 mb-6 h-64">
        <ResponsiveContainer width="100%" height="100%">
          <LineChart data={buckets.map(b => ({ ...b, total_yuan: b.total_cents / 100, cost_yuan: (b.cost_cents || 0) / 100, profit_yuan: (b.profit_cents ?? (b.total_cents - (b.cost_cents || 0))) / 100 }))}>
            <CartesianGrid stroke="#E5E0D6" strokeDasharray="3 3" />
            <XAxis dataKey="month" stroke="#6B6862" tick={{ fontSize: 12 }} />
            <YAxis stroke="#6B6862" tick={{ fontSize: 12 }} />
            <Tooltip contentStyle={{ background: "#18181b", border: "1px solid #E5E0D6" }} />
            <Line type="monotone" name="收入" dataKey="total_yuan" stroke="#C96442" strokeWidth={2} dot={{ r: 3 }} />
            <Line type="monotone" name="成本" dataKey="cost_yuan" stroke="#BE3A31" strokeWidth={2} dot={{ r: 3 }} />
            <Line type="monotone" name="利润" dataKey="profit_yuan" stroke="#3F8A4D" strokeWidth={2} dot={{ r: 3 }} />
          </LineChart>
        </ResponsiveContainer>
      </div>

      <div className="flex gap-2 mb-3">
        <input type="date" value={from} onChange={e => setFrom(e.target.value)} className="px-3 py-2 bg-surface border border-zinc-700 rounded text-sm text-ink" />
        <span className="self-center text-zinc-500">至</span>
        <input type="date" value={to} onChange={e => setTo(e.target.value)} className="px-3 py-2 bg-surface border border-zinc-700 rounded text-sm text-ink" />
        <button onClick={() => { setFrom(""); setTo("") }} className="px-3 py-2 text-sm text-zinc-400 hover:text-ink">清除</button>
      </div>

      <div className="bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden">
        <table className="w-full text-sm">
          <thead className="text-xs text-zinc-500 uppercase">
            <tr>
              <th className="text-left p-3">日期</th>
              <th className="text-left p-3">用户</th>
              <th className="text-left p-3">Plan</th>
              <th className="text-right p-3">金额</th>
              <th className="text-left p-3">类型</th>
              <th className="text-left p-3">备注</th>
              <th className="text-right p-3"></th>
            </tr>
          </thead>
          <tbody>
            {rows.length === 0 && <tr><td colSpan={7} className="p-6 text-center text-zinc-600">无记录</td></tr>}
            {rows.map(r => (
              <tr key={r.ID} className="border-t border-zinc-800">
                <td className="p-3 font-mono text-xs">{r.paid_at.slice(0,10)}</td>
                <td className="p-3">{r.username || `#${r.user_id}`}</td>
                <td className="p-3 text-xs text-zinc-400">{r.plan_name || "-"}</td>
                <td className="p-3 text-right font-mono">{yuan(r.amount_cents)}</td>
                <td className="p-3 text-xs">{r.kind === "new" ? "新购" : "续费"}</td>
                <td className="p-3 text-xs text-zinc-400">{r.note || "-"}</td>
                <td className="p-3 text-right">
                  <button onClick={() => setEditing(r)} className="p-1 text-zinc-400 hover:text-ink"><Edit2 size={14} /></button>
                  <button onClick={() => del(r.ID)} className="p-1 text-zinc-400 hover:text-red-400 ml-1"><Trash2 size={14} /></button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        <div className="p-3 flex items-center justify-between text-xs text-zinc-500">
          <span>共 {total} 条</span>
          <div className="flex gap-2">
            <button disabled={page <= 1} onClick={() => setPage(p => p - 1)} className="px-2 py-1 disabled:opacity-30">‹</button>
            <span>{page} / {Math.max(1, Math.ceil(total / size))}</span>
            <button disabled={page * size >= total} onClick={() => setPage(p => p + 1)} className="px-2 py-1 disabled:opacity-30">›</button>
          </div>
        </div>
      </div>

      {editing && (
        <EditPaymentModal row={editing} onClose={() => setEditing(null)} onSaved={() => { setEditing(null); load() }} />
      )}
      <div className="mt-6 bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden">
        <div className="p-3 flex items-center justify-between border-b border-zinc-800">
          <h3 className="text-sm font-semibold flex items-center gap-2"><ReceiptText size={15} /> 成本记录</h3>
          <button onClick={() => setShowNewCost(true)} className="flex items-center gap-1 px-2 py-1 text-xs bg-zinc-800 border border-zinc-700 rounded hover:bg-zinc-700">
            <Plus size={12} /> 新增
          </button>
        </div>
        <table className="w-full text-sm">
          <thead className="text-xs text-zinc-500 uppercase">
            <tr>
              <th className="text-left p-3">日期</th>
              <th className="text-left p-3">名称</th>
              <th className="text-left p-3">分类</th>
              <th className="text-right p-3">金额</th>
              <th className="text-left p-3">备注</th>
              <th className="text-right p-3"></th>
            </tr>
          </thead>
          <tbody>
            {costs.length === 0 && <tr><td colSpan={6} className="p-6 text-center text-zinc-600">无成本记录</td></tr>}
            {costs.map(cost => (
              <tr key={cost.ID} className="border-t border-zinc-800">
                <td className="p-3 font-mono text-xs">{cost.incurred_at.slice(0,10)}</td>
                <td className="p-3">{cost.name}</td>
                <td className="p-3 text-xs text-zinc-400">{cost.category || "-"}</td>
                <td className="p-3 text-right font-mono text-red-300">{yuan(cost.amount_cents)}</td>
                <td className="p-3 text-xs text-zinc-400">{cost.note || "-"}</td>
                <td className="p-3 text-right">
                  <button onClick={() => setEditingCost(cost)} className="p-1 text-zinc-400 hover:text-ink"><Edit2 size={14} /></button>
                  <button onClick={() => delCost(cost.ID)} className="p-1 text-zinc-400 hover:text-red-400 ml-1"><Trash2 size={14} /></button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        <div className="p-3 flex items-center justify-between text-xs text-zinc-500">
          <span>共 {costTotal} 条</span>
          <div className="flex gap-2">
            <button disabled={costPage <= 1} onClick={() => setCostPage(p => p - 1)} className="px-2 py-1 disabled:opacity-30">‹</button>
            <span>{costPage} / {Math.max(1, Math.ceil(costTotal / size))}</span>
            <button disabled={costPage * size >= costTotal} onClick={() => setCostPage(p => p + 1)} className="px-2 py-1 disabled:opacity-30">›</button>
          </div>
        </div>
      </div>

      {showNewCost && (
        <EditCostModal onClose={() => setShowNewCost(false)} onSaved={() => { setShowNewCost(false); load() }} />
      )}
      {editingCost && (
        <EditCostModal row={editingCost} onClose={() => setEditingCost(null)} onSaved={() => { setEditingCost(null); load() }} />
      )}
    </div>
  )
}

function EditPaymentModal({ row, onClose, onSaved }: { row: Row; onClose: () => void; onSaved: () => void }) {
  const [amountYuan, setAmountYuan] = useState((row.amount_cents / 100).toFixed(2))
  const [note, setNote] = useState(row.note)
  const [kind, setKind] = useState(row.kind as "new" | "renew")
  const [paidAt, setPaidAt] = useState(row.paid_at.slice(0, 10))
  const [busy, setBusy] = useState(false)
  const save = async () => {
    setBusy(true)
    try {
      await api.put(`/admin/payments/${row.ID}`, {
        amount_cents: Math.round(parseFloat(amountYuan || "0") * 100),
        note, kind, paid_at: dateToISO(paidAt),
      })
      onSaved()
    } finally { setBusy(false) }
  }
  return (
    <div className="fixed inset-0 bg-ink/40 flex items-center justify-center z-50">
      <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-6 w-[400px]">
        <h3 className="text-lg font-semibold mb-4">编辑缴费 #{row.ID}</h3>
        <div className="space-y-3 text-sm">
          <label className="block"><span className="text-zinc-400">金额 (¥)</span><input type="number" step="0.01" value={amountYuan} onChange={e => setAmountYuan(e.target.value)} className="w-full mt-1 px-3 py-2 bg-surface border border-zinc-700 rounded text-ink" /></label>
          <label className="block"><span className="text-zinc-400">备注</span><input value={note} onChange={e => setNote(e.target.value)} className="w-full mt-1 px-3 py-2 bg-surface border border-zinc-700 rounded text-ink" /></label>
          <label className="block"><span className="text-zinc-400">缴费日期</span><input type="date" value={paidAt} onChange={e => setPaidAt(e.target.value)} className="w-full mt-1 px-3 py-2 bg-surface border border-zinc-700 rounded text-ink" /></label>
          <div className="flex gap-3"><label className="flex items-center gap-1"><input type="radio" checked={kind === "renew"} onChange={() => setKind("renew")} /> 续费</label><label className="flex items-center gap-1"><input type="radio" checked={kind === "new"} onChange={() => setKind("new")} /> 新购</label></div>
        </div>
        <div className="flex justify-end gap-2 mt-4">
          <button onClick={onClose} className="px-3 py-2 text-sm text-zinc-400">Cancel</button>
          <button onClick={save} disabled={busy} className="px-3 py-2 bg-clay text-white rounded text-sm font-medium">Save</button>
        </div>
      </div>
    </div>
  )
}

function EditCostModal({ row, onClose, onSaved }: { row?: CostRow; onClose: () => void; onSaved: () => void }) {
  const [name, setName] = useState(row?.name || "")
  const [category, setCategory] = useState(row?.category || "")
  const [amountYuan, setAmountYuan] = useState(row ? (row.amount_cents / 100).toFixed(2) : "")
  const [note, setNote] = useState(row?.note || "")
  const [incurredAt, setIncurredAt] = useState(row ? row.incurred_at.slice(0, 10) : new Date().toISOString().slice(0, 10))
  const [busy, setBusy] = useState(false)
  const save = async () => {
    if (!name.trim()) return
    setBusy(true)
    try {
      const payload = {
        name: name.trim(),
        category: category.trim(),
        amount_cents: Math.round(parseFloat(amountYuan || "0") * 100),
        note,
        incurred_at: dateToISO(incurredAt),
      }
      if (row) await api.put(`/admin/costs/${row.ID}`, payload)
      else await api.post(`/admin/costs`, payload)
      onSaved()
    } finally { setBusy(false) }
  }
  return (
    <div className="fixed inset-0 bg-ink/40 flex items-center justify-center z-50">
      <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-6 w-[420px]">
        <h3 className="text-lg font-semibold mb-4">{row ? `编辑成本 #${row.ID}` : "新增成本"}</h3>
        <div className="space-y-3 text-sm">
          <label className="block"><span className="text-zinc-400">名称</span><input value={name} onChange={e => setName(e.target.value)} className="w-full mt-1 px-3 py-2 bg-surface border border-zinc-700 rounded text-ink" autoFocus /></label>
          <label className="block"><span className="text-zinc-400">金额 (¥)</span><input type="number" step="0.01" value={amountYuan} onChange={e => setAmountYuan(e.target.value)} className="w-full mt-1 px-3 py-2 bg-surface border border-zinc-700 rounded text-ink" /></label>
          <label className="block"><span className="text-zinc-400">分类</span><input value={category} onChange={e => setCategory(e.target.value)} placeholder="服务器 / IP / 机场 / 其他" className="w-full mt-1 px-3 py-2 bg-surface border border-zinc-700 rounded text-ink" /></label>
          <label className="block"><span className="text-zinc-400">发生日期</span><input type="date" value={incurredAt} onChange={e => setIncurredAt(e.target.value)} className="w-full mt-1 px-3 py-2 bg-surface border border-zinc-700 rounded text-ink" /></label>
          <label className="block"><span className="text-zinc-400">备注</span><input value={note} onChange={e => setNote(e.target.value)} className="w-full mt-1 px-3 py-2 bg-surface border border-zinc-700 rounded text-ink" /></label>
        </div>
        <div className="flex justify-end gap-2 mt-4">
          <button onClick={onClose} className="px-3 py-2 text-sm text-zinc-400">Cancel</button>
          <button onClick={save} disabled={busy || !name.trim()} className="px-3 py-2 bg-clay text-white rounded text-sm font-medium disabled:opacity-50">Save</button>
        </div>
      </div>
    </div>
  )
}
