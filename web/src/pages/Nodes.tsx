import { useEffect, useRef, useState } from "react"
import api from "@/api/client"
import { Plus, Trash2, Circle, Edit2, X, Activity, CheckCircle2, AlertTriangle, WifiOff, Clock3, ClipboardPaste, Wifi } from "lucide-react"
import { parseDeployOutput, type NodeImportForm } from "@/utils/nodeImport"
import { parseRegister } from "@/utils/nodeRegister"
import { fmtSpeed } from "@/lib/format"

// Live per-node connection data merged from /admin/stats.
type LiveStat = { online: number; tx: number; rx: number }
type LiveSpeed = { up: number; down: number }

const emptyForm = {
  name: "", host: "", port: 443, password: "", sni: "bing.com",
  insecure: true, obfs_type: "", obfs_password: "",
  traffic_api: "", traffic_secret: "", sort_order: 0,
  vless_enabled: false, vless_port: 443, reality_pubkey: "", reality_shortid: "",
  reality_sni: "www.apple.com", vless_stats_api: "", vless_stats_secret: "",
  trojan_enabled: false, trojan_port: 8443, trojan_sni: "www.apple.com",
}

export default function Nodes() {
  const [nodes, setNodes] = useState<any[]>([])
  const [showAdd, setShowAdd] = useState(false)
  const [form, setForm] = useState({ ...emptyForm })
  const [editId, setEditId] = useState<number | null>(null)
  const [msg, setMsg] = useState("")
  const [err, setErr] = useState("")
  const [regText, setRegText] = useState("")
  const [showReg, setShowReg] = useState(false)
  const [showImport, setShowImport] = useState(false)
  const [importText, setImportText] = useState("")
  const [importPreview, setImportPreview] = useState<NodeImportForm[]>([])
  const [importing, setImporting] = useState(false)
  const [live, setLive] = useState<Record<number, LiveStat>>({})
  const [speed, setSpeed] = useState<Record<number, LiveSpeed>>({})
  const prevRef = useRef<{ t: number; byId: Record<number, LiveStat> } | null>(null)

  const load = async () => {
    const [nres, sres] = await Promise.all([api.get("/admin/nodes"), api.get("/admin/stats")])
    setNodes(nres.data)
    const now = Date.now()
    const byId: Record<number, LiveStat> = {}
    for (const s of (sres.data?.nodes || [])) {
      let tx = 0, rx = 0
      if (s.traffic) for (const t of Object.values(s.traffic) as { tx: number; rx: number }[]) { tx += t.tx; rx += t.rx }
      byId[s.id] = { online: s.online || 0, tx, rx }
    }
    setLive(byId)
    const prev = prevRef.current
    if (prev) {
      const dt = (now - prev.t) / 1000
      const sp: Record<number, LiveSpeed> = {}
      if (dt > 0) for (const id of Object.keys(byId)) {
        const p = prev.byId[+id]
        if (p) sp[+id] = { up: Math.max(0, byId[+id].tx - p.tx) / dt, down: Math.max(0, byId[+id].rx - p.rx) / dt }
      }
      setSpeed(sp)
    }
    prevRef.current = { t: now, byId }
  }
  useEffect(() => {
    load()
    const t = setInterval(load, 3000)
    return () => clearInterval(t)
  }, [])
  const showMsg = (t: string) => { setMsg(t); setErr(""); setTimeout(() => setMsg(""), 2500) }
  const showErr = (t: string) => { setErr(t); setMsg("") }

  const add = async () => {
    if (!form.name || !form.host) return
    await api.post("/admin/nodes", form)
    setShowAdd(false)
    setForm({ ...emptyForm })
    showMsg("Node created")
    load()
  }

  const setImportValue = (value: string) => {
    setImportText(value)
    try {
      setImportPreview(parseDeployOutput(value))
    } catch {
      setImportPreview([])
    }
  }

  const createImportedNodes = async (parsed: NodeImportForm[]) => {
    const duplicated = parsed.filter((p) => nodes.some((n) => n.name === p.name)).map((p) => p.name)
    if (duplicated.length > 0) {
      throw new Error(`Node already exists: ${duplicated.join(", ")}`)
    }
    for (const node of parsed) {
      await api.post("/admin/nodes", node)
    }
  }

  const importNodesFromText = async (text: string) => {
    const parsed = parseDeployOutput(text)
    setImporting(true)
    try {
      await createImportedNodes(parsed)
      setShowImport(false)
      setImportText("")
      setImportPreview([])
      showMsg(`Imported ${parsed.length} nodes`)
      load()
    } finally {
      setImporting(false)
    }
  }

  const pasteAndImport = async () => {
    setEditId(null)
    setShowAdd(false)
    if (!navigator.clipboard?.readText) {
      setShowImport(true)
      showErr("Clipboard access is unavailable. Paste the deploy output below.")
      return
    }
    try {
      const text = await navigator.clipboard.readText()
      if (!text.trim()) throw new Error("Clipboard is empty")
      setImportValue(text)
      await importNodesFromText(text)
    } catch (e) {
      setShowImport(true)
      const error = e as Error
      showErr(error.message || "Paste failed. Paste the deploy output below.")
    }
  }

  const save = async () => {
    if (!editId) return
    await api.put(`/admin/nodes/${editId}`, form)
    setEditId(null)
    setForm({ ...emptyForm })
    showMsg("Node updated")
    load()
  }

  const startEdit = (n: any) => {
    setEditId(n.ID)
    setShowAdd(false)
    setForm({
      name: n.name, host: n.host, port: n.port, password: n.password || "",
      sni: n.sni || "", insecure: n.insecure, obfs_type: n.obfs_type || "",
      obfs_password: n.obfs_password || "", traffic_api: n.traffic_api || "",
      traffic_secret: n.traffic_secret || "", sort_order: n.sort_order || 0,
      vless_enabled: !!n.vless_enabled, vless_port: n.vless_port || 443,
      reality_pubkey: n.reality_pubkey || "", reality_shortid: n.reality_shortid || "",
      reality_sni: n.reality_sni || "www.apple.com", vless_stats_api: n.vless_stats_api || "",
      vless_stats_secret: "",
      trojan_enabled: !!n.trojan_enabled, trojan_port: n.trojan_port || 8443,
      trojan_sni: n.trojan_sni || "www.apple.com",
    })
  }

  const applyRegister = () => {
    const f = parseRegister(regText)
    if (Object.keys(f).length === 0) { showErr("未识别到注册信息"); return }
    setForm(prev => ({ ...prev, ...f }))
    setShowReg(false); setRegText(""); showMsg("已填入注册信息")
  }

  const cancelEdit = () => { setEditId(null); setForm({ ...emptyForm }) }

  const del = async (id: number) => {
    if (!confirm("Delete this node?")) return
    await api.delete(`/admin/nodes/${id}`)
    showMsg("Node deleted")
    load()
  }

  const inputCls = "px-3 py-2 bg-surface border border-zinc-700 rounded-lg text-sm text-ink focus:outline-none focus:border-zinc-500"

  const renderImportPanel = () => (
    <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-4 mb-4">
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-sm font-medium text-zinc-400">Paste Deploy Output</h3>
        <button onClick={() => setShowImport(false)} className="text-zinc-500 hover:text-ink" title="Close"><X size={14} /></button>
      </div>
      <textarea
        value={importText}
        onChange={(e) => setImportValue(e.target.value)}
        rows={12}
        className={inputCls + " w-full font-mono text-xs mb-3"}
        placeholder="plain node:
  Name: JP4-plain
  Host: 120.231.184.128
  Port: 443
  ...

obfs node:
  Name: JP4-obfs
  ..."
      />
      {importPreview.length > 0 && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3 mb-3">
          {importPreview.map((n) => (
            <div key={n.name} className="bg-surface border border-zinc-800 rounded-lg p-3 text-xs">
              <div className="font-semibold text-ink mb-1">{n.name}</div>
              <div className="text-zinc-400 font-mono">{n.host}:{n.port}</div>
              <div className="text-zinc-500 mt-1">{n.obfs_type ? `obfs: ${n.obfs_type}` : "plain"}</div>
              <div className="text-zinc-500 truncate">{n.traffic_api}</div>
            </div>
          ))}
        </div>
      )}
      <div className="flex gap-2">
        <button
          disabled={importing || importPreview.length === 0}
          onClick={() => importNodesFromText(importText).catch((e) => showErr((e as Error).message))}
          className="px-4 py-2 bg-clay text-white rounded-lg text-sm font-medium hover:bg-clay-hover disabled:opacity-50"
        >
          {importing ? "Importing..." : `Import ${importPreview.length || ""} Nodes`}
        </button>
        <button onClick={() => { setShowImport(false); setImportText(""); setImportPreview([]) }} className="px-4 py-2 text-sm text-zinc-400 hover:text-ink">Cancel</button>
      </div>
    </div>
  )

  const probeStatus = (n: any) => n.probe?.status || (n.healthy ? "pending" : "failed")
  const statusMeta = (n: any) => {
    const s = probeStatus(n)
    if (s === "ok") return { label: "可达 OK", dot: "fill-green-500 text-green-500", pill: "bg-green-500/10 text-green-400 border-green-500/20" }
    if (s === "warning") return { label: "注意 WARN", dot: "fill-amber-500 text-amber-500", pill: "bg-amber-500/10 text-amber-300 border-amber-500/20" }
    if (s === "failed") return { label: "故障 DOWN", dot: "fill-red-500 text-red-500", pill: "bg-red-500/10 text-red-400 border-red-500/20" }
    return { label: "等待探测", dot: "fill-zinc-500 text-zinc-500", pill: "bg-zinc-800 text-zinc-400 border-zinc-700" }
  }
  const probeTime = (n: any) => n.probe?.last_checked_at ? new Date(n.probe.last_checked_at).toLocaleTimeString() : "未探测"
  const summary = {
    total: nodes.length,
    ok: nodes.filter(n => probeStatus(n) === "ok").length,
    warning: nodes.filter(n => probeStatus(n) === "warning").length,
    failed: nodes.filter(n => probeStatus(n) === "failed").length,
    pending: nodes.filter(n => probeStatus(n) === "pending").length,
  }
  const lastCheckedAt = nodes
    .map(n => n.probe?.last_checked_at)
    .filter(Boolean)
    .sort()
    .pop()
  const lastCheckedLabel = lastCheckedAt ? new Date(lastCheckedAt).toLocaleTimeString() : "未探测"

  const renderForm = (isEdit: boolean) => (
    <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-4 mb-4">
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-sm font-medium text-zinc-400">{isEdit ? "Edit Node" : "New Node"}</h3>
        <button onClick={isEdit ? cancelEdit : () => setShowAdd(false)} className="text-zinc-500 hover:text-ink"><X size={14} /></button>
      </div>

      <div className="grid grid-cols-3 gap-3 mb-3">
        <div>
          <label className="text-xs text-zinc-500 mb-1 block">Name *</label>
          <input value={form.name} onChange={e => setForm({...form, name: e.target.value})} className={inputCls + " w-full"} />
        </div>
        <div>
          <label className="text-xs text-zinc-500 mb-1 block">Host *</label>
          <input value={form.host} onChange={e => setForm({...form, host: e.target.value})} className={inputCls + " w-full"} />
        </div>
        <div>
          <label className="text-xs text-zinc-500 mb-1 block">Port</label>
          <input type="number" value={form.port} onChange={e => setForm({...form, port: +e.target.value})} className={inputCls + " w-full"} />
        </div>
      </div>

      <div className="grid grid-cols-3 gap-3 mb-3">
        <div>
          <label className="text-xs text-zinc-500 mb-1 block">SNI</label>
          <input value={form.sni} onChange={e => setForm({...form, sni: e.target.value})} className={inputCls + " w-full"} />
        </div>
        <div>
          <label className="text-xs text-zinc-500 mb-1 block">Password</label>
          <input value={form.password} onChange={e => setForm({...form, password: e.target.value})} className={inputCls + " w-full"} />
        </div>
        <div className="flex items-end gap-3">
          <label className="flex items-center gap-2 text-sm text-zinc-400 cursor-pointer pb-2">
            <input type="checkbox" checked={form.insecure} onChange={e => setForm({...form, insecure: e.target.checked})}
              className="w-4 h-4 rounded border-zinc-600 bg-surface" />
            Skip Cert Verify
          </label>
        </div>
      </div>

      <div className="grid grid-cols-3 gap-3 mb-3">
        <div>
          <label className="text-xs text-zinc-500 mb-1 block">Obfs Type</label>
          <select value={form.obfs_type} onChange={e => setForm({...form, obfs_type: e.target.value})}
            className={inputCls + " w-full"}>
            <option value="">None</option>
            <option value="salamander">Salamander</option>
          </select>
        </div>
        <div>
          <label className="text-xs text-zinc-500 mb-1 block">Obfs Password</label>
          <input value={form.obfs_password} onChange={e => setForm({...form, obfs_password: e.target.value})}
            className={inputCls + " w-full"} disabled={!form.obfs_type} />
        </div>
        <div>
          <label className="text-xs text-zinc-500 mb-1 block">Sort Order</label>
          <input type="number" value={form.sort_order} onChange={e => setForm({...form, sort_order: +e.target.value})} className={inputCls + " w-full"} />
        </div>
      </div>

      <div className="grid grid-cols-2 gap-3 mb-3">
        <div>
          <label className="text-xs text-zinc-500 mb-1 block">Traffic API URL</label>
          <input placeholder="http://127.0.0.1:25413" value={form.traffic_api} onChange={e => setForm({...form, traffic_api: e.target.value})} className={inputCls + " w-full"} />
        </div>
        <div>
          <label className="text-xs text-zinc-500 mb-1 block">Traffic API Secret</label>
          <input value={form.traffic_secret} onChange={e => setForm({...form, traffic_secret: e.target.value})} className={inputCls + " w-full"} />
        </div>
      </div>

      <div className="border-t border-zinc-800 mt-2 pt-3 mb-3">
        <div className="flex items-center justify-between mb-2">
          <h4 className="text-sm font-medium text-zinc-300">TCP 回落 (VLESS / Trojan)</h4>
          <button type="button" onClick={() => setShowReg(s => !s)}
            className="text-xs px-2 py-1 rounded bg-zinc-800 text-zinc-300 hover:bg-zinc-700">📋 粘贴注册信息</button>
        </div>
        {showReg && (
          <div className="mb-3">
            <textarea value={regText} onChange={e => setRegText(e.target.value)} rows={6}
              placeholder="粘贴 vless-pilot-setup / trojan-add 打印的 REGISTER 段..."
              className={inputCls + " w-full font-mono text-xs"} />
            <button type="button" onClick={applyRegister}
              className="mt-2 px-3 py-1.5 bg-clay text-white rounded-lg text-sm font-medium hover:bg-clay-hover">填入</button>
          </div>
        )}
        <label className="flex items-center gap-2 text-sm text-zinc-300 mb-2">
          <input type="checkbox" checked={form.vless_enabled}
            onChange={e => setForm({...form, vless_enabled: e.target.checked})} className="w-4 h-4 rounded border-zinc-600 bg-surface" />
          启用 VLESS-Reality
        </label>
        {form.vless_enabled && (
          <div className="grid grid-cols-3 gap-3 mb-3 pl-6">
            <div><label className="text-xs text-zinc-500 mb-1 block">VLESS Port</label>
              <input type="number" value={form.vless_port} onChange={e => setForm({...form, vless_port: +e.target.value})} className={inputCls + " w-full"} /></div>
            <div><label className="text-xs text-zinc-500 mb-1 block">Reality SNI</label>
              <input value={form.reality_sni} onChange={e => setForm({...form, reality_sni: e.target.value})} className={inputCls + " w-full"} /></div>
            <div><label className="text-xs text-zinc-500 mb-1 block">Short ID</label>
              <input value={form.reality_shortid} onChange={e => setForm({...form, reality_shortid: e.target.value})} className={inputCls + " w-full"} /></div>
            <div className="col-span-3"><label className="text-xs text-zinc-500 mb-1 block">Reality Public Key</label>
              <input value={form.reality_pubkey} onChange={e => setForm({...form, reality_pubkey: e.target.value})} className={inputCls + " w-full font-mono text-xs"} /></div>
            <div className="col-span-2"><label className="text-xs text-zinc-500 mb-1 block">Stats API</label>
              <input value={form.vless_stats_api} onChange={e => setForm({...form, vless_stats_api: e.target.value})} className={inputCls + " w-full"} /></div>
            <div><label className="text-xs text-zinc-500 mb-1 block">Stats 密钥 (留空不变)</label>
              <input value={form.vless_stats_secret} placeholder="已设置 / 留空不变" onChange={e => setForm({...form, vless_stats_secret: e.target.value})} className={inputCls + " w-full"} /></div>
          </div>
        )}
        <label className="flex items-center gap-2 text-sm text-zinc-300 mb-2">
          <input type="checkbox" checked={form.trojan_enabled}
            onChange={e => setForm({...form, trojan_enabled: e.target.checked})} className="w-4 h-4 rounded border-zinc-600 bg-surface" />
          启用 Trojan
        </label>
        {form.trojan_enabled && (
          <div className="grid grid-cols-2 gap-3 pl-6">
            <div><label className="text-xs text-zinc-500 mb-1 block">Trojan Port</label>
              <input type="number" value={form.trojan_port} onChange={e => setForm({...form, trojan_port: +e.target.value})} className={inputCls + " w-full"} /></div>
            <div><label className="text-xs text-zinc-500 mb-1 block">Trojan SNI</label>
              <input value={form.trojan_sni} onChange={e => setForm({...form, trojan_sni: e.target.value})} className={inputCls + " w-full"} /></div>
          </div>
        )}
      </div>

      <div className="flex gap-2">
        <button onClick={isEdit ? save : add} className="px-4 py-2 bg-clay text-white rounded-lg text-sm font-medium hover:bg-clay-hover">
          {isEdit ? "Save Changes" : "Create Node"}
        </button>
        <button onClick={isEdit ? cancelEdit : () => setShowAdd(false)} className="px-4 py-2 text-sm text-zinc-400 hover:text-ink">Cancel</button>
      </div>
    </div>
  )

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-xl font-semibold">Nodes</h2>
        <div className="flex gap-2">
          <button onClick={pasteAndImport}
            className="flex items-center gap-2 px-3 py-1.5 bg-zinc-800 text-zinc-200 rounded-lg text-sm font-medium hover:bg-zinc-700">
            <ClipboardPaste size={14} /> Paste Deploy Output
          </button>
          <button onClick={() => { setShowAdd(!showAdd); setEditId(null); setShowImport(false); setForm({...emptyForm}) }}
            className="flex items-center gap-2 px-3 py-1.5 bg-clay text-white rounded-lg text-sm font-medium hover:bg-clay-hover">
            <Plus size={14} /> Add Node
          </button>
        </div>
      </div>

      {msg && <div className="mb-4 px-4 py-2 bg-green-500/10 border border-green-500/20 rounded-lg text-green-400 text-sm">{msg}</div>}
      {err && <div className="mb-4 px-4 py-2 bg-red-500/10 border border-red-500/20 rounded-lg text-red-300 text-sm">{err}</div>}

      {showImport && renderImportPanel()}
      {showAdd && !editId && renderForm(false)}
      {editId && renderForm(true)}

      <div className="grid grid-cols-2 lg:grid-cols-6 gap-3 mb-4">
        <div className="bg-zinc-900 border border-zinc-800 rounded-lg p-3">
          <div className="flex items-center justify-between text-zinc-500 text-xs mb-2">
            <span>监控节点</span><Activity size={14} />
          </div>
          <div className="text-2xl font-semibold text-ink">{summary.total}</div>
        </div>
        <div className="bg-zinc-900 border border-blue-500/20 rounded-lg p-3">
          <div className="flex items-center justify-between text-blue-300 text-xs mb-2">
            <span>实时在线</span><Wifi size={14} />
          </div>
          <div className="text-2xl font-semibold text-blue-300">{Object.values(live).reduce((s, v) => s + v.online, 0)}</div>
        </div>
        <div className="bg-zinc-900 border border-green-500/20 rounded-lg p-3">
          <div className="flex items-center justify-between text-green-400 text-xs mb-2">
            <span>可达</span><CheckCircle2 size={14} />
          </div>
          <div className="text-2xl font-semibold text-green-400">{summary.ok}</div>
        </div>
        <div className="bg-zinc-900 border border-amber-500/20 rounded-lg p-3">
          <div className="flex items-center justify-between text-amber-300 text-xs mb-2">
            <span>注意</span><AlertTriangle size={14} />
          </div>
          <div className="text-2xl font-semibold text-amber-300">{summary.warning}</div>
        </div>
        <div className="bg-zinc-900 border border-red-500/20 rounded-lg p-3">
          <div className="flex items-center justify-between text-red-400 text-xs mb-2">
            <span>故障</span><WifiOff size={14} />
          </div>
          <div className="text-2xl font-semibold text-red-400">{summary.failed}</div>
        </div>
        <div className="bg-zinc-900 border border-zinc-800 rounded-lg p-3">
          <div className="flex items-center justify-between text-zinc-500 text-xs mb-2">
            <span>最近探测</span><Clock3 size={14} />
          </div>
          <div className="text-lg font-semibold text-zinc-200">{lastCheckedLabel}</div>
          {summary.pending > 0 && <div className="text-xs text-zinc-500 mt-1">{summary.pending} 个等待探测</div>}
        </div>
      </div>

      <div className="bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden">
        <table className="w-full text-sm">
          <thead><tr className="border-b border-zinc-800 text-zinc-500">
            <th className="text-left p-4 w-[190px]">Monitor</th>
            <th className="text-left p-4">Name</th>
            <th className="text-left p-4">Host:Port</th>
            <th className="text-left p-4">在线</th>
            <th className="text-left p-4">实时 ↑/↓</th>
            <th className="text-left p-4">SNI</th>
            <th className="text-left p-4">Obfs</th>
            <th className="text-left p-4">Traffic API</th>
            <th className="text-right p-4">Actions</th>
          </tr></thead>
          <tbody>
            {nodes.map(n => (
              <tr key={n.ID} className={"border-b border-zinc-800/50 hover:bg-zinc-800/30" + (editId === n.ID ? " bg-zinc-800/20" : "")}>
                <td className="p-4">
                  <div className="space-y-1">
                    <span className={`inline-flex items-center gap-2 px-2.5 py-1 rounded border text-[11px] font-medium ${statusMeta(n).pill}`}>
                      <Circle size={8} className={statusMeta(n).dot} />
                      {statusMeta(n).label}
                    </span>
                    <div className="text-[11px] text-zinc-500 font-mono">
                      {probeTime(n)}
                      {n.probe?.last_latency_ms > 0 && <span> · {n.probe.last_latency_ms}ms</span>}
                    </div>
                    {n.probe?.fail_streak > 0 && <div className="text-[11px] text-red-400">连续失败 {n.probe.fail_streak} 次</div>}
                  </div>
                </td>
                <td className="p-4">
                  <p className="font-medium">{n.name}</p>
                  {n.insecure && <span className="text-xs text-zinc-500">insecure</span>}
                </td>
                <td className="p-4 text-zinc-400">{n.host}:{n.port}</td>
                <td className="p-4">
                  {live[n.ID] ? (
                    live[n.ID].online > 0
                      ? <span className="text-green-400 font-medium">{live[n.ID].online}</span>
                      : <span className="text-zinc-600">0</span>
                  ) : <span className="text-zinc-600">-</span>}
                </td>
                <td className="p-4 text-xs font-mono whitespace-nowrap">
                  {speed[n.ID] && (speed[n.ID].up > 0 || speed[n.ID].down > 0) ? (
                    <>
                      <span className="text-sky-400">↑{fmtSpeed(speed[n.ID].up)}</span>
                      <span className="text-emerald-400 ml-2">↓{fmtSpeed(speed[n.ID].down)}</span>
                    </>
                  ) : <span className="text-zinc-600">-</span>}
                </td>
                <td className="p-4 text-zinc-400">{n.sni || "-"}</td>
                <td className="p-4">
                  {n.obfs_type ? (
                    <span className="px-2 py-0.5 rounded text-xs bg-purple-500/10 text-purple-400">{n.obfs_type}</span>
                  ) : (
                    <span className="text-zinc-600">-</span>
                  )}
                </td>
                <td className="p-4 text-zinc-400 text-xs">
                  {n.traffic_api ? (
                    <div className="max-w-[260px]">
                      <span className="px-2 py-0.5 rounded bg-blue-500/10 text-blue-400">Configured</span>
                      {n.probe?.last_error && (
                        <p className="mt-1 truncate text-red-300" title={n.probe.last_error}>{n.probe.last_error}</p>
                      )}
                    </div>
                  ) : (
                    <span className="text-zinc-600">None</span>
                  )}
                </td>
                <td className="p-4">
                  <div className="flex items-center justify-end gap-1">
                    <button onClick={() => startEdit(n)} title="Edit" className="p-1.5 rounded hover:bg-zinc-800 text-zinc-500 hover:text-ink">
                      <Edit2 size={14} />
                    </button>
                    <button onClick={() => del(n.ID)} title="Delete" className="p-1.5 rounded hover:bg-zinc-800 text-zinc-500 hover:text-red-400">
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
