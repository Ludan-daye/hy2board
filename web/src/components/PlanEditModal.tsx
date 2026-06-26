import { useState } from "react"
import api from "@/api/client"
import { X } from "lucide-react"

export interface Plan {
  ID?: number
  name: string
  traffic_limit: number
  duration_days: number
  price_cents?: number
  node_ids: string
  rule_ai: boolean
  rule_streaming: boolean
  rule_china: boolean
  rule_ad_block: boolean
  auto_reset: boolean
  sort_order: number
  proxy_type?: string
  proxy_host?: string
  proxy_port?: number
  proxy_username?: string
  proxy_note?: string
}

type Tab = "basic" | "access" | "rules" | "ip"

interface FormState {
  name: string
  traffic_limit: number
  duration_days: number
  price_cents: number
  node_ids: string
  rule_ai: boolean
  rule_streaming: boolean
  rule_china: boolean
  rule_ad_block: boolean
  auto_reset: boolean
  sort_order: number
  proxy_type?: string
  proxy_host?: string
  proxy_port?: number
  proxy_username?: string
  proxy_password?: string
  proxy_note?: string
}

interface Props {
  plan: Plan | null
  onClose: () => void
  onSaved: () => void
  initialTab?: Tab
}

const blankForm: FormState = {
  name: "",
  traffic_limit: 0,
  duration_days: 30,
  price_cents: 0,
  node_ids: "all",
  rule_ai: false,
  rule_streaming: false,
  rule_china: true,
  rule_ad_block: false,
  auto_reset: false,
  sort_order: 0,
  proxy_type: "",
  proxy_host: "",
  proxy_port: 0,
  proxy_username: "",
  proxy_password: "",
  proxy_note: "",
}

export default function PlanEditModal({
  plan,
  onClose,
  onSaved,
  initialTab,
}: Props) {
  const [tab, setTab] = useState<Tab>(initialTab || "basic")
  const [form, setForm] = useState<FormState>(
    plan
      ? {
          name: plan.name,
          traffic_limit: plan.traffic_limit,
          duration_days: plan.duration_days,
          price_cents: plan.price_cents || 0,
          node_ids: plan.node_ids,
          rule_ai: plan.rule_ai,
          rule_streaming: plan.rule_streaming,
          rule_china: plan.rule_china,
          rule_ad_block: plan.rule_ad_block,
          auto_reset: plan.auto_reset,
          sort_order: plan.sort_order,
          proxy_type: plan.proxy_type || "",
          proxy_host: plan.proxy_host || "",
          proxy_port: plan.proxy_port || 0,
          proxy_username: plan.proxy_username || "",
          proxy_note: plan.proxy_note || "",
        }
      : { ...blankForm }
  )
  const [err, setErr] = useState("")
  const [showPwModal, setShowPwModal] = useState(false)
  const [pwInput, setPwInput] = useState("")
  // When admin paste-imports in edit mode, the parsed password is staged here
  // so save() can call set-proxy-password after the main PUT.
  const [quickImportPw, setQuickImportPw] = useState("")

  const save = async () => {
    try {
      setErr("")
      // Client-side: name is required; jump to Basic tab if missing.
      if (!form.name || !form.name.trim()) {
        setErr("请先在 Basic 标签页填写 Plan 名称")
        setTab("basic")
        return
      }
      if (plan?.ID) {
        // Edit mode: exclude proxy_password
        const body = {
          name: form.name,
          traffic_limit: form.traffic_limit,
          duration_days: form.duration_days,
          price_cents: form.price_cents,
          node_ids: form.node_ids,
          rule_ai: form.rule_ai,
          rule_streaming: form.rule_streaming,
          rule_china: form.rule_china,
          rule_ad_block: form.rule_ad_block,
          auto_reset: form.auto_reset,
          sort_order: form.sort_order,
          proxy_type: form.proxy_type,
          proxy_host: form.proxy_host,
          proxy_port: form.proxy_port,
          proxy_username: form.proxy_username,
          proxy_note: form.proxy_note,
        }
        await api.put(`/admin/plans/${plan.ID}`, body)
        // If a paste-import set a new password during edit, push it via the dedicated endpoint
        if (quickImportPw) {
          await api.post(`/admin/plans/${plan.ID}/set-proxy-password`, { password: quickImportPw })
          setQuickImportPw("")
        }
      } else {
        // Create mode: include proxy_password
        await api.post("/admin/plans", form)
      }
      onSaved()
      onClose()
    } catch (e: unknown) {
      setErr((e as { response?: { data?: { error?: string } } }).response?.data?.error || String(e))
    }
  }

  const inputCls =
    "w-full px-3 py-2 bg-surface border border-zinc-700 rounded-lg text-sm text-ink focus:outline-none focus:border-zinc-500"

  const tabBtnCls = (t: Tab) =>
    `px-3 py-1.5 text-xs ${tab === t ? "bg-zinc-800 text-ink" : "text-zinc-400 hover:text-ink"} rounded`

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-ink/45" onClick={onClose}>
      <div
        onClick={(e) => e.stopPropagation()}
        className="bg-zinc-900 border border-zinc-800 rounded-xl w-[520px] p-5 space-y-3"
      >
        {/* Header */}
        <div className="flex items-center justify-between">
          <h3 className="text-ink font-semibold">{plan?.ID ? "Edit Plan" : "New Plan"}</h3>
          <button onClick={onClose}>
            <X size={16} className="text-zinc-500" />
          </button>
        </div>

        {/* Tab strip */}
        <div className="flex gap-1 border-b border-zinc-800 pb-2">
          <button className={tabBtnCls("basic")} onClick={() => setTab("basic")}>Basic</button>
          <button className={tabBtnCls("access")} onClick={() => setTab("access")}>Access</button>
          <button className={tabBtnCls("rules")} onClick={() => setTab("rules")}>Rules</button>
          <button className={tabBtnCls("ip")} onClick={() => setTab("ip")}>IP</button>
        </div>

        {/* Basic tab */}
        {tab === "basic" && (
          <div className="space-y-3">
            <input
              placeholder="Name"
              value={form.name}
              onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
              className={inputCls}
            />
            <div className="grid grid-cols-2 gap-3">
              <input
                type="number"
                placeholder="Traffic GB (0 = unlimited)"
                value={form.traffic_limit / 1024 / 1024 / 1024 || ""}
                onChange={(e) =>
                  setForm((f) => ({ ...f, traffic_limit: +e.target.value * 1024 * 1024 * 1024 }))
                }
                className={inputCls}
              />
              <input
                type="number"
                placeholder="Duration days"
                value={form.duration_days}
                onChange={(e) => setForm((f) => ({ ...f, duration_days: +e.target.value }))}
                className={inputCls}
              />
            </div>
            <input
              type="number"
              placeholder="Sort order"
              value={form.sort_order}
              onChange={(e) => setForm((f) => ({ ...f, sort_order: +e.target.value }))}
              className={inputCls}
            />
            <label className="block">
              <span className="text-xs text-zinc-400">价格 (¥)</span>
              <input
                type="number"
                step="0.01"
                value={form.price_cents > 0 ? (form.price_cents / 100).toFixed(2) : ""}
                onChange={(e) =>
                  setForm((f) => ({
                    ...f,
                    price_cents: Math.round((parseFloat(e.target.value) || 0) * 100),
                  }))
                }
                className={inputCls + " mt-1"}
                placeholder="留空则不带默认金额"
              />
            </label>
          </div>
        )}

        {/* Access tab */}
        {tab === "access" && (
          <div className="space-y-3">
            <input
              placeholder="Node IDs (all or '1,3,5')"
              value={form.node_ids}
              onChange={(e) => setForm((f) => ({ ...f, node_ids: e.target.value }))}
              className={inputCls}
            />
            <label className="flex items-center gap-2 text-sm text-ink">
              <input
                type="checkbox"
                checked={form.auto_reset}
                onChange={(e) => setForm((f) => ({ ...f, auto_reset: e.target.checked }))}
              />
              Auto Reset 30d
            </label>
          </div>
        )}

        {/* Rules tab */}
        {tab === "rules" && (
          <div className="grid grid-cols-2 gap-3">
            {(
              [
                ["rule_ai", "AI Chain"],
                ["rule_streaming", "Streaming"],
                ["rule_china", "China Direct"],
                ["rule_ad_block", "Ad Block"],
              ] as [keyof FormState, string][]
            ).map(([k, label]) => (
              <label key={k} className="flex items-center gap-2 text-sm text-ink">
                <input
                  type="checkbox"
                  checked={form[k] as boolean}
                  onChange={(e) => setForm((f) => ({ ...f, [k]: e.target.checked }))}
                />
                {label}
              </label>
            ))}
          </div>
        )}

        {/* IP tab */}
        {tab === "ip" && (
          <div className="space-y-3">
            {/* Quick Import — paste host:port:user:password from supplier */}
            <div className="bg-clay/5 border border-clay/20 rounded p-2.5 space-y-1.5">
              <label className="text-xs text-clay uppercase tracking-wider">⚡ 一键粘贴导入</label>
              <input
                placeholder="host:port:user:password"
                onChange={(e) => {
                  const txt = e.target.value.trim()
                  if (!txt) return
                  const parts = txt.split(":")
                  if (parts.length < 4) return
                  const port = parseInt(parts[1], 10)
                  if (!parts[0] || !port) return
                  setForm((f) => ({
                    ...f,
                    proxy_type: f.proxy_type || "socks5",
                    proxy_host: parts[0],
                    proxy_port: port,
                    proxy_username: parts[2],
                    proxy_password: parts.slice(3).join(":"),
                  }))
                  e.target.value = ""
                  if (plan?.ID) {
                    setQuickImportPw(parts.slice(3).join(":"))
                  }
                }}
                className="w-full px-3 py-2 bg-surface border border-zinc-700 rounded text-xs font-mono text-ink"
              />
              <p className="text-[10px] text-zinc-500">
                粘贴一行 <code className="text-zinc-400">host:port:user:password</code>，自动填到下面 4 个字段。
                {plan?.ID && " 编辑模式下保存时密码会一起更新。"}
              </p>
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="text-xs text-zinc-500 uppercase">Type</label>
                <select
                  value={form.proxy_type || ""}
                  onChange={(e) => setForm((f) => ({ ...f, proxy_type: e.target.value }))}
                  className="w-full px-3 py-2 bg-surface border border-zinc-700 rounded text-sm text-ink"
                >
                  <option value="">(none — shared)</option>
                  <option value="socks5">socks5</option>
                  <option value="http">http</option>
                </select>
              </div>
              <div>
                <label className="text-xs text-zinc-500 uppercase">Port</label>
                <input
                  type="number"
                  value={form.proxy_port || ""}
                  onChange={(e) => setForm((f) => ({ ...f, proxy_port: +e.target.value }))}
                  className="w-full px-3 py-2 bg-surface border border-zinc-700 rounded text-sm text-ink"
                />
              </div>
            </div>
            <div>
              <label className="text-xs text-zinc-500 uppercase">Host</label>
              <input
                value={form.proxy_host || ""}
                onChange={(e) => setForm((f) => ({ ...f, proxy_host: e.target.value }))}
                placeholder="1.2.3.4"
                className="w-full px-3 py-2 bg-surface border border-zinc-700 rounded text-sm text-ink font-mono"
              />
            </div>
            <div>
              <label className="text-xs text-zinc-500 uppercase">Username</label>
              <input
                value={form.proxy_username || ""}
                onChange={(e) => setForm((f) => ({ ...f, proxy_username: e.target.value }))}
                className="w-full px-3 py-2 bg-surface border border-zinc-700 rounded text-sm text-ink font-mono"
              />
            </div>
            {!plan?.ID && (
              <div>
                <label className="text-xs text-zinc-500 uppercase">Initial Password</label>
                <input
                  type="password"
                  value={form.proxy_password || ""}
                  onChange={(e) => setForm((f) => ({ ...f, proxy_password: e.target.value }))}
                  className="w-full px-3 py-2 bg-surface border border-zinc-700 rounded text-sm text-ink"
                />
              </div>
            )}
            {plan?.ID && (
              <div>
                <label className="text-xs text-zinc-500 uppercase">Password</label>
                <div className="flex gap-2">
                  <input
                    value={plan?.proxy_host ? "(已设置)" : "(未设置)"}
                    disabled
                    className="flex-1 px-3 py-2 bg-zinc-900 border border-zinc-800 rounded text-sm text-zinc-500"
                  />
                  <button
                    type="button"
                    onClick={() => setShowPwModal(true)}
                    className="px-3 py-2 text-xs bg-zinc-800 hover:bg-zinc-700 rounded text-ink"
                  >
                    {plan?.proxy_host ? "Change" : "Set"}
                  </button>
                </div>
              </div>
            )}
            <div>
              <label className="text-xs text-zinc-500 uppercase">Note (备注，仅你可见，不是 Plan 名称)</label>
              <textarea
                rows={2}
                value={form.proxy_note || ""}
                onChange={(e) => setForm((f) => ({ ...f, proxy_note: e.target.value }))}
                className="w-full px-3 py-2 bg-surface border border-zinc-700 rounded text-sm text-ink"
                placeholder="e.g. 东京家宽 / 月付 100 USD / 续期日 2026-05-01"
              />
              <p className="text-[10px] text-amber-400/80 mt-1">
                ⚠️ Plan 名称在 Basic 标签页填写。这里是给自己看的备注（如供应商、续期日、给谁用）。
              </p>
            </div>
          </div>
        )}

        {err && <p className="text-xs text-red-400">{err}</p>}

        {/* Footer */}
        <div className="flex justify-end gap-2 pt-2">
          <button onClick={onClose} className="px-4 py-2 text-sm text-zinc-400">
            Cancel
          </button>
          <button
            onClick={save}
            className="px-4 py-2 text-sm bg-clay text-white rounded font-medium"
          >
            Save
          </button>
        </div>
      </div>

      {/* Change password sub-modal */}
      {showPwModal && plan?.ID && (
        <div
          className="fixed inset-0 bg-ink/45 flex items-center justify-center z-50"
          onClick={() => setShowPwModal(false)}
        >
          <div className="bg-zinc-900 rounded-lg p-6 w-96" onClick={(e) => e.stopPropagation()}>
            <h3 className="text-sm font-semibold text-ink mb-3">Change proxy password</h3>
            <input
              type="password"
              value={pwInput}
              onChange={(e) => setPwInput(e.target.value)}
              placeholder="new password"
              className="w-full px-3 py-2 bg-surface border border-zinc-700 rounded text-sm text-ink mb-4"
              autoFocus
            />
            <div className="flex justify-end gap-2">
              <button
                onClick={() => { setShowPwModal(false); setPwInput("") }}
                className="px-3 py-1.5 text-xs text-zinc-400 hover:text-ink"
              >
                Cancel
              </button>
              <button
                onClick={async () => {
                  if (!pwInput) return
                  await api.post(`/admin/plans/${plan.ID}/set-proxy-password`, { password: pwInput })
                  setShowPwModal(false)
                  setPwInput("")
                  alert("Password updated.")
                }}
                className="px-3 py-1.5 text-xs bg-clay hover:bg-clay text-ink rounded"
              >
                Save
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
