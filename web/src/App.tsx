import { lazy, Suspense } from "react"
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom"
import Login from "@/pages/Login"

// Lazy-load everything except Login to keep first-paint bundle small
const Layout = lazy(() => import("@/components/Layout"))
const Dashboard = lazy(() => import("@/pages/Dashboard"))
const Alerts = lazy(() => import("@/pages/Alerts"))
const Nodes = lazy(() => import("@/pages/Nodes"))
const Rules = lazy(() => import("@/pages/Rules"))
const Users = lazy(() => import("@/pages/Users"))
const UserPortal = lazy(() => import("@/pages/UserPortal"))
const Plans = lazy(() => import("@/pages/Plans"))
const StaticIPs = lazy(() => import("@/pages/StaticIPs"))
const Finance = lazy(() => import("@/pages/Finance"))
const TelegramBot = lazy(() => import("@/pages/TelegramBot"))
const AuditLogs = lazy(() => import("@/pages/AuditLogs"))
const Downloads = lazy(() => import("@/pages/Downloads"))

function Fallback() {
  return (
    <div className="min-h-screen flex items-center justify-center bg-black text-zinc-500 text-sm">
      <span className="inline-block w-4 h-4 border-2 border-zinc-700 border-t-zinc-400 rounded-full animate-spin mr-2" />
      Loading...
    </div>
  )
}

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const token = localStorage.getItem("token")
  const role = localStorage.getItem("role")
  if (!token) return <Navigate to="/login" />
  if (role === "user") return <Navigate to="/portal" />
  return (
    <Suspense fallback={<Fallback />}>
      <Layout>{children}</Layout>
    </Suspense>
  )
}

function UserRoute({ children }: { children: React.ReactNode }) {
  const token = localStorage.getItem("token")
  if (!token) return <Navigate to="/login" />
  return <Suspense fallback={<Fallback />}>{children}</Suspense>
}

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route path="/downloads" element={<Suspense fallback={<Fallback />}><Downloads /></Suspense>} />
        <Route path="/portal" element={<UserRoute><UserPortal /></UserRoute>} />
        <Route path="/" element={<ProtectedRoute><Dashboard /></ProtectedRoute>} />
        <Route path="/alerts" element={<ProtectedRoute><Alerts /></ProtectedRoute>} />
        <Route path="/nodes" element={<ProtectedRoute><Nodes /></ProtectedRoute>} />
        <Route path="/rules" element={<ProtectedRoute><Rules /></ProtectedRoute>} />
        <Route path="/users" element={<ProtectedRoute><Users /></ProtectedRoute>} />
        <Route path="/plans" element={<ProtectedRoute><Plans /></ProtectedRoute>} />
        <Route path="/static-ips" element={<ProtectedRoute><StaticIPs /></ProtectedRoute>} />
        <Route path="/finance" element={<ProtectedRoute><Finance /></ProtectedRoute>} />
        <Route path="/telegram" element={<ProtectedRoute><TelegramBot /></ProtectedRoute>} />
        <Route path="/audit-logs" element={<ProtectedRoute><AuditLogs /></ProtectedRoute>} />
      </Routes>
    </BrowserRouter>
  )
}
