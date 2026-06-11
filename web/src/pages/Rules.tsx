import { useEffect, useState } from "react"
import api from "@/api/client"
import { Circle, Edit2, Eye, Plus, Save, Trash2, X } from "lucide-react"

type RuleKind = "DOMAIN" | "DOMAIN-SUFFIX" | "DOMAIN-KEYWORD" | "IP-CIDR" | "GEOIP"
type RulePolicy = "Auto" | "HongKong" | "AI" | "Streaming" | "Global" | "DIRECT" | "REJECT"

interface RoutingRule {
  ID: number
  UpdatedAt?: string
  enabled: boolean
  name: string
  kind: RuleKind
  value: string
  policy: RulePolicy
  sort_order: number
  note: string
}

interface RuleForm {
  enabled: boolean
  name: string
  kind: RuleKind
  value: string
  policy: RulePolicy
  sort_order: number
  note: string
}

interface PreviewResult {
  clash: string
  surge: string
  shadowrocket: string
}

const kinds: RuleKind[] = ["DOMAIN", "DOMAIN-SUFFIX", "DOMAIN-KEYWORD", "IP-CIDR", "GEOIP"]
const policies: RulePolicy[] = ["Auto", "HongKong", "AI", "Streaming", "Global", "DIRECT", "REJECT"]

const emptyForm: RuleForm = {
  enabled: true,
  name: "",
  kind: "DOMAIN-SUFFIX",
  value: "",
  policy: "Auto",
  sort_order: 0,
  note: "",
}

function apiError(err: unknown): string {
  const e = err as { response?: { data?: { error?: string } }; message?: string }
  return e.response?.data?.error || e.message || String(err)
}

function formatTime(value?: string): string {
  if (!value) return "-"
  return new Date(value).toLocaleString()
}

export default function Rules() {
  const [rules, setRules] = useState<RoutingRule[]>([])
  const [form, setForm] = useState<RuleForm>({ ...emptyForm })
  const [showForm, setShowForm] = useState(false)
  const [editId, setEditId] = useState<number | null>(null)
  const [loading, setLoading] = useState(false)
  const [msg, setMsg] = useState("")
  const [err, setErr] = useState("")
  const [preview, setPreview] = useState<PreviewResult | null>(null)

  const load = () => api.get<RoutingRule[]>("/admin/routing-rules").then((res) => setRules(res.data || []))

  useEffect(() => {
    load()
  }, [])

  const showMsg = (text: string) => {
    setMsg(text)
    setErr("")
    window.setTimeout(() => setMsg(""), 2500)
  }

  const showErr = (text: string) => {
    setErr(text)
    setMsg("")
  }

  const resetForm = () => {
    setForm({ ...emptyForm })
    setShowForm(false)
    setEditId(null)
    setPreview(null)
    setErr("")
  }

  const startCreate = () => {
    setForm({ ...emptyForm })
    setEditId(null)
    setPreview(null)
    setShowForm(true)
    setErr("")
  }

  const startEdit = (rule: RoutingRule) => {
    setForm({
      enabled: rule.enabled,
      name: rule.name,
      kind: rule.kind,
      value: rule.value,
      policy: rule.policy,
      sort_order: rule.sort_order || 0,
      note: rule.note || "",
    })
    setEditId(rule.ID)
    setPreview(null)
    setShowForm(true)
    setErr("")
  }

  const submit = async () => {
    if (!form.name.trim() || !form.value.trim()) {
      showErr("Name and match value are required")
      return
    }
    setLoading(true)
    try {
      if (editId) {
        await api.put(`/admin/routing-rules/${editId}`, form)
        showMsg("Rule updated")
      } else {
        await api.post("/admin/routing-rules", form)
        showMsg("Rule created")
      }
      resetForm()
      await load()
    } catch (e) {
      showErr(apiError(e))
    } finally {
      setLoading(false)
    }
  }

  const remove = async (rule: RoutingRule) => {
    if (!confirm(`Delete routing rule "${rule.name}"?`)) return
    try {
      await api.delete(`/admin/routing-rules/${rule.ID}`)
      showMsg("Rule deleted")
      await load()
    } catch (e) {
      showErr(apiError(e))
    }
  }

  const doPreview = async () => {
    setLoading(true)
    try {
      const res = await api.post<PreviewResult>("/admin/routing-rules/preview", form)
      setPreview(res.data)
      setErr("")
    } catch (e) {
      showErr(apiError(e))
    } finally {
      setLoading(false)
    }
  }

  const inputCls = "px-3 py-2 bg-black border border-zinc-700 rounded-lg text-sm text-white focus:outline-none focus:border-zinc-500"
  const ruleSummary = {
    total: rules.length,
    enabled: rules.filter((r) => r.enabled).length,
    disabled: rules.filter((r) => !r.enabled).length,
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h2 className="text-xl font-semibold">Rules</h2>
          <div className="mt-1 text-xs text-zinc-500">
            {ruleSummary.total} total · {ruleSummary.enabled} enabled · {ruleSummary.disabled} disabled
          </div>
        </div>
        <button
          onClick={startCreate}
          className="flex items-center gap-2 px-3 py-1.5 bg-white text-black rounded-lg text-sm font-medium hover:bg-zinc-200"
        >
          <Plus size={14} /> Add Rule
        </button>
      </div>

      {msg && <div className="mb-4 px-4 py-2 bg-green-500/10 border border-green-500/20 rounded-lg text-green-400 text-sm">{msg}</div>}
      {err && <div className="mb-4 px-4 py-2 bg-red-500/10 border border-red-500/20 rounded-lg text-red-300 text-sm">{err}</div>}

      {showForm && (
        <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-4 mb-4">
          <div className="flex items-center justify-between mb-3">
            <h3 className="text-sm font-medium text-zinc-400">{editId ? "Edit Rule" : "New Rule"}</h3>
            <button onClick={resetForm} className="text-zinc-500 hover:text-white" title="Close">
              <X size={14} />
            </button>
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-4 gap-3 mb-3">
            <div>
              <label className="text-xs text-zinc-500 mb-1 block">Name *</label>
              <input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} className={inputCls + " w-full"} />
            </div>
            <div>
              <label className="text-xs text-zinc-500 mb-1 block">Rule Type</label>
              <select value={form.kind} onChange={(e) => setForm({ ...form, kind: e.target.value as RuleKind })} className={inputCls + " w-full"}>
                {kinds.map((k) => <option key={k} value={k}>{k}</option>)}
              </select>
            </div>
            <div>
              <label className="text-xs text-zinc-500 mb-1 block">Policy</label>
              <select value={form.policy} onChange={(e) => setForm({ ...form, policy: e.target.value as RulePolicy })} className={inputCls + " w-full"}>
                {policies.map((p) => <option key={p} value={p}>{p}</option>)}
              </select>
            </div>
            <div>
              <label className="text-xs text-zinc-500 mb-1 block">Sort Order</label>
              <input type="number" value={form.sort_order} onChange={(e) => setForm({ ...form, sort_order: Number(e.target.value) })} className={inputCls + " w-full"} />
            </div>
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-3 gap-3 mb-3">
            <div>
              <label className="text-xs text-zinc-500 mb-1 block">Match Value *</label>
              <input value={form.value} onChange={(e) => setForm({ ...form, value: e.target.value })} className={inputCls + " w-full"} />
            </div>
            <div>
              <label className="text-xs text-zinc-500 mb-1 block">Note</label>
              <input value={form.note} onChange={(e) => setForm({ ...form, note: e.target.value })} className={inputCls + " w-full"} />
            </div>
            <label className="flex items-end gap-2 pb-2 text-sm text-zinc-400 cursor-pointer">
              <input
                type="checkbox"
                checked={form.enabled}
                onChange={(e) => setForm({ ...form, enabled: e.target.checked })}
                className="w-4 h-4 rounded border-zinc-600 bg-black"
              />
              Enabled
            </label>
          </div>

          <div className="flex flex-wrap gap-2">
            <button
              onClick={submit}
              disabled={loading}
              className="inline-flex items-center gap-2 px-4 py-2 bg-white text-black rounded-lg text-sm font-medium hover:bg-zinc-200 disabled:opacity-60"
            >
              <Save size={14} /> {editId ? "Save Changes" : "Create Rule"}
            </button>
            <button
              onClick={doPreview}
              disabled={loading}
              className="inline-flex items-center gap-2 px-4 py-2 bg-zinc-800 text-zinc-200 rounded-lg text-sm hover:bg-zinc-700 disabled:opacity-60"
            >
              <Eye size={14} /> Preview
            </button>
            <button onClick={resetForm} className="px-4 py-2 text-sm text-zinc-400 hover:text-white">Cancel</button>
          </div>

          {preview && (
            <div className="grid grid-cols-1 lg:grid-cols-3 gap-3 mt-4">
              {(["clash", "surge", "shadowrocket"] as const).map((name) => (
                <div key={name} className="bg-black border border-zinc-800 rounded-lg p-3">
                  <div className="text-xs text-zinc-500 uppercase mb-2">{name}</div>
                  <pre className="text-xs text-zinc-200 whitespace-pre-wrap break-all">{preview[name]}</pre>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      <div className="bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-zinc-800 text-zinc-500">
              <th className="text-left p-4 w-[120px]">Status</th>
              <th className="text-left p-4">Name</th>
              <th className="text-left p-4">Match</th>
              <th className="text-left p-4">Policy</th>
              <th className="text-left p-4">Sort</th>
              <th className="text-left p-4">Note</th>
              <th className="text-left p-4">Updated</th>
              <th className="text-right p-4">Actions</th>
            </tr>
          </thead>
          <tbody>
            {rules.length === 0 && (
              <tr>
                <td colSpan={8} className="p-8 text-center text-zinc-600">No custom routing rules.</td>
              </tr>
            )}
            {rules.map((rule) => (
              <tr key={rule.ID} className={"border-b border-zinc-800/50 hover:bg-zinc-800/30" + (editId === rule.ID ? " bg-zinc-800/20" : "")}>
                <td className="p-4">
                  <span className={`inline-flex items-center gap-2 px-2.5 py-1 rounded border text-[11px] font-medium ${rule.enabled ? "bg-green-500/10 text-green-400 border-green-500/20" : "bg-zinc-800 text-zinc-500 border-zinc-700"}`}>
                    <Circle size={8} className={rule.enabled ? "fill-green-500 text-green-500" : "fill-zinc-500 text-zinc-500"} />
                    {rule.enabled ? "Enabled" : "Disabled"}
                  </span>
                </td>
                <td className="p-4">
                  <p className="font-medium">{rule.name}</p>
                  <span className="text-xs text-zinc-600">#{rule.ID}</span>
                </td>
                <td className="p-4">
                  <div className="text-xs text-zinc-500">{rule.kind}</div>
                  <div className="font-mono text-xs text-zinc-300 break-all">{rule.value}</div>
                </td>
                <td className="p-4">
                  <span className="px-2 py-0.5 rounded text-xs bg-blue-500/10 text-blue-400">{rule.policy}</span>
                </td>
                <td className="p-4 text-zinc-400">{rule.sort_order}</td>
                <td className="p-4 text-zinc-400 max-w-[240px] truncate" title={rule.note}>{rule.note || "-"}</td>
                <td className="p-4 text-xs text-zinc-500">{formatTime(rule.UpdatedAt)}</td>
                <td className="p-4">
                  <div className="flex items-center justify-end gap-1">
                    <button onClick={() => startEdit(rule)} title="Edit" className="p-1.5 rounded hover:bg-zinc-800 text-zinc-500 hover:text-white">
                      <Edit2 size={14} />
                    </button>
                    <button onClick={() => remove(rule)} title="Delete" className="p-1.5 rounded hover:bg-zinc-800 text-zinc-500 hover:text-red-400">
                      <Trash2 size={14} />
                    </button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
