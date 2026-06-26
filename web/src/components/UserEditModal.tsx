import { useState, useEffect } from "react"
import api from "@/api/client"
import { X, RefreshCw, Copy, Check, QrCode } from "lucide-react"
import { copyToClipboard } from "@/lib/clipboard"
import PaymentDialog from "@/components/PaymentDialog"

type Tab = "basic" | "subscription" | "access" | "rules"

export interface EditableUser {
  ID: number
  username: string
  email?: string
  notes?: string
  tags?: string
  hy2_password: string
  sub_token: string
  node_ids: string

  // Added for Access + Rules tabs
  traffic_limit: number
  traffic_used: number
  expires_at: string
  enabled: boolean
  auto_reset?: boolean
  last_reset_at?: string
  rule_ai?: boolean
  rule_streaming?: boolean
  rule_china?: boolean
  rule_ad_block?: boolean

  // Telegram
  telegram_id?: number
}

export default function UserEditModal({ user, onClose, onSaved }: {
  user: EditableUser
  onClose: () => void
  onSaved: () => void
}) {
  const [tab, setTab] = useState<Tab>("basic")
  const [form, setForm] = useState({
    email: user.email || "",
    notes: user.notes || "",
    tags: user.tags || "",
    login_password: "",
  })
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState("")
  const [nodes, setNodes] = useState<{ ID: number; name: string; healthy: boolean }[]>([])
  const [plans, setPlans] = useState<{ ID: number; name: string; price_cents?: number; duration_days?: number }[]>([])
  const [selectedPlan, setSelectedPlan] = useState<number | "">("")
  const [applyPayDialog, setApplyPayDialog] = useState<{ planId: number; planName: string; price: number; days: number } | null>(null)

  useEffect(() => {
    api.get("/admin/nodes").then((r) => setNodes(r.data))
  }, [])

  useEffect(() => {
    api.get("/admin/plans").then((r) => setPlans(r.data))
  }, [])

  const applyPlan = () => {
    if (!selectedPlan) return
    const plan = plans.find((p) => p.ID === selectedPlan)
    const planName = plan?.name || "plan"
    if (!confirm(`Apply "${planName}" — this overwrites traffic limit, expiry, nodes, and rule flags. Continue?`)) return
    setApplyPayDialog({
      planId: selectedPlan as number,
      planName,
      price: plan?.price_cents || 0,
      days: plan?.duration_days || 0,
    })
  }

  const save = async () => {
    setSaving(true)
    setError("")
    try {
      const body: Record<string, unknown> = {
        email: form.email,
        notes: form.notes,
        tags: form.tags,
      }
      await api.put(`/admin/users/${user.ID}`, body)
      if (form.login_password) {
        await api.post(`/admin/users/${user.ID}/set-password`, { password: form.login_password })
      }
      onSaved()
      onClose()
    } catch (e: unknown) {
      const msg = (e as { response?: { data?: { error?: string } } }).response?.data?.error || String(e)
      setError(msg)
    }
    setSaving(false)
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-ink/45" onClick={onClose}>
      <div
        className="bg-zinc-900 border border-zinc-800 rounded-xl w-[600px] max-w-[90vw] max-h-[85vh] overflow-hidden flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between p-4 border-b border-zinc-800">
          <h3 className="text-sm font-semibold text-ink">Edit User: {user.username}</h3>
          <button onClick={onClose} className="text-zinc-500 hover:text-ink">
            <X size={16} />
          </button>
        </div>

        <div className="px-4 py-3 border-b border-zinc-800 flex items-center gap-3 flex-wrap bg-zinc-900/50">
          <span className="text-xs text-zinc-400 flex items-center gap-1.5">
            📋 Quick Apply Plan:
          </span>
          <select
            value={selectedPlan}
            onChange={(e) => setSelectedPlan(e.target.value === "" ? "" : +e.target.value)}
            className="px-3 py-1.5 bg-surface border border-zinc-700 rounded text-sm text-ink"
          >
            <option value="">Select plan</option>
            {plans.map((p) => (
              <option key={p.ID} value={p.ID}>
                {p.name}
              </option>
            ))}
          </select>
          <button
            onClick={applyPlan}
            disabled={!selectedPlan}
            className="px-3 py-1.5 text-xs bg-clay text-white rounded font-medium hover:bg-clay-hover disabled:opacity-40 disabled:cursor-not-allowed"
          >
            Apply
          </button>
        </div>

        <div className="flex border-b border-zinc-800 px-4">
          {(["basic", "subscription", "access", "rules"] as const).map((t) => (
            <button
              key={t}
              onClick={() => setTab(t)}
              className={
                "px-4 py-2 text-xs uppercase tracking-wider " +
                (tab === t
                  ? "text-ink border-b-2 border-blue-400"
                  : "text-zinc-500 hover:text-zinc-300")
              }
            >
              {t}
            </button>
          ))}
        </div>

        <div className="flex-1 overflow-y-auto p-5">
          {tab === "basic" && <BasicTab form={form} setForm={setForm} username={user.username} user={user} onSaved={onSaved} />}
          {tab === "subscription" && <SubscriptionTab user={user} onSaved={onSaved} />}
          {tab === "access" && <AccessTab user={user} nodes={nodes} onSaved={onSaved} />}
          {tab === "rules"  && <RulesTab  user={user}              onSaved={onSaved} />}
        </div>

        {error && (
          <div className="p-3 text-red-400 text-xs border-t border-zinc-800">{error}</div>
        )}

        {tab === "basic" && (
          <div className="flex justify-end gap-2 p-4 border-t border-zinc-800">
            <button onClick={onClose} className="px-4 py-2 text-sm text-zinc-400 hover:text-ink">
              Cancel
            </button>
            <button
              onClick={save}
              disabled={saving}
              className="px-4 py-2 text-sm bg-clay text-white rounded font-medium hover:bg-clay-hover disabled:opacity-50"
            >
              {saving ? "Saving..." : "Save"}
            </button>
          </div>
        )}
      </div>

      {applyPayDialog && (
        <PaymentDialog
          title={`应用 Plan ${applyPayDialog.planName} → ${user.username}`}
          defaultDays={applyPayDialog.days}
          defaultAmountCents={applyPayDialog.price}
          defaultKind="renew"
          showDays={false}
          onCancel={() => setApplyPayDialog(null)}
          onConfirm={async (_days, payment) => {
            try {
              await api.post(`/admin/plans/${applyPayDialog.planId}/apply-to/${user.ID}`, { payment })
              setApplyPayDialog(null)
              onSaved()
              onClose()
            } catch (e: unknown) {
              alert((e as { response?: { data?: { error?: string } } }).response?.data?.error || String(e))
              throw e
            }
          }}
        />
      )}
    </div>
  )
}

/* Basic tab — meta fields + login password */
function BasicTab({
  form,
  setForm,
  username,
  user,
  onSaved,
}: {
  form: { email: string; notes: string; tags: string; login_password: string }
  setForm: (f: { email: string; notes: string; tags: string; login_password: string }) => void
  username: string
  user: EditableUser
  onSaved: () => void
}) {
  const inputCls =
    "w-full px-3 py-2 bg-surface border border-zinc-700 rounded-lg text-sm text-ink focus:outline-none focus:border-zinc-500"
  return (
    <div className="space-y-4">
      <div>
        <label className="text-xs text-zinc-500 uppercase tracking-wider">Username</label>
        <p className="text-sm text-ink font-mono mt-1">{username}</p>
      </div>
      <div>
        <label className="text-xs text-zinc-500 uppercase tracking-wider">Email</label>
        <input
          type="email"
          value={form.email}
          onChange={(e) => setForm({ ...form, email: e.target.value })}
          placeholder="notes only — no notifications sent"
          className={inputCls + " mt-1"}
        />
      </div>
      <div>
        <label className="text-xs text-zinc-500 uppercase tracking-wider">Notes</label>
        <textarea
          value={form.notes}
          onChange={(e) => setForm({ ...form, notes: e.target.value })}
          rows={3}
          className={inputCls + " mt-1 resize-none"}
        />
      </div>
      <div>
        <label className="text-xs text-zinc-500 uppercase tracking-wider">Tags (comma-separated)</label>
        <input
          value={form.tags}
          onChange={(e) => setForm({ ...form, tags: e.target.value })}
          placeholder="paid, trial, beta"
          className={inputCls + " mt-1"}
        />
      </div>
      <div>
        <label className="text-xs text-zinc-500 uppercase tracking-wider">Telegram Link</label>
        <TelegramLinkRow user={user} onSaved={onSaved} />
      </div>
      <div>
        <label className="text-xs text-zinc-500 uppercase tracking-wider">New Login Password</label>
        <input
          type="password"
          value={form.login_password}
          onChange={(e) => setForm({ ...form, login_password: e.target.value })}
          placeholder="leave blank to keep existing"
          className={inputCls + " mt-1"}
        />
      </div>
    </div>
  )
}

function TelegramLinkRow({ user, onSaved }: { user: EditableUser; onSaved: () => void }) {
  const [unlinking, setUnlinking] = useState(false)
  const tgID = user.telegram_id
  if (!tgID) {
    return (
      <p className="text-xs text-zinc-500 mt-1">
        Not linked. User can run <code className="font-mono text-zinc-300">/link</code> in
        the bot to link their Telegram account.
      </p>
    )
  }
  const unlink = async () => {
    if (!confirm("Unlink Telegram from this user?")) return
    setUnlinking(true)
    try {
      await api.put(`/admin/users/${user.ID}`, { telegram_id: 0 })
      onSaved()
    } catch (e: unknown) {
      alert(String(e))
    }
    setUnlinking(false)
  }
  return (
    <div className="flex items-center gap-2 mt-1">
      <span className="text-sm text-sky-400 font-mono">TG #{tgID}</span>
      <button
        onClick={unlink}
        disabled={unlinking}
        className="text-xs text-zinc-400 hover:text-red-400 disabled:opacity-50"
      >
        {unlinking ? "..." : "Unlink"}
      </button>
    </div>
  )
}

function genHexPassword(username: string): string {
  const bytes = new Uint8Array(16)
  crypto.getRandomValues(bytes)
  const hex = Array.from(bytes).map(b => b.toString(16).padStart(2, "0")).join("")
  return `${username}:${hex}`
}

function genSubToken(): string {
  const bytes = new Uint8Array(16)
  crypto.getRandomValues(bytes)
  return Array.from(bytes).map(b => b.toString(16).padStart(2, "0")).join("")
}

function SubscriptionTab({ user, onSaved }: { user: EditableUser; onSaved: () => void }) {
  const [hy2, setHy2] = useState(user.hy2_password)
  const [tok, setTok] = useState(user.sub_token)
  const [saving, setSaving] = useState(false)
  const [copied, setCopied] = useState("")
  const [error, setError] = useState("")
  const [qrFormat, setQrFormat] = useState<string | null>(null)

  const origin = window.location.origin
  const baseUrl = `${origin}/api/sub/${tok}`
  const formats = [
    { label: "URI",          url: baseUrl },
    { label: "Clash",        url: `${baseUrl}?format=clash` },
    { label: "Surge",        url: `${baseUrl}?format=surge` },
    { label: "Shadowrocket full config", url: `${baseUrl}?format=shadowrocket-conf` },
    { label: "V2Ray",        url: `${baseUrl}?format=v2ray` },
  ]

  const save = async (field: "hy2_password" | "sub_token", value: string) => {
    setSaving(true)
    setError("")
    try {
      await api.put(`/admin/users/${user.ID}`, { [field]: value })
      onSaved()
    } catch (e: unknown) {
      setError((e as { response?: { data?: { error?: string } } }).response?.data?.error || String(e))
    }
    setSaving(false)
  }

  const doCopy = (url: string, label: string) => {
    copyToClipboard(url).then(() => {
      setCopied(label)
      setTimeout(() => setCopied(""), 2000)
    })
  }

  const inputCls =
    "flex-1 px-3 py-2 bg-surface border border-zinc-700 rounded-lg text-xs text-ink font-mono focus:outline-none focus:border-zinc-500"

  return (
    <div className="space-y-4">
      <div>
        <label className="text-xs text-zinc-500 uppercase tracking-wider">HY2 Password</label>
        <div className="flex gap-2 mt-1">
          <input value={hy2} onChange={(e) => setHy2(e.target.value)} className={inputCls} />
          <button
            onClick={() => setHy2(genHexPassword(user.username))}
            title="Regenerate"
            className="px-3 py-2 bg-zinc-800 rounded hover:bg-zinc-700"
          >
            <RefreshCw size={14} className="text-zinc-400" />
          </button>
          <button
            onClick={() => save("hy2_password", hy2)}
            disabled={saving}
            className="px-3 py-2 text-xs bg-clay text-white rounded font-medium hover:bg-clay-hover disabled:opacity-50"
          >
            Apply
          </button>
        </div>
        <p className="text-xs text-zinc-600 mt-1">Min 8 chars. Clients must reconnect after change.</p>
      </div>

      <div>
        <label className="text-xs text-zinc-500 uppercase tracking-wider">Subscription Token</label>
        <div className="flex gap-2 mt-1">
          <input value={tok} onChange={(e) => setTok(e.target.value)} className={inputCls} />
          <button
            onClick={() => setTok(genSubToken())}
            title="Regenerate"
            className="px-3 py-2 bg-zinc-800 rounded hover:bg-zinc-700"
          >
            <RefreshCw size={14} className="text-zinc-400" />
          </button>
          <button
            onClick={() => save("sub_token", tok)}
            disabled={saving}
            className="px-3 py-2 text-xs bg-clay text-white rounded font-medium hover:bg-clay-hover disabled:opacity-50"
          >
            Apply
          </button>
        </div>
        <p className="text-xs text-zinc-600 mt-1">
          Format: <code>^[a-zA-Z0-9_-]{"{8,64}"}$</code>. Must be globally unique.
        </p>
      </div>

      {error && <p className="text-xs text-red-400">{error}</p>}

      <div>
        <label className="text-xs text-zinc-500 uppercase tracking-wider">Subscription URLs</label>
        <div className="grid grid-cols-2 gap-2 mt-2">
          {formats.map((f) => (
            <div key={f.label} className="bg-surface border border-zinc-800 rounded-lg p-3">
              <p className="text-xs text-zinc-400 font-semibold mb-2">{f.label}</p>
              <p className="text-[10px] text-zinc-600 font-mono truncate mb-2">{f.url}</p>
              <div className="flex items-center">
                <button
                  onClick={() => doCopy(f.url, f.label)}
                  className="flex items-center gap-1 text-xs text-zinc-300 hover:text-ink"
                >
                  {copied === f.label ? (
                    <>
                      <Check size={11} /> Copied
                    </>
                  ) : (
                    <>
                      <Copy size={11} /> Copy
                    </>
                  )}
                </button>
                <button
                  onClick={() => setQrFormat(f.label.toLowerCase())}
                  className="flex items-center gap-1 text-xs text-zinc-300 hover:text-ink ml-2"
                  title="Show QR code"
                >
                  <QrCode size={11} /> QR
                </button>
              </div>
            </div>
          ))}
        </div>
      </div>
      {qrFormat && <QRModal userId={user.ID} format={qrFormat === "uri" ? "uri" : qrFormat} onClose={() => setQrFormat(null)} />}
    </div>
  )
}

function AccessTab({
  user,
  nodes,
  onSaved,
}: {
  user: EditableUser
  nodes: { ID: number; name: string; healthy: boolean }[]
  onSaved: () => void
}) {
  const [form, setForm] = useState({
    traffic_limit_gb: user.traffic_limit / 1024 / 1024 / 1024,
    expires_at: (user.expires_at || "").split("T")[0],
    enabled: user.enabled,
    auto_reset: user.auto_reset || false,
    restrict: user.node_ids !== "all" && user.node_ids !== "",
    allowedNodeIds:
      user.node_ids === "all" || user.node_ids === ""
        ? ([] as number[])
        : user.node_ids.split(",").map(Number),
  })
  const [err, setErr] = useState("")

  const addDays = (d: number) => {
    const base = form.expires_at ? new Date(form.expires_at) : new Date()
    base.setDate(base.getDate() + d)
    setForm({ ...form, expires_at: base.toISOString().split("T")[0] })
  }

  const save = async () => {
    try {
      setErr("")
      const body: Record<string, unknown> = {
        traffic_limit: Math.round(form.traffic_limit_gb * 1024 * 1024 * 1024),
        enabled: form.enabled,
        auto_reset: form.auto_reset,
        expires_at: form.expires_at,
        node_ids:
          form.restrict && form.allowedNodeIds.length
            ? form.allowedNodeIds.sort((a, b) => a - b).join(",")
            : "all",
      }
      await api.put(`/admin/users/${user.ID}`, body)
      onSaved()
    } catch (e: unknown) {
      setErr(
        (e as { response?: { data?: { error?: string } } }).response?.data?.error ||
          String(e)
      )
    }
  }

  const inputCls =
    "w-full px-3 py-2 bg-surface border border-zinc-700 rounded-lg text-sm text-ink focus:outline-none focus:border-zinc-500"

  return (
    <div className="space-y-4">
      <div>
        <label className="text-xs text-zinc-500 uppercase tracking-wider">
          Traffic Limit (GB, 0 = unlimited)
        </label>
        <input
          type="number"
          min={0}
          value={form.traffic_limit_gb}
          onChange={(e) => setForm({ ...form, traffic_limit_gb: +e.target.value })}
          className={inputCls + " mt-1"}
        />
      </div>
      <div>
        <label className="text-xs text-zinc-500 uppercase tracking-wider">
          Traffic Used (readonly)
        </label>
        <p className="text-sm text-ink font-mono mt-1">
          {(user.traffic_used / 1024 / 1024 / 1024).toFixed(2)} GB
        </p>
      </div>
      <div className="flex items-center gap-3">
        <input
          type="checkbox"
          id="auto_reset"
          checked={form.auto_reset}
          onChange={(e) => setForm({ ...form, auto_reset: e.target.checked })}
        />
        <label htmlFor="auto_reset" className="text-sm text-ink">
          Auto-reset traffic every 30 days
        </label>
      </div>
      <div>
        <label className="text-xs text-zinc-500 uppercase tracking-wider">
          Expires At
        </label>
        <div className="flex gap-2 mt-1">
          <input
            type="date"
            value={form.expires_at}
            onChange={(e) => setForm({ ...form, expires_at: e.target.value })}
            className={inputCls + " flex-1"}
          />
          {[30, 90, 365].map((d) => (
            <button
              key={d}
              type="button"
              onClick={() => addDays(d)}
              className="px-3 py-2 text-xs bg-zinc-800 rounded hover:bg-zinc-700 text-zinc-300"
            >
              +{d}d
            </button>
          ))}
        </div>
      </div>
      <div className="flex items-center gap-3">
        <input
          type="checkbox"
          id="enabled"
          checked={form.enabled}
          onChange={(e) => setForm({ ...form, enabled: e.target.checked })}
        />
        <label htmlFor="enabled" className="text-sm text-ink">
          Enabled
        </label>
      </div>
      <div>
        <div className="flex items-center gap-3">
          <input
            type="checkbox"
            id="restrict"
            checked={form.restrict}
            onChange={(e) => setForm({ ...form, restrict: e.target.checked })}
          />
          <label htmlFor="restrict" className="text-sm text-ink">
            Restrict to specific nodes (otherwise all healthy nodes)
          </label>
        </div>
        {form.restrict && (
          <div className="mt-2 pl-6 space-y-1">
            {nodes.map((n) => (
              <label key={n.ID} className="flex items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  checked={form.allowedNodeIds.includes(n.ID)}
                  onChange={(e) => {
                    const set = new Set(form.allowedNodeIds)
                    if (e.target.checked) set.add(n.ID)
                    else set.delete(n.ID)
                    setForm({ ...form, allowedNodeIds: [...set] })
                  }}
                />
                <span className={n.healthy ? "text-ink" : "text-zinc-600"}>
                  {n.name}
                  {!n.healthy && " (offline)"}
                </span>
              </label>
            ))}
          </div>
        )}
      </div>
      {err && <p className="text-xs text-red-400">{err}</p>}
      <button
        onClick={save}
        className="px-4 py-2 text-sm bg-clay text-white rounded font-medium hover:bg-clay-hover"
      >
        Save Access Settings
      </button>
    </div>
  )
}

function RulesTab({
  user,
  onSaved,
}: {
  user: EditableUser
  onSaved: () => void
}) {
  const [r, setR] = useState({
    rule_ai: user.rule_ai || false,
    rule_streaming: user.rule_streaming || false,
    rule_china: user.rule_china !== undefined ? user.rule_china : true,
    rule_ad_block: user.rule_ad_block || false,
  })
  const [err, setErr] = useState("")

  const save = async () => {
    try {
      setErr("")
      await api.put(`/admin/users/${user.ID}`, r)
      onSaved()
    } catch (e: unknown) {
      setErr(
        (e as { response?: { data?: { error?: string } } }).response?.data?.error ||
          String(e)
      )
    }
  }

  const row = (key: keyof typeof r, label: string, desc: string) => (
    <label className="flex items-start gap-3 py-2 border-b border-zinc-800">
      <input
        type="checkbox"
        checked={r[key]}
        onChange={(e) => setR({ ...r, [key]: e.target.checked })}
        className="mt-1"
      />
      <div>
        <p className="text-sm text-ink">{label}</p>
        <p className="text-xs text-zinc-500">{desc}</p>
      </div>
    </label>
  )

  return (
    <div className="space-y-1">
      {row("rule_ai", "AI Proxy Chain", "OpenAI, Claude, Gemini via residential proxy chain.")}
      {row("rule_streaming", "Streaming", "Netflix, Disney+, HBO Max routed via VPN.")}
      {row("rule_china", "China Direct (CN)", "Domestic CN domains bypass tunnel (default ON).")}
      {row("rule_ad_block", "Ad Block", "Reject known ad / tracker domains.")}
      {err && <p className="text-xs text-red-400">{err}</p>}
      <button
        onClick={save}
        className="mt-4 px-4 py-2 text-sm bg-clay text-white rounded font-medium hover:bg-clay-hover"
      >
        Save Rules
      </button>
    </div>
  )
}

function QRModal({ userId, format, onClose }: {
  userId: number
  format: string
  onClose: () => void
}) {
  const [blobUrl, setBlobUrl] = useState("")
  const [err, setErr] = useState("")

  useEffect(() => {
    let revoke: string | null = null
    api
      .get(`/admin/users/${userId}/qrcode?format=${format}`, { responseType: "blob" })
      .then((r) => {
        const url = URL.createObjectURL(r.data as Blob)
        revoke = url
        setBlobUrl(url)
      })
      .catch((e) => setErr(String(e)))
    return () => {
      if (revoke) URL.revokeObjectURL(revoke)
    }
  }, [userId, format])

  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center bg-ink/55" onClick={onClose}>
      <div
        className="bg-surface p-4 rounded flex flex-col items-center"
        onClick={(e) => e.stopPropagation()}
      >
        {err ? (
          <p className="text-red-500 text-sm p-8">{err}</p>
        ) : blobUrl ? (
          <>
            <img src={blobUrl} width={384} height={384} alt="Subscription QR" />
            <p className="text-xs text-zinc-600 mt-2 uppercase tracking-wider">{format}</p>
          </>
        ) : (
          <p className="text-zinc-600 text-sm p-8">Loading...</p>
        )}
      </div>
    </div>
  )
}
