import { useState, useEffect, useRef } from "react"
import { useNavigate } from "react-router-dom"
import api from "@/api/client"

/* ─── Loading Screen ─── */
function LoadingScreen({ onDone }: { onDone: () => void }) {
  const [count, setCount] = useState(0)

  useEffect(() => {
    const interval = setInterval(() => {
      setCount((c) => {
        if (c >= 100) {
          clearInterval(interval)
          setTimeout(onDone, 400)
          return 100
        }
        return c + Math.floor(Math.random() * 8 + 2)
      })
    }, 40)
    return () => clearInterval(interval)
  }, [onDone])

  return (
    <div
      className="fixed inset-0 z-50 flex flex-col items-center justify-center overflow-hidden"
      style={{
        background: "var(--accent)",
        transition: "transform 0.7s cubic-bezier(0.76, 0, 0.24, 1)",
        transform: count >= 100 ? "translateY(-100%)" : "translateY(0)",
      }}
    >
      <div className="relative z-10 flex flex-col items-center">
        <div style={{ fontFamily: "var(--font-alt)", fontSize: "clamp(48px, 8vw, 120px)", color: "#fff", textTransform: "uppercase", letterSpacing: "0.08em", lineHeight: 1 }}>
          {"HY2BOARD".split("").map((ch, i) => (
            <span
              key={i}
              className="inline-block"
              style={{
                animation: `loadUp 0.5s cubic-bezier(0.16, 1, 0.3, 1) ${i * 0.04}s forwards`,
                transform: "translateY(100%)",
                opacity: 0,
              }}
            >
              {ch}
            </span>
          ))}
        </div>
        <div
          style={{
            fontFamily: "var(--font-head)",
            fontSize: "clamp(36px, 6vw, 80px)",
            fontWeight: 900,
            color: "rgba(255,255,255,0.12)",
            letterSpacing: "0.1em",
            marginTop: "0.5rem",
          }}
        >
          {String(Math.min(count, 100)).padStart(3, "0")}
        </div>
        <div className="mt-6" style={{ width: 180, height: 2, background: "rgba(255,255,255,0.15)", borderRadius: 1 }}>
          <div style={{ width: `${Math.min(count, 100)}%`, height: "100%", background: "#fff", borderRadius: 1, transition: "width 0.08s" }} />
        </div>
      </div>
    </div>
  )
}

/* ─── Grain overlay ─── */
function Grain() {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  useEffect(() => {
    const c = canvasRef.current
    if (!c) return
    const ctx = c.getContext("2d")
    if (!ctx) return
    c.width = 256
    c.height = 256
    const img = ctx.createImageData(256, 256)
    for (let i = 0; i < img.data.length; i += 4) {
      const v = Math.random() * 255
      img.data[i] = img.data[i + 1] = img.data[i + 2] = v
      img.data[i + 3] = 12
    }
    ctx.putImageData(img, 0, 0)
  }, [])
  return (
    <canvas
      ref={canvasRef}
      className="fixed inset-0 pointer-events-none z-40"
      style={{ width: "100%", height: "100%", opacity: 0.4, mixBlendMode: "multiply" }}
    />
  )
}

/* ─── Main ─── */
export default function Login() {
  const [username, setUsername] = useState("")
  const [password, setPassword] = useState("")
  const [error, setError] = useState("")
  const [isAdmin, setIsAdmin] = useState(false)
  const [loading, setLoading] = useState(false)
  const [loaded, setLoaded] = useState(false)
  const [entered, setEntered] = useState(false)
  const navigate = useNavigate()

  useEffect(() => {
    if (loaded) setTimeout(() => setEntered(true), 100)
  }, [loaded])

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!username || !password || loading) return
    setError("")
    setLoading(true)
    try {
      const ep = isAdmin ? "/admin/login" : "/user/login"
      const { data } = await api.post(ep, { username, password })
      localStorage.setItem("token", data.token)
      localStorage.setItem("role", isAdmin ? "admin" : "user")
      navigate(isAdmin ? "/" : "/portal")
    } catch {
      setError("ACCESS DENIED")
      setLoading(false)
    }
  }

  return (
    <>
      <link
        href="https://fonts.googleapis.com/css2?family=Alumni+Sans:wght@400;700;900&family=Gasoek+One&family=Noto+Serif+SC:wght@200;400;700&display=swap"
        rel="stylesheet"
      />

      <style>{`
        :root {
          --accent: #2b47ff;
          --accent-light: #9499ff;
          --gold: #ffae00;
          --black: #000;
          --white: #fff;
          --border: 2px;
          --font-head: "Alumni Sans", sans-serif;
          --font-alt: "Gasoek One", sans-serif;
          --font-body: "Noto Serif SC", serif;
        }
        @keyframes loadUp {
          to { transform: translateY(0); opacity: 1; }
        }
        @keyframes fadeUp {
          from { transform: translateY(24px); opacity: 0; }
          to { transform: translateY(0); opacity: 1; }
        }
        @keyframes borderGrow {
          from { transform: scaleX(0); }
          to { transform: scaleX(1); }
        }
        @keyframes shake {
          0%, 100% { transform: translateX(0); }
          20% { transform: translateX(-8px); }
          40% { transform: translateX(8px); }
          60% { transform: translateX(-5px); }
          80% { transform: translateX(5px); }
        }
        .shake { animation: shake 0.4s ease-in-out; }
        .login-input {
          width: 100%;
          padding: 16px 0;
          font-family: var(--font-head);
          font-weight: 700;
          font-size: clamp(16px, 2.5vw, 22px);
          letter-spacing: 0.05em;
          text-transform: uppercase;
          background: transparent;
          border: none;
          border-bottom: var(--border) solid var(--black);
          outline: none;
          color: var(--black);
          transition: border-color 0.3s;
        }
        .login-input::placeholder { color: rgba(0,0,0,0.2); text-transform: uppercase; }
        .login-input:focus { border-bottom-color: var(--accent); }
        .login-input[type="password"] { letter-spacing: 0.15em; }
      `}</style>

      {!loaded && <LoadingScreen onDone={() => setLoaded(true)} />}
      <Grain />

      <div className="min-h-screen flex" style={{ background: "var(--white)", color: "var(--black)" }}>

        {/* ─── Left: Branding panel ─── */}
        <div
          className="hidden lg:flex flex-col justify-between relative overflow-hidden"
          style={{
            width: "45%",
            background: "var(--accent)",
            borderRight: "var(--border) solid var(--black)",
            opacity: entered ? 1 : 0,
            transform: entered ? "translateX(0)" : "translateX(-40px)",
            transition: "all 0.8s cubic-bezier(0.16, 1, 0.3, 1)",
          }}
        >
          {/* Grid pattern */}
          <div className="absolute inset-0 opacity-[0.06]" style={{
            backgroundImage: `
              linear-gradient(var(--white) 1px, transparent 1px),
              linear-gradient(90deg, var(--white) 1px, transparent 1px)
            `,
            backgroundSize: "60px 60px",
          }} />

          {/* Top bar */}
          <div className="relative z-10 flex items-center justify-between p-6" style={{ borderBottom: "var(--border) solid rgba(255,255,255,0.15)" }}>
            <span style={{ fontFamily: "var(--font-alt)", fontSize: 14, color: "#fff", letterSpacing: "0.05em", textTransform: "uppercase" }}>
              LinkByFree
            </span>
            <span style={{ fontFamily: "var(--font-head)", fontSize: 12, fontWeight: 700, color: "rgba(255,255,255,0.5)", letterSpacing: "0.15em", textTransform: "uppercase" }}>
              Network
            </span>
          </div>

          {/* Center title */}
          <div className="relative z-10 px-10">
            <div
              style={{
                fontFamily: "var(--font-alt)",
                fontSize: "clamp(56px, 7vw, 110px)",
                color: "#fff",
                textTransform: "uppercase",
                lineHeight: 0.85,
                letterSpacing: "-0.02em",
              }}
            >
              <div style={{ borderBottom: "var(--border) dashed rgba(255,255,255,0.2)", paddingBottom: "0.3em" }}>
                HY2
                <br />
                <span style={{ color: "var(--gold)" }}>BOARD</span>
              </div>
            </div>
            <p
              className="mt-6"
              style={{
                fontFamily: "var(--font-body)",
                fontSize: "clamp(14px, 1.5vw, 18px)",
                color: "rgba(255,255,255,0.6)",
                lineHeight: 1.8,
                maxWidth: 320,
                fontWeight: 200,
              }}
            >
              Hysteria 2 网络管理面板<br />
              高速、安全、全球节点
            </p>
          </div>

          {/* Bottom stats */}
          <div className="relative z-10 flex" style={{ borderTop: "var(--border) solid rgba(255,255,255,0.15)" }}>
            {[
              { label: "NODES", value: "4" },
              { label: "PROTOCOL", value: "HY2" },
              { label: "STATUS", value: "ONLINE" },
            ].map((s, i) => (
              <div
                key={s.label}
                className="flex-1 p-5"
                style={{ borderRight: i < 2 ? "var(--border) solid rgba(255,255,255,0.15)" : "none" }}
              >
                <div style={{ fontFamily: "var(--font-head)", fontSize: 10, fontWeight: 700, letterSpacing: "0.3em", color: "rgba(255,255,255,0.4)", textTransform: "uppercase" }}>
                  {s.label}
                </div>
                <div style={{ fontFamily: "var(--font-head)", fontSize: "clamp(20px, 2.5vw, 32px)", fontWeight: 900, color: "#fff", marginTop: 4 }}>
                  {s.value}
                  {s.label === "STATUS" && (
                    <span className="inline-block w-2 h-2 rounded-full ml-2 align-middle" style={{ background: "#22c55e", boxShadow: "0 0 8px #22c55e" }} />
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* ─── Right: Login form ─── */}
        <div className="flex-1 flex flex-col justify-center items-center px-6 lg:px-16 relative">

          {/* Mobile logo */}
          <div
            className="lg:hidden mb-10 text-center"
            style={{
              opacity: entered ? 1 : 0,
              transform: entered ? "translateY(0)" : "translateY(20px)",
              transition: "all 0.6s cubic-bezier(0.16, 1, 0.3, 1) 0.1s",
            }}
          >
            <div style={{ fontFamily: "var(--font-alt)", fontSize: 36, textTransform: "uppercase", lineHeight: 1 }}>
              HY2<span style={{ color: "var(--accent)" }}>BOARD</span>
            </div>
            <div style={{ fontFamily: "var(--font-head)", fontSize: 11, fontWeight: 700, letterSpacing: "0.3em", color: "rgba(0,0,0,0.35)", marginTop: 8, textTransform: "uppercase" }}>
              Secure Network Gateway
            </div>
          </div>

          <div
            className="w-full max-w-[400px]"
            style={{
              opacity: entered ? 1 : 0,
              transform: entered ? "translateY(0)" : "translateY(30px)",
              transition: "all 0.7s cubic-bezier(0.16, 1, 0.3, 1) 0.2s",
            }}
          >
            {/* Section label */}
            <div className="flex items-center gap-4 mb-8">
              <div style={{ fontFamily: "var(--font-head)", fontSize: 11, fontWeight: 700, letterSpacing: "0.4em", color: "rgba(0,0,0,0.35)", textTransform: "uppercase" }}>
                Sign In
              </div>
              <div className="flex-1" style={{ height: "var(--border)", background: "var(--black)" }} />
            </div>

            {/* Role switch */}
            <div className="flex mb-10" style={{ border: "var(--border) solid var(--black)" }}>
              {(["user", "admin"] as const).map((role) => {
                const active = (role === "admin") === isAdmin
                return (
                  <button
                    key={role}
                    type="button"
                    onClick={() => setIsAdmin(role === "admin")}
                    className="flex-1 relative transition-all duration-300"
                    style={{
                      padding: "14px 0",
                      fontFamily: "var(--font-head)",
                      fontWeight: 900,
                      fontSize: "clamp(12px, 1.5vw, 14px)",
                      letterSpacing: "0.2em",
                      textTransform: "uppercase",
                      background: active ? (role === "admin" ? "var(--gold)" : "var(--accent)") : "transparent",
                      color: active ? (role === "admin" ? "var(--black)" : "var(--white)") : "rgba(0,0,0,0.3)",
                      borderRight: role === "user" ? "var(--border) solid var(--black)" : "none",
                      cursor: "pointer",
                    }}
                  >
                    {role === "admin" ? "◆ ADMIN" : "◇ USER"}
                  </button>
                )
              })}
            </div>

            <form onSubmit={handleLogin}>
              {error && (
                <div
                  className="mb-6 py-3 text-center shake"
                  style={{
                    fontFamily: "var(--font-head)",
                    fontSize: 13,
                    fontWeight: 700,
                    letterSpacing: "0.15em",
                    textTransform: "uppercase",
                    color: "#dc2626",
                    border: "var(--border) solid #dc2626",
                    background: "rgba(220,38,38,0.04)",
                  }}
                >
                  ✕ {error}
                </div>
              )}

              {/* Username */}
              <div className="mb-2">
                <label style={{ fontFamily: "var(--font-head)", fontSize: 10, fontWeight: 700, letterSpacing: "0.35em", color: "rgba(0,0,0,0.3)", textTransform: "uppercase" }}>
                  Username
                </label>
                <input
                  type="text"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  className="login-input"
                  placeholder="Enter username"
                  autoComplete="username"
                />
              </div>

              {/* Password */}
              <div className="mb-8 mt-6">
                <label style={{ fontFamily: "var(--font-head)", fontSize: 10, fontWeight: 700, letterSpacing: "0.35em", color: "rgba(0,0,0,0.3)", textTransform: "uppercase" }}>
                  Password
                </label>
                <input
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="login-input"
                  placeholder="••••••••"
                  autoComplete="current-password"
                />
              </div>

              {/* Submit */}
              <button
                type="submit"
                disabled={loading || !username || !password}
                className="group w-full relative overflow-hidden transition-all duration-300"
                style={{
                  padding: "18px 0",
                  fontFamily: "var(--font-head)",
                  fontWeight: 900,
                  fontSize: "clamp(13px, 1.5vw, 16px)",
                  letterSpacing: "0.2em",
                  textTransform: "uppercase",
                  border: "var(--border) solid var(--black)",
                  background: !username || !password ? "transparent" : isAdmin ? "var(--gold)" : "var(--accent)",
                  color: !username || !password ? "rgba(0,0,0,0.2)" : isAdmin ? "var(--black)" : "var(--white)",
                  cursor: !username || !password ? "not-allowed" : "pointer",
                }}
              >
                {loading ? (
                  <span className="inline-flex items-center gap-3">
                    <span className="inline-block w-4 h-4 border-2 border-current border-t-transparent rounded-full animate-spin" />
                    AUTHENTICATING...
                  </span>
                ) : (
                  <span className="inline-flex items-center gap-3">
                    INITIATE ACCESS
                    <span className="inline-block transition-transform duration-300 group-hover:translate-x-2">&rarr;</span>
                  </span>
                )}
              </button>
            </form>

            {/* Footer */}
            <div className="mt-10 flex items-center justify-between">
              <span style={{ fontFamily: "var(--font-head)", fontSize: 10, fontWeight: 700, letterSpacing: "0.2em", color: "rgba(0,0,0,0.2)", textTransform: "uppercase" }}>
                Hysteria 2 Protocol
              </span>
              <span style={{ fontFamily: "var(--font-head)", fontSize: 10, fontWeight: 700, letterSpacing: "0.2em", color: "rgba(0,0,0,0.2)", textTransform: "uppercase" }}>
                &copy; LinkByFree
              </span>
            </div>
          </div>
        </div>
      </div>
    </>
  )
}
