# Static-IP routing (AI→US / default→Taiwan) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans (or subagent-driven-development). Steps use `- [ ]`.

**Goal:** Gate all chain/static-IP routing behind `user.ChainProxy` (5 chosen users only); for those users route AI→US chain, everything-else→Taiwan chain, China→direct, per-user override; non-chosen users lose the leaked `-AI`/US entries and are otherwise byte-identical.

**Architecture:** Split `EffectiveProxyChain` into `EffectiveAIChain` (global US, gated on `ChainProxy`) + `EffectiveDefaultChain` (per-user `ProxyHost` override else global Taiwan `ProxyChainDefault`, gated on `ChainProxy`). In each rule-capable generator, emit `<node>-AI` (US) and NEW `<node>-TW` (Taiwan) chained proxies for static users; **swap the general node pool used by Auto/Manual/Global/Streaming to the `-TW` names** so all non-AI/non-China traffic exits via Taiwan. Non-static users: no chains, pool = raw nodes (current behavior). URI/base64/plain-shadowrocket untouched.

**Tech Stack:** Go (Gin/GORM/SQLite). DB-free `go test`. Build/deploy via Docker on the server (no local go/docker — same as prior work: rsync `internal/` → `docker run golang:1.26-alpine go test ./internal/...`).

**Test runner:** `sshpass -p 'v4BwlprDUqwXRGVQaHD8' ssh root@76.13.217.10` then `docker run --rm -v /opt/hy2board:/app -v hy2board_gocache:/go -w /app -e CGO_ENABLED=1 golang:1.26-alpine sh -c "apk add --no-cache gcc musl-dev >/dev/null 2>&1 && go test ./internal/service/ -run X -v"`. rsync `internal/` before each run.

---

## File structure

| File | Change |
|---|---|
| `internal/config/config.go` | add `ProxyChainDefault` field + `HasProxyChainDefault()` |
| `internal/service/proxy_resolver.go` | replace `EffectiveProxyChain` → `EffectiveAIChain` + `EffectiveDefaultChain` |
| `internal/service/proxy_resolver_test.go` (new) | resolver unit tests |
| `internal/service/subscription.go` | 3 generators: new resolver calls, `-TW` emission, pool-swap, FINAL |
| `internal/service/subscription_routing_test.go` (new) | rendered-sub assertions |
| `config.yaml` (server, not git) | add `proxy_chain_default` (Taiwan) |
| DB (server) | `chain_proxy` flags + clear the 5's `proxy_host` |

---

## Migration reference (the per-generator change, applied to Surge/Clash/Shadowrocket)

Each generator currently starts:
```go
pc, hasChainConfig := EffectiveProxyChain(user)
rs := BuildRuleSet(user, customRules...)
hasChain := rs.HasAI && hasChainConfig
aiPolicy := GroupAuto
if hasChain { aiPolicy = GroupAI }
```
Becomes:
```go
aiPC, hasAIChain := EffectiveAIChain(user)
defPC, hasDefChain := EffectiveDefaultChain(user)
rs := BuildRuleSet(user, customRules...)
emitAI := rs.HasAI && hasAIChain
aiPolicy := GroupAuto
if emitAI { aiPolicy = GroupAI }
```
In the node loop, the `-AI` block uses `emitAI` and `aiPC` (was `hasChain`/`pc`), and a NEW `-TW` block is added using `hasDefChain`/`defPC`. After the loop, the general pool that feeds Auto/Manual/Global/Streaming is swapped to the `-TW` names for static users. Every `if rs.HasAI && hasChain` group guard becomes `if emitAI`.

---

## Task 1: config — Taiwan default chain

**Files:** `internal/config/config.go`

- [ ] **Step 1:** After the `ProxyChain ProxyChainConfig \`yaml:"proxy_chain"\`` line in `Config` (line 51), add:
```go
	ProxyChainDefault ProxyChainConfig `yaml:"proxy_chain_default"`
```
- [ ] **Step 2:** After `HasProxyChain()` (line 66-68), add:
```go
func (c *Config) HasProxyChainDefault() bool {
	return c.ProxyChainDefault.Host != "" && c.ProxyChainDefault.Port > 0
}
```
- [ ] **Step 3: Commit** — `git add internal/config/config.go && git commit -m "config: add proxy_chain_default (Taiwan) chain"`

---

## Task 2: resolver split (TDD)

**Files:** `internal/service/proxy_resolver.go`; new `internal/service/proxy_resolver_test.go`

- [ ] **Step 1: Failing test** — `internal/service/proxy_resolver_test.go`:
```go
package service

import (
	"testing"

	"github.com/ludandaye/hy2board/internal/config"
	"github.com/ludandaye/hy2board/internal/model"
)

func TestEffectiveChains(t *testing.T) {
	config.C.ProxyChain = config.ProxyChainConfig{Type: "socks5", Host: "1.1.1.1", Port: 8022}          // US/AI
	config.C.ProxyChainDefault = config.ProxyChainConfig{Type: "socks5", Host: "2.2.2.2", Port: 8022}   // Taiwan

	// non-static user: no chains at all
	u := model.User{ChainProxy: false}
	if _, ok := EffectiveAIChain(u); ok {
		t.Fatal("non-static must have no AI chain")
	}
	if _, ok := EffectiveDefaultChain(u); ok {
		t.Fatal("non-static must have no default chain")
	}

	// static user: AI=US global, default=Taiwan global
	s := model.User{ChainProxy: true}
	if ai, ok := EffectiveAIChain(s); !ok || ai.Host != "1.1.1.1" {
		t.Fatalf("static AI want 1.1.1.1 got %v ok=%v", ai.Host, ok)
	}
	if d, ok := EffectiveDefaultChain(s); !ok || d.Host != "2.2.2.2" {
		t.Fatalf("static default want 2.2.2.2 got %v ok=%v", d.Host, ok)
	}

	// static user with per-user ProxyHost override -> default uses the override, AI unchanged
	o := model.User{ChainProxy: true, ProxyType: "socks5", ProxyHost: "9.9.9.9", ProxyPort: 1080}
	if d, ok := EffectiveDefaultChain(o); !ok || d.Host != "9.9.9.9" {
		t.Fatalf("override default want 9.9.9.9 got %v", d.Host)
	}
	if ai, _ := EffectiveAIChain(o); ai.Host != "1.1.1.1" {
		t.Fatal("override must not change AI chain")
	}
}
```
- [ ] **Step 2: Run, expect FAIL** (undefined `EffectiveAIChain`).
- [ ] **Step 3: Implement** — in `internal/service/proxy_resolver.go`, replace the whole `EffectiveProxyChain` func (lines 11-30) with:
```go
// EffectiveAIChain returns the AI-exit chain (global US config.ProxyChain) for a
// static-IP user (ChainProxy == true). Non-static users get no AI chain.
func EffectiveAIChain(u model.User) (config.ProxyChainConfig, bool) {
	if u.ChainProxy && config.C.HasProxyChain() {
		return config.C.ProxyChain, true
	}
	return config.ProxyChainConfig{}, false
}

// EffectiveDefaultChain returns the default (non-AI, non-China) exit chain for a
// static-IP user: per-user ProxyHost override if set, else the global Taiwan
// config.ProxyChainDefault. Non-static users get no default chain.
func EffectiveDefaultChain(u model.User) (config.ProxyChainConfig, bool) {
	if !u.ChainProxy {
		return config.ProxyChainConfig{}, false
	}
	if u.ProxyHost != "" && u.ProxyPort > 0 {
		return config.ProxyChainConfig{
			Type:     u.ProxyType,
			Host:     u.ProxyHost,
			Port:     u.ProxyPort,
			Username: u.ProxyUsername,
			Password: u.ProxyPassword,
		}, true
	}
	if config.C.HasProxyChainDefault() {
		return config.C.ProxyChainDefault, true
	}
	return config.ProxyChainConfig{}, false
}
```
- [ ] **Step 4: Run, expect PASS.** (subscription.go still references `EffectiveProxyChain` — it won't compile yet; run just this test file's package after Task 3, OR temporarily expect compile error here and verify in Task 5. To keep this task green in isolation, proceed to Task 3 before running the whole package.)
- [ ] **Step 5: Commit** — `git add internal/service/proxy_resolver.go internal/service/proxy_resolver_test.go && git commit -m "service: split proxy chain resolver into AI + default (gated on ChainProxy)"`

---

## Task 3: Surge generator

**Files:** `internal/service/subscription.go` — `GenerateSurgeWithCustomRules` (17-93)

- [ ] **Step 1:** Replace the header (lines 19-25) with:
```go
	aiPC, hasAIChain := EffectiveAIChain(user)
	defPC, hasDefChain := EffectiveDefaultChain(user)
	rs := BuildRuleSet(user, customRules...)
	emitAI := rs.HasAI && hasAIChain
	aiPolicy := GroupAuto
	if emitAI {
		aiPolicy = GroupAI
	}
```
- [ ] **Step 2:** In the `[Proxy]` block, add a `twNames` slice next to `chainNames` (line 41-42):
```go
	chainNames := make([]string, 0)
	twNames := make([]string, 0)
	vlessNames := make([]string, 0)
```
- [ ] **Step 3:** In the node loop, replace the `-AI` block (lines 58-64) with `-AI` (using `emitAI`/`aiPC`) + new `-TW`:
```go
		if emitAI {
			chainName := n.Name + "-AI"
			lines = append(lines, fmt.Sprintf("%s = %s, %s, %d, username=%s, password=%s, underlying-proxy=%s",
				chainName, aiPC.Type, aiPC.Host, aiPC.Port, aiPC.Username, aiPC.Password, n.Name))
			chainNames = append(chainNames, chainName)
		}
		if hasDefChain {
			twName := n.Name + "-TW"
			lines = append(lines, fmt.Sprintf("%s = %s, %s, %d, username=%s, password=%s, underlying-proxy=%s",
				twName, defPC.Type, defPC.Host, defPC.Port, defPC.Username, defPC.Password, n.Name))
			twNames = append(twNames, twName)
		}
```
- [ ] **Step 4:** After the loop, swap the general pool to Taiwan for static users. Replace line 68 (`names = append(names, vlessNames...)`) with:
```go
	pool := names
	if hasDefChain && len(twNames) > 0 {
		pool = twNames
	}
	pool = append(pool, vlessNames...)
```
- [ ] **Step 5:** In the `[Proxy Group]` block (lines 70-83), replace every `strings.Join(names, ...)` with `strings.Join(pool, ...)` for `GroupAuto`, `GroupManual`, `GroupStreaming`, `GroupGlobal`; change the AI-group guard `if rs.HasAI && hasChain` to `if emitAI` (keep `chainNames`); leave `GroupHongKong` on `hongKongNames`:
```go
	lines = append(lines, fmt.Sprintf("%s = url-test, %s, url=%s, interval=300", GroupAuto, strings.Join(pool, ", "), proxyTestURL))
	lines = append(lines, fmt.Sprintf("%s = select, %s", GroupManual, strings.Join(pool, ", ")))
	if len(hongKongNames) > 0 {
		lines = append(lines, fmt.Sprintf("%s = url-test, %s, url=%s, interval=300", GroupHongKong, strings.Join(hongKongNames, ", "), proxyTestURL))
	}
	if emitAI {
		lines = append(lines, fmt.Sprintf("%s = url-test, %s, url=%s, interval=300", GroupAI, strings.Join(chainNames, ", "), proxyTestURL))
	}
	if rs.HasStreaming {
		lines = append(lines, fmt.Sprintf("%s = url-test, %s, url=%s, interval=300", GroupStreaming, strings.Join(pool, ", "), proxyTestURL))
	}
	if rs.HasGlobal {
		lines = append(lines, fmt.Sprintf("%s = url-test, %s, url=%s, interval=300", GroupGlobal, strings.Join(pool, ", "), proxyTestURL))
	}
```
(FINAL stays `"FINAL,"+GroupAuto` — `Auto` now = Taiwan for static users because `pool` is swapped.)
- [ ] **Step 6: Commit** — `git commit -am "sub(surge): static users route default->Taiwan pool, AI->US, gated on ChainProxy"` (⚠ use explicit `git add internal/service/subscription.go` to avoid the stray subscription.go/other files — actually only this file changed here).

---

## Task 4: Clash generator

**Files:** `internal/service/subscription.go` — `GenerateClashWithCustomRules` (99-230)

- [ ] **Step 1:** Header replace (101-107) — same as Surge Step 1 (`aiPC/hasAIChain`, `defPC/hasDefChain`, `emitAI`, `aiPolicy`).
- [ ] **Step 2:** Add `twNames := make([]string, 0)` next to `chainNames` (line 122-123).
- [ ] **Step 3:** In the node loop replace the `-AI` YAML block (145-156) with `emitAI`/`aiPC` version + new `-TW`:
```go
		if emitAI {
			chainName := n.Name + "-AI"
			lines = append(lines, fmt.Sprintf("  - name: \"%s\"", chainName))
			lines = append(lines, fmt.Sprintf("    type: %s", aiPC.Type))
			lines = append(lines, fmt.Sprintf("    server: %s", aiPC.Host))
			lines = append(lines, fmt.Sprintf("    port: %d", aiPC.Port))
			lines = append(lines, fmt.Sprintf("    username: %s", aiPC.Username))
			lines = append(lines, fmt.Sprintf("    password: %s", aiPC.Password))
			lines = append(lines, fmt.Sprintf("    dialer-proxy: %s", n.Name))
			lines = append(lines, "")
			chainNames = append(chainNames, chainName)
		}
		if hasDefChain {
			twName := n.Name + "-TW"
			lines = append(lines, fmt.Sprintf("  - name: \"%s\"", twName))
			lines = append(lines, fmt.Sprintf("    type: %s", defPC.Type))
			lines = append(lines, fmt.Sprintf("    server: %s", defPC.Host))
			lines = append(lines, fmt.Sprintf("    port: %d", defPC.Port))
			lines = append(lines, fmt.Sprintf("    username: %s", defPC.Username))
			lines = append(lines, fmt.Sprintf("    password: %s", defPC.Password))
			lines = append(lines, fmt.Sprintf("    dialer-proxy: %s", n.Name))
			lines = append(lines, "")
			twNames = append(twNames, twName)
		}
```
- [ ] **Step 4:** Replace line 159 (`names = append(names, vlessNames...)`) with the `pool` computation (same as Surge Step 4).
- [ ] **Step 5:** In `proxy-groups`, replace the `for _, name := range names` loops for `GroupAuto` (164-166), `GroupManual` (173-175), `GroupStreaming` (203-205), `GroupGlobal` (213-215) to iterate `pool`; change the AI-group guard `if rs.HasAI && hasChain` (188) to `if emitAI`. Leave `GroupHongKong` on `hongKongNames`. (MATCH stays `"  - MATCH,"+GroupAuto`.)
- [ ] **Step 6: Commit** — `git add internal/service/subscription.go && git commit -m "sub(clash): static users default->Taiwan pool, AI->US"`

---

## Task 5: Shadowrocket generator

**Files:** `internal/service/subscription.go` — `GenerateShadowrocketWithCustomRules` (244-311)

- [ ] **Step 1:** Header replace (246-252) — same as Surge Step 1.
- [ ] **Step 2:** Add `twNames := make([]string, 0)` (line 268-269).
- [ ] **Step 3:** Replace the `-AI` block (276-282) with `emitAI`/`aiPC` + new `-TW` (identical `underlying-proxy` format string as Surge Step 3).
- [ ] **Step 4:** Replace line 286 (`names = append(names, vlessNames...)`) with the `pool` computation.
- [ ] **Step 5:** In `[Proxy Group]` (288-300), swap `names`→`pool` for Auto/Manual/Streaming/Global; AI guard → `if emitAI`. HongKong on `hongKongNames`. FINAL stays `GroupAuto`.
- [ ] **Step 6: Build the whole package** — rsync + `go build ./internal/... ; echo EXIT $?` (must be 0 — confirms `EffectiveProxyChain` has no remaining callers and everything compiles).
- [ ] **Step 7: Commit** — `git add internal/service/subscription.go && git commit -m "sub(shadowrocket): static users default->Taiwan pool, AI->US"`

---

## Task 6: rendered-subscription test

**Files:** new `internal/service/subscription_routing_test.go`

- [ ] **Step 1: Test** (asserts the leak fix + Taiwan pool). Then run (expect PASS after Tasks 3-5):
```go
package service

import (
	"strings"
	"testing"

	"github.com/ludandaye/hy2board/internal/config"
	"github.com/ludandaye/hy2board/internal/model"
)

func routingNodes() []model.Node {
	return []model.Node{{Name: "SG1", Host: "9.9.9.9", Port: 443, SNI: "bing.com", Insecure: true}}
}

func TestStaticVsNonStaticRouting(t *testing.T) {
	config.C.ProxyChain = config.ProxyChainConfig{Type: "socks5", Host: "1.1.1.1", Port: 8022, Username: "us", Password: "up"}
	config.C.ProxyChainDefault = config.ProxyChainConfig{Type: "socks5", Host: "2.2.2.2", Port: 8022, Username: "tw", Password: "tp"}

	// non-static user with rule_ai ON must get NO -AI leak, no -TW
	non := model.User{Hy2Password: "p", RuleAI: true, RuleChina: true, ChainProxy: false}
	c := GenerateClash(non, routingNodes())
	if strings.Contains(c, "SG1-AI") || strings.Contains(c, "1.1.1.1") {
		t.Fatal("non-static user leaked the US -AI proxy")
	}
	if strings.Contains(c, "SG1-TW") {
		t.Fatal("non-static user should have no -TW")
	}

	// static user: US -AI present + Taiwan -TW present + Auto over -TW
	st := model.User{Hy2Password: "p", RuleAI: true, RuleChina: true, ChainProxy: true}
	c2 := GenerateClash(st, routingNodes())
	if !strings.Contains(c2, "SG1-AI") || !strings.Contains(c2, "1.1.1.1") {
		t.Fatal("static user missing US AI chain")
	}
	if !strings.Contains(c2, "SG1-TW") || !strings.Contains(c2, "2.2.2.2") {
		t.Fatal("static user missing Taiwan default chain")
	}
	// Auto group should list the -TW node (Taiwan pool), not raw SG1
	autoIdx := strings.Index(c2, "name: "+GroupAuto)
	seg := c2[autoIdx : autoIdx+200]
	if !strings.Contains(seg, "SG1-TW") {
		t.Fatalf("Auto group should use Taiwan pool: %s", seg)
	}
}
```
- [ ] **Step 2: Run** `go test ./internal/service/ -run 'TestEffectiveChains|TestStaticVsNonStaticRouting' -v` → PASS. Then full `go test ./internal/...` → all `ok`.
- [ ] **Step 3: Commit** — `git add internal/service/subscription_routing_test.go && git commit -m "test: static vs non-static subscription routing"`

---

## Task 7: build, deploy, BYTE-DIFF GATE (code only, Taiwan not yet active)

- [ ] **Step 1:** Capture BEFORE subs for a non-chosen (`yyz`) and a chosen (`sjb`) user — Clash + Surge — to `/tmp/before_*.txt` (fetch `http://localhost:9000/api/sub/<token>` with UA clash/surge on the server; get tokens from DB).
- [ ] **Step 2:** rsync `internal/` → tag `rollback-prerouting` → `docker compose build && up -d`. Build must be clean.
- [ ] **Step 3:** Capture AFTER subs for the same two users.
- [ ] **Step 4: Verify (you + me):**
  - `diff before_yyz after_yyz` (non-chosen): the ONLY changes are removed `SG1-AI`-type lines + removed `AI` group + AI rules now → `Auto`. **No other differences.** (yyz is `chain_proxy=false` today, so this proves the leak is gone and nothing else moved.)
  - `sjb` (chosen, `chain_proxy=true`, still has `proxy_host`=US today): still shows `-AI` (US); Taiwan not present yet (config not set) → default still `Auto`(direct). Confirms static path intact.
- [ ] **Step 5:** If diff is clean → proceed. If anything unexpected → `docker tag rollback-prerouting → latest; up -d` and stop. **Commit + push** the code: `git push origin HEAD:main`.

---

## Task 8: enable Taiwan + set the chosen set, verify

- [ ] **Step 1:** Edit `config.yaml` on the server: add
```yaml
proxy_chain_default:
  host: "202.160.86.52"
  port: 8022
  username: "scdrawqaee"
  password: "ipimorpmwrmti"
  type: "socks5"
```
- [ ] **Step 2:** DB data (via admin API PUT `/api/admin/users/:id`, token from `ludandaye`/`8011`): set `chain_proxy=true` for `ludandaye, sjb, qmf, yzp, xrx`; `chain_proxy=false` for `a100` (and any other user currently true); for the 5 chosen set `proxy_host=""` `proxy_port=0` (clear the old US override so default resolves to global Taiwan). Restart container so config reloads.
- [ ] **Step 3: Verify (you + me):** fetch `sjb` (chosen) Clash — must now contain `SG1-TW` + `202.160.86.52`, `Auto` group over `-TW`, AI group over `-AI` (`199.234.95.128`), China rules → DIRECT. Fetch `yyz` (non-chosen) — unchanged from Task 7 (no `-AI`, no `-TW`, `Auto` over raw nodes). **Live client check**: `sjb` on Clash — AI site shows US exit, a normal site shows Taiwan exit, a CN site direct.
- [ ] **Step 4:** Update memory (`hy2board-panel-deployment.md`): static-IP routing live (chain_proxy = static flag; US=AI, Taiwan=default; 5 chosen; proxy_chain_default in config; rollback-prerouting).

---

## Self-review

- **Spec coverage:** two chains (Task 1 + 8), resolver split gated on ChainProxy (Task 2), AI→US / default→Taiwan / China direct via pool-swap (Tasks 3-5), leak removed for non-chosen (Task 6 test + Task 7 byte-diff), chosen set + a100 removed + Taiwan config (Task 8), per-user override (Task 2 test + resolver), URI/base64 untouched (not in scope). ✓
- **Placeholders:** every code step has full code; the 3 generators are given explicitly (not "same as"); byte-diff + live-exit checks are concrete. ✓
- **Naming:** `EffectiveAIChain`/`EffectiveDefaultChain`, `emitAI`, `aiPC`/`defPC`, `twNames`/`pool`, `HasProxyChainDefault`, `ProxyChainDefault`, `GroupAuto`/`GroupAI` consistent across tasks. Old `EffectiveProxyChain` fully removed (Task 5 Step 6 build check catches stragglers). ✓
- **Risk:** the byte-diff gate (Task 7) on a real non-chosen user is the safety net — code ships with Taiwan inactive, so the only live change is the leak removal; Taiwan/pool-swap only activates after Task 8 data, re-verified. Rollback image at each stage.
