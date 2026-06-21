#!/usr/bin/env bash
# vless-agent: keep this node's Xray VLESS users in sync with the panel, and
# expose cumulative per-user traffic for the panel to poll.
#
# Modes:
#   vless-agent.sh sync   -> pull active users from panel, apply to Xray (restart on change)
#   vless-agent.sh stats  -> run an authenticated HTTP server returning {email:{tx,rx}}
#
# Env (set by systemd units, see vless-pilot-setup.sh):
#   PANEL_CLIENTS_URL  e.g. https://vpn.linkbyfree.com/api/node/vless/clients
#   NODE_SECRET        shared node->panel secret (must match panel config node.secret)
#   STATS_SECRET       secret the panel sends to read this node's stats
#   STATS_PORT         tcp port for the stats server (default 25415)
#   XRAY_CONFIG        path to xray config.json (default /usr/local/etc/xray/config.json)
#   XRAY_API           xray gRPC api address (default 127.0.0.1:10085)
set -euo pipefail
XRAY_CONFIG="${XRAY_CONFIG:-/usr/local/etc/xray/config.json}"
XRAY_API="${XRAY_API:-127.0.0.1:10085}"
STATS_PORT="${STATS_PORT:-25415}"

sync_users() {
  : "${PANEL_CLIENTS_URL:?}" "${NODE_SECRET:?}"
  local clients new
  # fail-open: on any panel/network error keep the current user set (never wipe)
  clients="$(curl -fsS --max-time 10 -H "Authorization: ${NODE_SECRET}" "$PANEL_CLIENTS_URL")" || {
    echo "[vless-agent] panel unreachable; keeping current users"; return 0; }
  echo "$clients" | jq -e 'type=="array"' >/dev/null 2>&1 || {
    echo "[vless-agent] bad client list; skipping"; return 0; }

  # Apply users to the vless-reality inbound (uuid) and the trojan inbound
  # (password). The trojan selector is a no-op on nodes without a trojan inbound.
  new="$(jq --argjson c "$clients" '
    (.inbounds[] | select(.tag=="vless-reality") | .settings.clients) =
      ($c | map({id:.uuid, email:.email, flow:"xtls-rprx-vision"})) |
    (.inbounds[] | select(.tag=="trojan") | .settings.clients) =
      ($c | map({password:.password, email:.email}))
  ' "$XRAY_CONFIG")"

  if diff -q <(jq -S . "$XRAY_CONFIG") <(echo "$new" | jq -S .) >/dev/null 2>&1; then
    return 0  # unchanged
  fi
  echo "$new" > "$XRAY_CONFIG"
  systemctl restart xray
  echo "[vless-agent] user set changed -> xray restarted ($(echo "$clients" | jq length) users)"
}

# Emit current cumulative stats as {email:{tx,rx}} (used by the stats server).
emit_stats_json() {
  xray api statsquery --server="$XRAY_API" 2>/dev/null | python3 -c '
import sys, json, re
try:
    data = json.load(sys.stdin)
except Exception:
    print("{}"); sys.exit(0)
out = {}
for s in data.get("stat", []) or []:
    m = re.match(r"user>>>(.+)>>>traffic>>>(uplink|downlink)$", s.get("name",""))
    if not m: continue
    email, direction = m.group(1), m.group(2)
    v = int(s.get("value", 0) or 0)
    out.setdefault(email, {"tx":0,"rx":0})
    out[email]["tx" if direction=="uplink" else "rx"] += v
print(json.dumps(out))
'
}

serve_stats() {
  : "${STATS_SECRET:?}"
  STATS_SECRET="$STATS_SECRET" STATS_PORT="$STATS_PORT" python3 - "$0" <<'PY'
import os, subprocess, json
from http.server import BaseHTTPRequestHandler, HTTPServer
SECRET = os.environ["STATS_SECRET"]; PORT = int(os.environ["STATS_PORT"]); AGENT = __import__("sys").argv[1]
class H(BaseHTTPRequestHandler):
    def log_message(self, *a): pass
    def do_GET(self):
        if self.headers.get("Authorization","") != SECRET:
            self.send_response(401); self.end_headers(); self.wfile.write(b"unauthorized"); return
        try:
            body = subprocess.check_output(["bash", AGENT, "emit-stats"], timeout=8)
        except Exception:
            body = b"{}"
        self.send_response(200); self.send_header("Content-Type","application/json"); self.end_headers()
        self.wfile.write(body)
HTTPServer(("0.0.0.0", PORT), H).serve_forever()
PY
}

case "${1:-sync}" in
  sync)        sync_users ;;
  stats)       serve_stats ;;
  emit-stats)  emit_stats_json ;;
  *) echo "usage: $0 {sync|stats}"; exit 2 ;;
esac
