import { useState } from "react"

export interface PaymentInput {
  amount_cents: number
  kind: "new" | "renew"
  note: string
  paid_at?: string  // ISO date "YYYY-MM-DD"
}

interface Props {
  title: string
  defaultDays: number
  defaultAmountCents: number   // 0 = blank
  defaultKind: "new" | "renew"
  showDays: boolean             // false when invoked from Apply Plan (days = plan.duration_days)
  onCancel: () => void
  onConfirm: (days: number, payment: PaymentInput | null) => Promise<void> | void
}

export default function PaymentDialog(p: Props) {
  const [days, setDays] = useState(p.defaultDays)
  const [paid, setPaid] = useState<"yes" | "no" | "">("")
  const [amountYuan, setAmountYuan] = useState(p.defaultAmountCents > 0 ? (p.defaultAmountCents / 100).toFixed(2) : "")
  const [note, setNote] = useState("")
  const [paidAt, setPaidAt] = useState(new Date().toISOString().slice(0, 10))
  const [kind, setKind] = useState<"new" | "renew">(p.defaultKind)
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState("")

  const submit = async () => {
    setErr("")
    if (paid === "") { setErr("请选择 已缴费 / 不缴费"); return }
    if (p.showDays && days <= 0) { setErr("续期天数必须大于 0"); return }
    if (paid === "yes") {
      const a = parseFloat(amountYuan || "0")
      if (isNaN(a) || a < 0) { setErr("金额必须 >= 0"); return }
      if (paidAt > new Date().toISOString().slice(0,10)) { setErr("缴费日期不能是未来"); return }
    }
    setBusy(true)
    try {
      const payment: PaymentInput | null = paid === "yes"
        ? { amount_cents: Math.round(parseFloat(amountYuan || "0") * 100), kind, note, paid_at: paidAt + "T00:00:00Z" }
        : null
      await p.onConfirm(days, payment)
    } catch (e: any) {
      setErr(e?.response?.data?.error || String(e))
    } finally {
      setBusy(false)
    }
  }

  const inputCls = "px-3 py-2 bg-surface border border-zinc-700 rounded text-sm text-ink"

  return (
    <div
      className="fixed inset-0 bg-ink/40 flex items-center justify-center z-50"
      onClick={(e) => e.stopPropagation()}
    >
      <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-6 w-[420px] max-w-[90vw]">
        <h3 className="text-lg font-semibold mb-4">{p.title}</h3>
        {err && <div className="mb-3 text-sm text-red-400">{err}</div>}

        {p.showDays && (
          <label className="block text-sm mb-3">
            <span className="text-zinc-400">续期天数</span>
            <input type="number" value={days} onChange={e => setDays(+e.target.value)} className={inputCls + " w-full mt-1"} />
          </label>
        )}

        <div className="space-y-2 mb-4">
          <label className="flex items-center gap-2 text-sm cursor-pointer">
            <input type="radio" checked={paid === "yes"} onChange={() => setPaid("yes")} />
            <span>本月已缴费</span>
          </label>

          {paid === "yes" && (
            <div className="ml-6 space-y-2">
              <label className="block text-sm">
                <span className="text-zinc-400">金额 (¥)</span>
                <input type="number" step="0.01" value={amountYuan} onChange={e => setAmountYuan(e.target.value)} className={inputCls + " w-full mt-1"} placeholder="30.00" />
              </label>
              <label className="block text-sm">
                <span className="text-zinc-400">备注</span>
                <input value={note} onChange={e => setNote(e.target.value)} className={inputCls + " w-full mt-1"} placeholder="微信 / 朋友代付 / ..." />
              </label>
              <label className="block text-sm">
                <span className="text-zinc-400">缴费日期</span>
                <input type="date" value={paidAt} onChange={e => setPaidAt(e.target.value)} className={inputCls + " w-full mt-1"} />
              </label>
              <div className="flex gap-3 text-sm">
                <label className="flex items-center gap-1 cursor-pointer">
                  <input type="radio" checked={kind === "renew"} onChange={() => setKind("renew")} /> 续费
                </label>
                <label className="flex items-center gap-1 cursor-pointer">
                  <input type="radio" checked={kind === "new"} onChange={() => setKind("new")} /> 新购
                </label>
              </div>
            </div>
          )}

          <label className="flex items-center gap-2 text-sm cursor-pointer">
            <input type="radio" checked={paid === "no"} onChange={() => setPaid("no")} />
            <span>不缴费 (赠送/试用)</span>
          </label>
        </div>

        <div className="flex justify-end gap-2">
          <button onClick={p.onCancel} disabled={busy} className="px-3 py-2 text-sm text-zinc-400 hover:text-ink">Cancel</button>
          <button onClick={submit} disabled={busy || paid === ""} className="px-3 py-2 bg-clay text-white rounded text-sm font-medium hover:bg-clay-hover disabled:opacity-40">
            {busy ? "..." : "Confirm"}
          </button>
        </div>
      </div>
    </div>
  )
}
