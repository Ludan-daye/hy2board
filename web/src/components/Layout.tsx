import { Link, useLocation } from "react-router-dom"
import { LayoutDashboard, Download, Server, Users, Package, LogOut, Zap, ChevronRight, Activity, Globe, Wallet, ScrollText, Bot, BellRing, ListFilter } from "lucide-react"
import { useState, useEffect } from "react"

const nav = [
  { to: "/", icon: LayoutDashboard, label: "Dashboard", desc: "Overview & analytics" },
  { to: "/alerts", icon: BellRing, label: "Alerts", desc: "Risk radar" },
  { to: "/nodes", icon: Server, label: "Nodes", desc: "Server management" },
  { to: "/rules", icon: ListFilter, label: "Rules", desc: "Routing policy" },
  { to: "/users", icon: Users, label: "Users", desc: "User accounts" },
  { to: "/plans", icon: Package, label: "Plans", desc: "Subscription templates" },
  { to: "/static-ips", icon: Globe, label: "Static IPs", desc: "Dedicated IP catalog" },
  { to: "/finance", icon: Wallet, label: "Finance", desc: "Income & payments" },
  { to: "/telegram", icon: Bot, label: "Telegram", desc: "Bot operations" },
  { to: "/audit-logs", icon: ScrollText, label: "Audit Logs", desc: "Admin activity trail" },
  { to: "/downloads", icon: Download, label: "Downloads", desc: "Client apps" },
]

export default function Layout({ children }: { children: React.ReactNode }) {
  const location = useLocation()
  const [time, setTime] = useState("")

  useEffect(() => {
    const tick = () => setTime(new Date().toLocaleTimeString("en-US", { hour12: false }))
    tick()
    const t = setInterval(tick, 1000)
    return () => clearInterval(t)
  }, [])

  const logout = () => {
    localStorage.removeItem("token")
    localStorage.removeItem("role")
    window.location.href = "/login"
  }

  return (
    <div className="min-h-screen flex" style={{ background: "#08090d" }}>
      {/* Sidebar */}
      <aside
        className="w-[240px] flex flex-col relative shrink-0"
        style={{ background: "#0c0d12", borderRight: "1px solid rgba(20,20,19,0.08)" }}
      >
        {/* Logo */}
        <div className="px-5 py-6" style={{ borderBottom: "1px solid rgba(20,20,19,0.08)" }}>
          <div className="flex items-center gap-3">
            <div
              className="w-9 h-9 rounded-lg flex items-center justify-center"
              style={{ background: "linear-gradient(135deg, #C96442 0%, #B5573A 100%)" }}
            >
              <Zap size={16} className="text-ink" />
            </div>
            <div>
              <h1 className="text-[15px] font-bold tracking-wide text-ink" style={{ fontFamily: "'Alumni Sans', sans-serif" }}>
                HY2BOARD
              </h1>
              <p className="text-[10px] tracking-[0.15em] text-zinc-500 uppercase">Admin Panel</p>
            </div>
          </div>
        </div>

        {/* Navigation */}
        <nav className="flex-1 px-3 py-4 space-y-1">
          <p className="px-3 mb-3 text-[10px] font-semibold tracking-[0.2em] text-zinc-600 uppercase">Navigation</p>
          {nav.map(({ to, icon: Icon, label, desc }) => {
            const active = location.pathname === to
            return (
              <Link
                key={to}
                to={to}
                className="group flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm transition-all duration-200 relative"
                style={{
                  background: active ? "rgba(43, 71, 255, 0.1)" : "transparent",
                  color: active ? "#fff" : "rgba(255,255,255,0.4)",
                }}
              >
                {/* Active indicator */}
                {active && (
                  <div
                    className="absolute left-0 top-1/2 -translate-y-1/2 w-[3px] h-5 rounded-r-full"
                    style={{ background: "#C96442" }}
                  />
                )}
                <div
                  className="w-8 h-8 rounded-lg flex items-center justify-center transition-all duration-200 shrink-0"
                  style={{
                    background: active ? "rgba(43, 71, 255, 0.15)" : "rgba(20,20,19,0.04)",
                    color: active ? "#C96442" : undefined,
                  }}
                >
                  <Icon size={15} />
                </div>
                <div className="flex-1 min-w-0">
                  <p className={"text-[13px] font-medium " + (active ? "text-ink" : "text-zinc-400 group-hover:text-zinc-200")} style={{ transition: "color 0.2s" }}>
                    {label}
                  </p>
                  <p className="text-[10px] text-zinc-600 truncate">{desc}</p>
                </div>
                {active && <ChevronRight size={12} className="text-zinc-600" />}
              </Link>
            )
          })}
        </nav>

        {/* Bottom section */}
        <div className="px-3 pb-4 space-y-2">
          {/* Service Status link */}
          <a
            href="/status/"
            target="_blank"
            rel="noopener noreferrer"
            className="mx-2 flex items-center gap-2.5 px-3 py-2.5 rounded-lg text-[12px] font-medium text-green-400 transition-colors"
            style={{ background: "rgba(34,197,94,0.06)", border: "1px solid rgba(34,197,94,0.12)" }}
            onMouseEnter={(e) => { e.currentTarget.style.background = "rgba(34,197,94,0.12)" }}
            onMouseLeave={(e) => { e.currentTarget.style.background = "rgba(34,197,94,0.06)" }}
          >
            <Activity size={14} />
            Service Status
          </a>

          {/* System status */}
          <div
            className="mx-2 px-3 py-3 rounded-lg"
            style={{ background: "rgba(20,20,19,0.03)", border: "1px solid rgba(20,20,19,0.06)" }}
          >
            <div className="flex items-center justify-between mb-2">
              <span className="text-[10px] font-semibold tracking-[0.15em] text-zinc-600 uppercase">System</span>
              <span className="text-[10px] font-mono text-zinc-600">{time}</span>
            </div>
            <div className="flex items-center gap-2">
              <span className="relative flex h-2 w-2">
                <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75" />
                <span className="relative inline-flex rounded-full h-2 w-2 bg-green-500" />
              </span>
              <span className="text-[11px] text-green-400/80">All systems operational</span>
            </div>
          </div>

          {/* Logout */}
          <button
            onClick={logout}
            className="w-full flex items-center gap-3 px-5 py-2.5 rounded-lg text-sm text-zinc-500 hover:text-red-400 hover:bg-red-500/5 transition-all duration-200"
          >
            <LogOut size={15} />
            <span className="text-[13px]">Sign Out</span>
          </button>
        </div>
      </aside>

      {/* Main content */}
      <main className="flex-1 overflow-auto">
        {/* Top bar */}
        <div
          className="sticky top-0 z-30 flex items-center justify-between px-8 py-4"
          style={{
            background: "rgba(8, 9, 13, 0.8)",
            backdropFilter: "blur(12px)",
            borderBottom: "1px solid rgba(20,20,19,0.06)",
          }}
        >
          <div>
            <h2 className="text-[15px] font-semibold text-ink">
              {nav.find((n) => n.to === location.pathname)?.label || "Page"}
            </h2>
            <p className="text-[11px] text-zinc-500">
              {nav.find((n) => n.to === location.pathname)?.desc || ""}
            </p>
          </div>
          <div className="flex items-center gap-3">
            <div className="text-[11px] text-zinc-600 font-mono">{new Date().toLocaleDateString("en-US", { weekday: "short", month: "short", day: "numeric" })}</div>
            <div className="w-8 h-8 rounded-full flex items-center justify-center text-[11px] font-bold text-ink" style={{ background: "linear-gradient(135deg, #C96442, #B5573A)" }}>
              A
            </div>
          </div>
        </div>

        {/* Page content */}
        <div className="p-8 text-ink">{children}</div>
      </main>
    </div>
  )
}
