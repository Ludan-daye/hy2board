import { copyToClipboard } from "@/lib/clipboard"
import { useEffect, useState } from "react"
import { useSearchParams } from "react-router-dom"
import api from "@/api/client"
import { Plus, Copy, Check, ChevronUp, Link, Server, ArrowUp, ArrowDown, Activity, Edit2 } from "lucide-react"
import UserEditModal from "@/components/UserEditModal"
import type { EditableUser } from "@/components/UserEditModal"
import PaymentDialog from "@/components/PaymentDialog"

function formatBytes(bytes: number): string {
  if (bytes === 0) return "Unlimited"
  const k = 1024
  const sizes = ["B", "KB", "MB", "GB", "TB"]
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return (bytes / Math.pow(k, i)).toFixed(1) + " " + sizes[i]
}

function fmt(bytes: number): string {
  if (!bytes || bytes === 0) return "0 B"
  const k = 1024
  const sizes = ["B", "KB", "MB", "GB", "TB"]
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return (bytes / Math.pow(k, i)).toFixed(1) + " " + sizes[i]
}

interface NodeStat {
  id: number; name: string; host: string; port: number; healthy: boolean; online: number
  traffic: Record<string, { tx: number; rx: number }> | null
}
interface StatsData {
  nodes: NodeStat[]
}

function formatDate(d: string): string {
  if (!d || d.startsWith("0001") || d.startsWith("1970")) return "Never"
  return new Date(d).toLocaleDateString()
}

export default function Users() {
  const [searchParams] = useSearchParams()
  const [users, setUsers] = useState<any[]>([])
  const [stats, setStats] = useState<StatsData | null>(null)
  const [showAdd, setShowAdd] = useState(false)
  const [form, setForm] = useState({ username: "", password: "", traffic_limit: 0, expires_at: "", chain_proxy: false })
  const [copied, setCopied] = useState("")
  const [expanded, setExpanded] = useState<number | null>(null)
  const [trafficExpanded, setTrafficExpanded] = useState<number | null>(null)
  const [pwForm, setPwForm] = useState<{ id: number; pw: string } | null>(null)
  const [editForm, setEditForm] = useState<{ id: number; traffic_limit: number; expires_at: string } | null>(null)
  const [msg, setMsg] = useState("")
  const [lastUpdate, setLastUpdate] = useState<string>("")
  const [editingUser, setEditingUser] = useState<EditableUser | null>(null)
  const [search, setSearch] = useState("")
  const [tagFilter, setTagFilter] = useState("")
  const [statusFilter, setStatusFilter] = useState("")
  const [planFilter, setPlanFilter] = useState("")
  const [allPlans, setAllPlans] = useState<{ ID: number; name: string; price_cents?: number }[]>([])
  const [selected, setSelected] = useState<Set<number>>(new Set())
  const [moreOpen, setMoreOpen] = useState<number | null>(null)
  const [payDialog, setPayDialog] = useState<{ user: any; days: number } | null>(null)

  useEffect(() => {
    const q = searchParams.get("search")
    if (q !== null) setSearch(q)
  }, [searchParams])

  const load = () => Promise.all([
    api.get("/admin/users"),
    api.get("/admin/stats"),
  ]).then(([u, s]) => {
    setUsers(u.data)
    setStats(s.data)
    setLastUpdate(new Date().toLocaleTimeString("en-US", { hour12: false }))
  })

  useEffect(() => {
    load()
    const timer = setInterval(load, 5000)
    return () => clearInterval(timer)
  }, [])

  useEffect(() => {
    if (moreOpen === null) return
    const handler = (e: MouseEvent) => {
      const target = e.target as HTMLElement
      if (!target.closest('[aria-label="More actions"]') && !target.closest('.more-dropdown')) {
        setMoreOpen(null)
      }
    }
    document.addEventListener("click", handler)
    return () => document.removeEventListener("click", handler)
  }, [moreOpen])

  useEffect(() => {
    api.get("/admin/plans").then((r) => setAllPlans(r.data))
  }, [])

  // Helper: get per-node traffic breakdown for a user
  const getUserNodeTraffic = (username: string) => {
    if (!stats) return []
    return stats.nodes.map(n => {
      const t = n.traffic?.[username]
      return {
        name: n.name,
        host: n.host,
        port: n.port,
        healthy: n.healthy,
        tx: t?.tx || 0,
        rx: t?.rx || 0,
        total: (t?.tx || 0) + (t?.rx || 0),
      }
    })
  }

  const showMsg = (text: string) => { setMsg(text); setTimeout(() => setMsg(""), 2000) }

  const add = async () => {
    if (!form.username) return
    await api.post("/admin/users", {
      username: form.username,
      password: form.password || undefined,
      traffic_limit: form.traffic_limit * 1024 * 1024 * 1024,
      expires_at: form.expires_at || undefined,
      chain_proxy: form.chain_proxy,
    })
    setShowAdd(false)
    setForm({ username: "", password: "", traffic_limit: 0, expires_at: "", chain_proxy: false })
    showMsg("User created")
    load()
  }

  const del = async (id: number) => {
    if (!confirm("Delete this user?")) return
    await api.delete(`/admin/users/${id}`)
    showMsg("User deleted")
    load()
  }

  const toggle = async (id: number) => {
    await api.post(`/admin/users/${id}/toggle`)
    load()
  }

  const resetTraffic = async (id: number) => {
    if (!confirm("Reset traffic for this user?")) return
    await api.post(`/admin/users/${id}/reset-traffic`)
    showMsg("Traffic reset")
    load()
  }

  const setPassword = async () => {
    if (!pwForm || !pwForm.pw) return
    await api.post(`/admin/users/${pwForm.id}/set-password`, { password: pwForm.pw })
    setPwForm(null)
    showMsg("Password set")
  }

  const saveEdit = async () => {
    if (!editForm) return
    await api.put(`/admin/users/${editForm.id}`, {
      traffic_limit: editForm.traffic_limit * 1024 * 1024 * 1024,
      expires_at: editForm.expires_at || undefined,
    })
    setEditForm(null)
    showMsg("User updated")
    load()
  }

  const doCopy = (text: string, label: string) => {
    copyToClipboard(text).then(() => {
      setCopied(label)
      setTimeout(() => setCopied(""), 2000)
    })
  }

  const subBase = window.location.origin + "/api/sub/"

  // Client-side filter
  const filtered = users.filter((u: any) => {
    const q = search.toLowerCase()
    if (q) {
      const hay = [u.username, u.email || "", u.notes || "", u.tags || ""].join(" ").toLowerCase()
      if (!hay.includes(q)) return false
    }
    if (tagFilter) {
      const tags = (u.tags || "").split(",").map((s: string) => s.trim()).filter(Boolean)
      if (!tags.includes(tagFilter)) return false
    }
    if (planFilter === "none" && u.plan_id) return false
    if (planFilter && planFilter !== "none" && String(u.plan_id) !== planFilter) return false
    if (statusFilter) {
      const expired = u.expires_at && !u.expires_at.startsWith("0001") && new Date(u.expires_at) < new Date()
      const liveUsed = (u.live_tx || 0) + (u.live_rx || 0)
      const over = u.traffic_limit > 0 && liveUsed >= u.traffic_limit
      if (statusFilter === "active"   && !(u.enabled && !expired && !over)) return false
      if (statusFilter === "expired"  && !expired) return false
      if (statusFilter === "over"     && !over) return false
      if (statusFilter === "disabled" && u.enabled) return false
    }
    return true
  })

  const allTags = [
    ...new Set(
      users.flatMap((u: any) =>
        (u.tags || "").split(",").map((s: string) => s.trim()).filter(Boolean),
      ),
    ),
  ]

  const toggleRow = (id: number) => {
    const next = new Set(selected)
    if (next.has(id)) next.delete(id)
    else next.add(id)
    setSelected(next)
  }

  const toggleAll = () => {
    if (selected.size === filtered.length) setSelected(new Set())
    else setSelected(new Set(filtered.map((u: any) => u.ID)))
  }

  const bulk = async (action: string) => {
    if (selected.size === 0) return
    if (action === "delete" && !confirm(`Delete ${selected.size} users? This is permanent.`)) return
    await api.post("/admin/users/bulk", { ids: [...selected], action })
    setSelected(new Set())
    load()
  }

  const inputCls = "px-3 py-2 bg-black border border-zinc-700 rounded-lg text-sm text-white focus:outline-none focus:border-zinc-500"
  const btnCls = "px-3 py-2 bg-white text-black rounded-lg text-sm font-medium hover:bg-zinc-200"

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-xl font-semibold">Users</h2>
        <div className="flex items-center gap-3">
          {lastUpdate && (
            <span className="flex items-center gap-1.5 text-xs text-zinc-500">
              <span className="relative flex h-2 w-2">
                <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75" />
                <span className="relative inline-flex rounded-full h-2 w-2 bg-green-500" />
              </span>
              Live · {lastUpdate}
            </span>
          )}
          <button onClick={() => setShowAdd(!showAdd)} className="flex items-center gap-2 px-3 py-1.5 bg-white text-black rounded-lg text-sm font-medium hover:bg-zinc-200">
            <Plus size={14} /> Add User
          </button>
        </div>
      </div>

      {msg && <div className="mb-4 px-4 py-2 bg-green-500/10 border border-green-500/20 rounded-lg text-green-400 text-sm">{msg}</div>}

      {showAdd && (
        <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-4 mb-4">
          <h3 className="text-sm font-medium text-zinc-400 mb-3">New User</h3>
          <div className="grid grid-cols-2 gap-3">
            <input placeholder="Username *" value={form.username} onChange={e => setForm({...form, username: e.target.value})} className={inputCls} />
            <input placeholder="Login Password" type="password" value={form.password} onChange={e => setForm({...form, password: e.target.value})} className={inputCls} />
            <input placeholder="Traffic Limit (GB, 0=unlimited)" type="number" value={form.traffic_limit || ""} onChange={e => setForm({...form, traffic_limit: +e.target.value})} className={inputCls} />
            <input placeholder="Expires" type="date" value={form.expires_at} onChange={e => setForm({...form, expires_at: e.target.value})} className={inputCls} />
          </div>
          <label className="flex items-center gap-2 mt-3 text-sm text-zinc-400 cursor-pointer">
            <input type="checkbox" checked={form.chain_proxy} onChange={e => setForm({...form, chain_proxy: e.target.checked})} className="w-4 h-4 rounded border-zinc-600 bg-black" />
            Rule: AI 链式代理 (走住宅 IP)
          </label>
          <div className="flex gap-2 mt-3">
            <button onClick={add} className={btnCls}>Create User</button>
            <button onClick={() => setShowAdd(false)} className="px-3 py-2 text-sm text-zinc-400 hover:text-white">Cancel</button>
          </div>
        </div>
      )}

      {pwForm && (
        <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-4 mb-4">
          <h3 className="text-sm font-medium text-zinc-400 mb-3">Set Login Password</h3>
          <div className="flex gap-2">
            <input placeholder="New password" type="password" value={pwForm.pw} onChange={e => setPwForm({...pwForm, pw: e.target.value})} className={inputCls + " flex-1"} autoFocus />
            <button onClick={setPassword} className={btnCls}>Save</button>
            <button onClick={() => setPwForm(null)} className="px-3 py-2 text-sm text-zinc-400 hover:text-white">Cancel</button>
          </div>
        </div>
      )}

      {editForm && (
        <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-4 mb-4">
          <h3 className="text-sm font-medium text-zinc-400 mb-3">Edit User</h3>
          <div className="grid grid-cols-2 gap-3">
            <input placeholder="Traffic Limit (GB)" type="number" value={editForm.traffic_limit || ""} onChange={e => setEditForm({...editForm, traffic_limit: +e.target.value})} className={inputCls} />
            <input placeholder="Expires" type="date" value={editForm.expires_at} onChange={e => setEditForm({...editForm, expires_at: e.target.value})} className={inputCls} />
          </div>
          <div className="flex gap-2 mt-3">
            <button onClick={saveEdit} className={btnCls}>Save</button>
            <button onClick={() => setEditForm(null)} className="px-3 py-2 text-sm text-zinc-400 hover:text-white">Cancel</button>
          </div>
        </div>
      )}

      <div className="flex flex-wrap gap-2 mb-4">
        <input
          type="search"
          name="hy2-user-search"
          autoComplete="off"
          data-form-type="other"
          data-lpignore="true"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Search users, email, notes, tags..."
          className="px-3 py-2 bg-black border border-zinc-700 rounded text-sm flex-1 min-w-[200px] text-white placeholder-zinc-500"
        />
        <select
          value={tagFilter}
          onChange={(e) => setTagFilter(e.target.value)}
          className="px-3 py-2 bg-black border border-zinc-700 rounded text-sm text-white"
        >
          <option value="">All tags</option>
          {allTags.map((t) => (
            <option key={t} value={t}>
              {t}
            </option>
          ))}
        </select>
        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value)}
          className="px-3 py-2 bg-black border border-zinc-700 rounded text-sm text-white"
        >
          <option value="">All status</option>
          <option value="active">Active</option>
          <option value="expired">Expired</option>
          <option value="over">Over Limit</option>
          <option value="disabled">Disabled</option>
        </select>
        <select
          value={planFilter}
          onChange={(e) => setPlanFilter(e.target.value)}
          className="px-3 py-2 bg-black border border-zinc-700 rounded text-sm text-white"
        >
          <option value="">All plans</option>
          <option value="none">No plan</option>
          {allPlans.map((p) => (
            <option key={p.ID} value={String(p.ID)}>
              {p.name}
            </option>
          ))}
        </select>
      </div>

      {selected.size > 0 && (
        <div className="mb-3 p-3 bg-zinc-800 rounded flex items-center gap-3">
          <span className="text-sm text-white">{selected.size} selected</span>
          <button
            onClick={() => bulk("enable")}
            className="px-3 py-1 text-xs bg-green-600 text-white rounded hover:bg-green-500"
          >
            Enable
          </button>
          <button
            onClick={() => bulk("disable")}
            className="px-3 py-1 text-xs bg-amber-600 text-white rounded hover:bg-amber-500"
          >
            Disable
          </button>
          <button
            onClick={() => bulk("delete")}
            className="px-3 py-1 text-xs bg-red-600 text-white rounded hover:bg-red-500"
          >
            Delete
          </button>
          <button
            onClick={() => setSelected(new Set())}
            className="px-3 py-1 text-xs text-zinc-300 hover:text-white ml-auto"
          >
            Clear
          </button>
        </div>
      )}

      <div className="space-y-2">
        {filtered.length > 0 && (
          <div className="flex items-center gap-2 px-4 py-2">
            <input
              type="checkbox"
              checked={selected.size === filtered.length && filtered.length > 0}
              onChange={toggleAll}
              className="mr-1"
              title="Select all"
            />
            <span className="text-xs text-zinc-500">Select all ({filtered.length})</span>
          </div>
        )}
        {filtered.map(u => {
          const isExpired = u.expires_at && !u.expires_at.startsWith("0001") && new Date(u.expires_at) < new Date()
          // Live traffic = HY2 节点累计 (real-time); traffic_used 是 DB 里手动重置的计数器, 平时 0.
          const liveUsed = (u.live_tx || 0) + (u.live_rx || 0)
          const trafficExceeded = u.traffic_limit > 0 && liveUsed >= u.traffic_limit
          const isActive = u.enabled && !isExpired && !trafficExceeded
          const isExp = expanded === u.ID
          const isTraffic = trafficExpanded === u.ID
          const subUrl = subBase + u.sub_token
          const nodeTraffic = getUserNodeTraffic(u.username)
          const activeNodes = nodeTraffic.filter(n => n.total > 0).length
          const currSpeed = (u.speed_tx || 0) + (u.speed_rx || 0)

          return (
            <div key={u.ID} className="bg-zinc-900 border border-zinc-800 rounded-xl overflow-visible">
              <div className="flex flex-col p-4">
                {/* Main horizontal row */}
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-4 min-w-0">
                    <input
                      type="checkbox"
                      checked={selected.has(u.ID)}
                      onChange={() => toggleRow(u.ID)}
                      className="mr-2"
                    />
                    <div>
                      <p className="font-medium flex items-center gap-2 flex-wrap">
                        {u.username}
                        {u.telegram_id ? (
                          <span
                            className="ml-1.5 text-[9px] px-1.5 py-0.5 rounded bg-sky-500/10 text-sky-400 font-mono"
                            title={`Telegram ID: ${u.telegram_id}`}
                          >
                            TG
                          </span>
                        ) : null}
                        <span className="text-xs text-zinc-500">#{u.ID}</span>
                        {(() => {
                          if (!u.plan_id) return null
                          const plan = allPlans.find(p => p.ID === u.plan_id)
                          if (!plan) return null
                          return (
                            <span
                              className="text-[10px] px-1.5 py-0.5 rounded bg-indigo-500/10 text-indigo-300 font-medium"
                              title={`Plan #${plan.ID}: ${plan.name}`}
                            >
                              {plan.name}
                            </span>
                          )
                        })()}
                        {currSpeed > 0 && (
                          <span className="flex items-center gap-1 text-xs text-green-400 font-mono">
                            <Activity size={10} className="animate-pulse" />
                            {fmt(currSpeed)}/s
                          </span>
                        )}
                      </p>
                      <p className="text-xs text-zinc-500">{formatBytes(liveUsed)} / {formatBytes(u.traffic_limit)}
                        {u.traffic_limit > 0 && (
                          <span className="inline-block w-16 h-1.5 bg-zinc-800 rounded-full ml-2 align-middle overflow-hidden">
                            <span className={"block h-full rounded-full " + (trafficExceeded ? "bg-red-500" : "bg-blue-500")} style={{ width: Math.min(100, (liveUsed / u.traffic_limit) * 100) + "%" }} />
                          </span>
                        )}
                        {activeNodes > 0 && <span className="ml-2 text-zinc-400">· {activeNodes}/{nodeTraffic.length} servers</span>}
                      </p>
                    </div>
                  </div>

                  <div className="flex items-center gap-2 flex-shrink-0">
                    <span className="text-xs text-zinc-500">{formatDate(u.expires_at)}</span>
                    <span className={"px-2 py-0.5 rounded text-xs " + (isActive ? "bg-green-500/10 text-green-400" : "bg-red-500/10 text-red-400")}>
                      {!u.enabled ? "Disabled" : isExpired ? "Expired" : trafficExceeded ? "Over Limit" : "Active"}
                    </span>
                    {u.chain_proxy && <span className="px-2 py-0.5 rounded text-xs bg-purple-500/10 text-purple-400">AI</span>}

                    {/* Per-server traffic toggle */}
                    <button onClick={() => setTrafficExpanded(isTraffic ? null : u.ID)} title="Per-server traffic" className={"p-1.5 rounded hover:bg-zinc-800 " + (isTraffic ? "text-blue-400" : "text-zinc-500 hover:text-white")}>
                      <Server size={14} />
                    </button>

                    {/* Subscription links toggle */}
                    <button onClick={() => setExpanded(isExp ? null : u.ID)} title="Subscription Links" className={"p-1.5 rounded hover:bg-zinc-800 " + (isExp ? "text-blue-400" : "text-zinc-500 hover:text-white")}>
                      {isExp ? <ChevronUp size={14} /> : <Link size={14} />}
                    </button>

                    {/* Edit (pencil) */}
                    <button onClick={() => setEditingUser(u)} title="Edit user"
                      className="p-1.5 rounded hover:bg-zinc-800 text-zinc-500 hover:text-white">
                      <Edit2 size={14} />
                    </button>

                    {/* More actions dropdown */}
                    <div className="relative">
                      <button
                        onClick={(e) => { e.stopPropagation(); setMoreOpen(moreOpen === u.ID ? null : u.ID) }}
                        className="p-1.5 rounded hover:bg-zinc-700 text-zinc-300"
                        aria-label="More actions"
                        title="More actions"
                      >
                        ⚙️
                      </button>
                      {moreOpen === u.ID && (
                        <div className="more-dropdown absolute right-0 mt-1 z-30 bg-zinc-800 border border-zinc-700 rounded shadow-lg w-44 py-1 text-xs">
                          <button onClick={() => { toggle(u.ID); setMoreOpen(null) }} className="block w-full text-left px-3 py-1.5 hover:bg-zinc-700 text-white">
                            {u.enabled ? "Disable" : "Enable"}
                          </button>
                          <button onClick={() => { setPwForm({ id: u.ID, pw: "" }); setMoreOpen(null) }} className="block w-full text-left px-3 py-1.5 hover:bg-zinc-700 text-white">
                            Set login password
                          </button>
                          <button onClick={() => { resetTraffic(u.ID); setMoreOpen(null) }} className="block w-full text-left px-3 py-1.5 hover:bg-zinc-700 text-white">
                            Reset traffic
                          </button>
                          <button onClick={() => { api.post(`/admin/users/${u.ID}/toggle-chain`).then(load); setMoreOpen(null) }} className="block w-full text-left px-3 py-1.5 hover:bg-zinc-700 text-white">
                            {u.chain_proxy ? "关闭 Rule: AI" : "开启 Rule: AI"}
                          </button>
                          <div className="border-t border-zinc-700 my-1" />
                          <button onClick={() => { setPayDialog({ user: u, days: 30 }); setMoreOpen(null) }} className="block w-full text-left px-3 py-1.5 hover:bg-zinc-700 text-white">
                            Renew +30d
                          </button>
                          <button onClick={() => { setPayDialog({ user: u, days: 90 }); setMoreOpen(null) }} className="block w-full text-left px-3 py-1.5 hover:bg-zinc-700 text-white">
                            Renew +90d
                          </button>
                          <button onClick={() => { setPayDialog({ user: u, days: 365 }); setMoreOpen(null) }} className="block w-full text-left px-3 py-1.5 hover:bg-zinc-700 text-white">
                            Renew +365d
                          </button>
                          <div className="border-t border-zinc-700 my-1" />
                          <button onClick={() => { del(u.ID); setMoreOpen(null) }} className="block w-full text-left px-3 py-1.5 hover:bg-zinc-700 text-red-400">
                            Delete
                          </button>
                        </div>
                      )}
                    </div>
                  </div>
                </div>

                {/* Per-server traffic expand panel */}
                {isTraffic && (
                  <div className="mt-3 border-t border-zinc-800 pt-3">
                    <div className="flex items-center justify-between mb-3">
                      <p className="text-xs text-zinc-500">Per-Server Traffic for <span className="text-white">{u.username}</span></p>
                      <span className="text-xs text-zinc-600 font-mono">Total: {fmt(nodeTraffic.reduce((s, n) => s + n.total, 0))}</span>
                    </div>
                    {nodeTraffic.length === 0 ? (
                      <div className="text-xs text-zinc-600 text-center py-3">No servers available</div>
                    ) : (
                      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-2">
                        {nodeTraffic.map((n, i) => {
                          const maxTotal = Math.max(...nodeTraffic.map(x => x.total), 1)
                          return (
                            <div key={i} className="bg-black/40 rounded-lg p-3 border border-zinc-800 relative overflow-hidden">
                              <div className="flex items-center justify-between mb-1.5">
                                <div className="flex items-center gap-2 min-w-0">
                                  <span className={"w-1.5 h-1.5 rounded-full flex-shrink-0 " + (n.healthy ? "bg-green-500" : "bg-red-500")} />
                                  <span className="text-sm font-medium text-white truncate">{n.name}</span>
                                  {n.total === 0 && <span className="text-[10px] text-zinc-600">idle</span>}
                                </div>
                                <span className="text-xs font-semibold text-white font-mono">{fmt(n.total)}</span>
                              </div>
                              <div className="flex items-center gap-4 text-xs text-zinc-500">
                                <span className="flex items-center gap-1">
                                  <ArrowUp size={10} className="text-blue-400" />
                                  <span className="font-mono">{fmt(n.tx)}</span>
                                </span>
                                <span className="flex items-center gap-1">
                                  <ArrowDown size={10} className="text-green-400" />
                                  <span className="font-mono">{fmt(n.rx)}</span>
                                </span>
                                <span className="ml-auto text-[10px] text-zinc-600 font-mono">{n.host}:{n.port}</span>
                              </div>
                              {n.total > 0 && (
                                <div className="mt-2 w-full h-1 bg-zinc-800 rounded-full overflow-hidden">
                                  <div className="h-full bg-gradient-to-r from-blue-500 to-green-500" style={{ width: (n.total / maxTotal * 100) + "%" }} />
                                </div>
                              )}
                            </div>
                          )
                        })}
                      </div>
                    )}
                  </div>
                )}

                {/* Subscription links expand panel */}
                {isExp && (
                  <div className="mt-3 border-t border-zinc-800 pt-3">
                    <p className="text-xs text-zinc-500 mb-3">Subscription Links for <span className="text-white">{u.username}</span></p>
                    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-2">
                      {[
                        { label: "Universal (URI)", url: subUrl },
                        { label: "Clash / mihomo", url: subUrl + "?format=clash" },
                        { label: "Surge", url: subUrl + "?format=surge" },
                        { label: "Shadowrocket full config", url: subUrl + "?format=shadowrocket-conf" },
                        { label: "V2Ray / V2RayN / NekoBox", url: subUrl + "?format=v2ray" },
                      ].map(({ label, url }) => (
                        <div key={label} className="flex items-center justify-between bg-black rounded-lg p-2.5 border border-zinc-800">
                          <div className="min-w-0 flex-1 mr-3">
                            <p className="text-xs font-medium text-zinc-300">{label}</p>
                            <p className="text-xs text-zinc-600 truncate">{url}</p>
                          </div>
                          <button onClick={() => doCopy(url, u.ID + label)} className="flex-shrink-0 flex items-center gap-1 px-2.5 py-1.5 bg-zinc-800 rounded text-xs text-zinc-300 hover:bg-zinc-700">
                            {copied === u.ID + label ? <><Check size={11} /> Copied</> : <><Copy size={11} /> Copy</>}
                          </button>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            </div>
          )
        })}
      </div>

      {editingUser && (
        <UserEditModal
          user={editingUser}
          onClose={() => setEditingUser(null)}
          onSaved={load}
        />
      )}

      {payDialog && (() => {
        const planPrice = (allPlans.find((p: any) => p.ID === payDialog.user.plan_id) as any)?.price_cents || 0
        return (
          <PaymentDialog
            title={`续期 ${payDialog.user.username} · +${payDialog.days} 天`}
            defaultDays={payDialog.days}
            defaultAmountCents={planPrice}
            defaultKind="renew"
            showDays={true}
            onCancel={() => setPayDialog(null)}
            onConfirm={async (days, payment) => {
              await api.post(`/admin/users/${payDialog.user.ID}/renew`, { days, payment })
              setPayDialog(null)
              load()
            }}
          />
        )
      })()}
    </div>
  )
}
