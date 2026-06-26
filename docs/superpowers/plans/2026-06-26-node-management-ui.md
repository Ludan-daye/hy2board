# Node Management UI (VLESS/Trojan/Reality) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let the admin enable/configure VLESS+Reality and Trojan node fields from the Nodes page — by pasting the setup-script REGISTER output and via manual form fields — eliminating raw SQL.

**Architecture:** Extend the node handler (`NodeRequest`/Create/Update) to accept the existing-but-SQL-only columns; add a collapsible "TCP 回落" section + a paste-REGISTER parser to the React Nodes page. DB columns already exist (added in the VLESS/Trojan work).

**Tech Stack:** Go (Gin/GORM/SQLite) backend; React 19 + Vite frontend. Build/test in `golang:1.26-alpine` (CGO_ENABLED=1). Spec: `docs/superpowers/specs/2026-06-26-node-management-ui-design.md`.

**Test runner:** `docker run --rm -v /opt/hy2board:/app -v hy2board_gocache:/go -w /app -e CGO_ENABLED=1 golang:1.26-alpine sh -c "apk add --no-cache gcc musl-dev >/dev/null 2>&1; go test ./internal/handler/ -run <name> -v"`

**Sync note:** edit in `/root/ludandaye/ladder/hy2board` (commit/push) AND scp each changed file to `/opt/hy2board` before building.

---

## File structure

| File | Responsibility | New? |
|---|---|---|
| `internal/handler/node.go` | NodeRequest + Create/Update accept vless/trojan fields | modify |
| `internal/handler/node_update_test.go` | TDD: UpdateNode persists fields, write-only secret | create |
| `web/src/utils/nodeRegister.ts` (+ no runner → build-verified) | parse REGISTER blocks → form fields | create |
| `web/src/pages/Nodes.tsx` | form section + paste-register + state | modify |

---

## Task 1: Backend — accept VLESS/Trojan fields

**Files:** `internal/handler/node.go`, `internal/handler/node_update_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/handler/node_update_test.go
package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ludandaye/hy2board/internal/database"
	"github.com/ludandaye/hy2board/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUpdateNodePersistsVlessTrojanFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&model.Node{}, &model.AuditLog{})
	database.DB = db
	n := model.Node{Name: "HK1", Host: "1.2.3.4", Port: 443, VlessStatsSecret: "oldsecret"}
	db.Create(&n)

	r := gin.New()
	r.PUT("/admin/nodes/:id", UpdateNode)
	put := func(body string) {
		req := httptest.NewRequest("PUT", "/admin/nodes/"+fmt.Sprint(n.ID), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("PUT got %d: %s", w.Code, w.Body.String())
		}
	}

	put(`{"name":"HK1","host":"1.2.3.4","port":443,
	  "vless_enabled":true,"vless_port":443,"reality_pubkey":"PUB","reality_shortid":"ab12",
	  "reality_sni":"www.apple.com","vless_stats_api":"http://1.2.3.4:25415/","vless_stats_secret":"",
	  "trojan_enabled":true,"trojan_port":8443,"trojan_sni":"www.apple.com"}`)

	var got model.Node
	db.First(&got, n.ID)
	if !got.VlessEnabled || got.VlessPort != 443 || got.RealityPubkey != "PUB" || got.RealityShortID != "ab12" ||
		got.RealitySNI != "www.apple.com" || got.VlessStatsAPI != "http://1.2.3.4:25415/" ||
		!got.TrojanEnabled || got.TrojanPort != 8443 || got.TrojanSNI != "www.apple.com" {
		t.Fatalf("fields not persisted: %#v", got)
	}
	if got.VlessStatsSecret != "oldsecret" {
		t.Fatalf("blank secret must leave existing unchanged, got %q", got.VlessStatsSecret)
	}

	// non-empty secret replaces; disabling persists false
	put(`{"name":"HK1","host":"1.2.3.4","port":443,"vless_enabled":false,"vless_stats_secret":"news3cret","trojan_enabled":false}`)
	db.First(&got, n.ID)
	if got.VlessEnabled || got.TrojanEnabled {
		t.Fatalf("disable must persist false")
	}
	if got.VlessStatsSecret != "news3cret" {
		t.Fatalf("non-empty secret must replace, got %q", got.VlessStatsSecret)
	}
}
```

- [ ] **Step 2: Run, expect FAIL** (fields not persisted — handler ignores them).

- [ ] **Step 3: Extend `NodeRequest`** — add after `SortOrder int json:"sort_order"`:

```go
	VlessEnabled     *bool  `json:"vless_enabled"`
	VlessPort        int    `json:"vless_port"`
	RealityPubkey    string `json:"reality_pubkey"`
	RealityShortID   string `json:"reality_shortid"`
	RealitySNI       string `json:"reality_sni"`
	VlessStatsAPI    string `json:"vless_stats_api"`
	VlessStatsSecret string `json:"vless_stats_secret"`
	TrojanEnabled    *bool  `json:"trojan_enabled"`
	TrojanPort       int    `json:"trojan_port"`
	TrojanSNI        string `json:"trojan_sni"`
```

- [ ] **Step 4: Extend `CreateNode`** — add to the `model.Node{...}` literal (before `Healthy: true,`):

```go
		VlessEnabled:     req.VlessEnabled != nil && *req.VlessEnabled,
		VlessPort:        req.VlessPort,
		RealityPubkey:    req.RealityPubkey,
		RealityShortID:   req.RealityShortID,
		RealitySNI:       req.RealitySNI,
		VlessStatsAPI:    req.VlessStatsAPI,
		VlessStatsSecret: req.VlessStatsSecret,
		TrojanEnabled:    req.TrojanEnabled != nil && *req.TrojanEnabled,
		TrojanPort:       req.TrojanPort,
		TrojanSNI:        req.TrojanSNI,
```

- [ ] **Step 5: Extend `UpdateNode`** — after the existing `updates := map[string]interface{}{...}` block (and before `tx.Model(&node).Updates(updates)`), add:

```go
	updates["vless_port"] = req.VlessPort
	updates["reality_pubkey"] = req.RealityPubkey
	updates["reality_short_id"] = req.RealityShortID // DB column is reality_short_id
	updates["reality_sni"] = req.RealitySNI
	updates["vless_stats_api"] = req.VlessStatsAPI
	updates["trojan_port"] = req.TrojanPort
	updates["trojan_sni"] = req.TrojanSNI
	if req.VlessEnabled != nil {
		updates["vless_enabled"] = *req.VlessEnabled
	}
	if req.TrojanEnabled != nil {
		updates["trojan_enabled"] = *req.TrojanEnabled
	}
	if req.VlessStatsSecret != "" { // write-only: blank leaves existing unchanged
		updates["vless_stats_secret"] = req.VlessStatsSecret
	}
```

> Note: the `updates` map key for the short-id is the DB column `reality_short_id` (GORM `Updates(map)` uses column names), while the JSON/struct is `RealityShortID`/`reality_shortid`. Other keys match their snake_case columns.

- [ ] **Step 6: Run, expect PASS.**
- [ ] **Step 7: Build** — `go build -o /tmp/hy2board .` → `BUILD_OK`.
- [ ] **Step 8: Commit** — `git add internal/handler/node.go internal/handler/node_update_test.go && git commit -m "feat(nodes): handler accepts vless/trojan/reality fields"`

---

## Task 2: Frontend — REGISTER parser

**Files:** Create `web/src/utils/nodeRegister.ts`

- [ ] **Step 1: Implement the parser** (pure; no test runner exists — verified by build + Task 4 live check)

```ts
// web/src/utils/nodeRegister.ts
// Parses the REGISTER blocks printed by vless-pilot-setup.sh and trojan-add.sh.
// Handles both multi-line "key = value" and inline "key=value" forms.
export interface NodeRegisterFields {
  vless_enabled?: boolean
  vless_port?: number
  reality_pubkey?: string
  reality_shortid?: string
  reality_sni?: string
  vless_stats_api?: string
  vless_stats_secret?: string
  trojan_enabled?: boolean
  trojan_port?: number
  trojan_sni?: string
}

function grab(text: string, key: string): string | undefined {
  // key, optional spaces, =, optional spaces, then a non-space token (quotes stripped)
  const m = text.match(new RegExp(`(?:^|[\\s])${key}\\s*=\\s*"?([^\\s"]+)"?`, "i"))
  return m ? m[1] : undefined
}

export function parseRegister(text: string): NodeRegisterFields {
  const out: NodeRegisterFields = {}
  const str = (k: string, set: (v: string) => void) => { const v = grab(text, k); if (v) set(v) }
  const num = (k: string, set: (v: number) => void) => { const v = grab(text, k); if (v && /^\d+$/.test(v)) set(parseInt(v, 10)) }
  const bool = (k: string, set: (v: boolean) => void) => { const v = grab(text, k); if (v) set(v === "1" || v.toLowerCase() === "true") }

  bool("vless_enabled", v => out.vless_enabled = v)
  num("vless_port", v => out.vless_port = v)
  str("reality_pubkey", v => out.reality_pubkey = v)
  str("reality_shortid", v => out.reality_shortid = v)
  str("reality_sni", v => out.reality_sni = v)
  str("vless_stats_api", v => out.vless_stats_api = v)
  str("vless_stats_secret", v => out.vless_stats_secret = v)
  bool("trojan_enabled", v => out.trojan_enabled = v)
  num("trojan_port", v => out.trojan_port = v)
  str("trojan_sni", v => out.trojan_sni = v)
  return out
}
```

- [ ] **Step 2: Commit** — `git add web/src/utils/nodeRegister.ts && git commit -m "feat(nodes): REGISTER-block parser"`

---

## Task 3: Frontend — Nodes form section + paste

**Files:** Modify `web/src/pages/Nodes.tsx`

- [ ] **Step 1: Import the parser** — add near the top imports:

```ts
import { parseRegister } from "@/utils/nodeRegister"
```

- [ ] **Step 2: Extend `emptyForm`** — add the new fields:

```ts
  vless_enabled: false, vless_port: 443, reality_pubkey: "", reality_shortid: "",
  reality_sni: "www.apple.com", vless_stats_api: "", vless_stats_secret: "",
  trojan_enabled: false, trojan_port: 8443, trojan_sni: "www.apple.com",
```

- [ ] **Step 3: Add a register-paste state** — near the other `useState`s:

```ts
  const [regText, setRegText] = useState("")
  const [showReg, setShowReg] = useState(false)
```

- [ ] **Step 4: Pre-fill on edit** — in `startEdit(n)`'s `setForm({...})`, add (after the existing fields; the write-only secret stays blank):

```ts
      vless_enabled: !!n.vless_enabled, vless_port: n.vless_port || 443,
      reality_pubkey: n.reality_pubkey || "", reality_shortid: n.reality_shortid || "",
      reality_sni: n.reality_sni || "www.apple.com", vless_stats_api: n.vless_stats_api || "",
      vless_stats_secret: "",
      trojan_enabled: !!n.trojan_enabled, trojan_port: n.trojan_port || 8443,
      trojan_sni: n.trojan_sni || "www.apple.com",
```

- [ ] **Step 5: Apply-register handler** — add near the other handlers:

```ts
  const applyRegister = () => {
    const f = parseRegister(regText)
    if (Object.keys(f).length === 0) { showErr("未识别到注册信息"); return }
    setForm(prev => ({ ...prev, ...f }))
    setShowReg(false); setRegText(""); showMsg("已填入注册信息")
  }
```

- [ ] **Step 6: Add the form section** — inside `renderForm`, before the final `<div className="flex gap-2">` (the Save/Cancel buttons), insert:

```tsx
      <div className="border-t border-zinc-800 mt-2 pt-3">
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
              className="mt-2 px-3 py-1.5 bg-white text-black rounded-lg text-sm font-medium hover:bg-zinc-200">填入</button>
          </div>
        )}
        <label className="flex items-center gap-2 text-sm text-zinc-300 mb-2">
          <input type="checkbox" checked={form.vless_enabled}
            onChange={e => setForm({...form, vless_enabled: e.target.checked})} className="w-4 h-4 rounded border-zinc-600 bg-black" />
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
            onChange={e => setForm({...form, trojan_enabled: e.target.checked})} className="w-4 h-4 rounded border-zinc-600 bg-black" />
          启用 Trojan
        </label>
        {form.trojan_enabled && (
          <div className="grid grid-cols-2 gap-3 mb-1 pl-6">
            <div><label className="text-xs text-zinc-500 mb-1 block">Trojan Port</label>
              <input type="number" value={form.trojan_port} onChange={e => setForm({...form, trojan_port: +e.target.value})} className={inputCls + " w-full"} /></div>
            <div><label className="text-xs text-zinc-500 mb-1 block">Trojan SNI</label>
              <input value={form.trojan_sni} onChange={e => setForm({...form, trojan_sni: e.target.value})} className={inputCls + " w-full"} /></div>
          </div>
        )}
      </div>
```

- [ ] **Step 7:** The `add` and `save` handlers already POST/PUT `form` as the body, so the new fields are sent automatically — no change needed. (`vless_stats_secret` blank is ignored by the backend.)

- [ ] **Step 8: Build the frontend** — test-runner `go build` is backend-only; build the web bundle via the Docker image build in Task 4 (which runs `tsc -b && vite build`). No standalone step.

- [ ] **Step 9: Commit** — `git add web/src/pages/Nodes.tsx && git commit -m "feat(nodes): TCP-fallback form section + paste-register"`

---

## Task 4: Build, deploy, verify

- [ ] **Step 1:** scp changed files to `/opt/hy2board`: `internal/handler/node.go`, `internal/handler/node_update_test.go`, `web/src/utils/nodeRegister.ts`, `web/src/pages/Nodes.tsx`.
- [ ] **Step 2:** Backend suite — test-runner `go test ./internal/...` → all `ok`.
- [ ] **Step 3:** Tag rollback + build (build runs `tsc -b && vite build`, catching any TS error) + hot swap:

```bash
cd /opt/hy2board
docker tag "$(docker inspect hy2board-hy2board-1 --format '{{.Image}}')" hy2board-hy2board:rollback-nodeui
docker compose build && docker compose up -d && sleep 8
```

- [ ] **Step 4: Verify** — auth `ok:true`; the built Nodes chunk contains the new code (`docker exec hy2board-hy2board-1 sh -c "grep -l reality-pubkey... "` — or grep the chunk for `vless_enabled`); GET `/admin/nodes` still returns the vless/trojan fields. **Live check (you):** open Nodes → edit a node → the "TCP 回落" section shows current VLESS/Trojan state; paste a REGISTER block → fields fill; Save → re-open shows persisted values (and re-GET shows `vless_enabled` etc.).
- [ ] **Step 5: Commit + push** all changes to GitHub.

---

## Self-review

- **Spec coverage:** backend NodeRequest+Create+Update (T1); write-only secret + bool-false persist (T1 test); parser both formats (T2); form section + toggles + manual fields + paste (T3); pre-fill on edit (T3 S4); deploy/verify (T4). All spec sections mapped.
- **Placeholders:** none — full code in every step.
- **Type consistency:** form field names (`vless_enabled`, `vless_port`, `reality_pubkey`, `reality_shortid`, `reality_sni`, `vless_stats_api`, `vless_stats_secret`, `trojan_enabled`, `trojan_port`, `trojan_sni`) match `NodeRequest` JSON tags (T1) and `parseRegister` output keys (T2) and the form state (T3). UpdateNode map uses DB column `reality_short_id` (noted) while JSON is `reality_shortid` — handled explicitly.
