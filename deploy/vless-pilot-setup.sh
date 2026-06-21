#!/usr/bin/env bash
# One-shot pilot installer: add VLESS+Reality (TCP 443) on this node alongside
# the existing Hysteria2 (UDP 443), wired to the hy2board panel.
#
# Run ON THE NODE (e.g. HK1) as root:
#   NODE_SECRET=xxx STATS_SECRET=yyy bash vless-pilot-setup.sh
#   (NODE_SECRET must equal the panel's config.yaml node.secret)
#
# After it finishes it prints the values to register on the panel's HK1 node row.
set -euo pipefail
: "${NODE_SECRET:?set NODE_SECRET (must match panel node.secret)}"
: "${STATS_SECRET:?set STATS_SECRET (panel will send this to read stats)}"
PANEL_CLIENTS_URL="${PANEL_CLIENTS_URL:-https://vpn.linkbyfree.com/api/node/vless/clients}"
REALITY_DEST="${REALITY_DEST:-www.apple.com:443}"
REALITY_SNI="${REALITY_SNI:-www.apple.com}"
VLESS_PORT="${VLESS_PORT:-443}"
STATS_PORT="${STATS_PORT:-25415}"
XRAY_CONFIG="/usr/local/etc/xray/config.json"
AGENT_DST="/usr/local/bin/vless-agent.sh"
SELF_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "== 1/5 deps =="
command -v jq >/dev/null   || { apt-get update -y && apt-get install -y jq; }
command -v python3 >/dev/null || { apt-get update -y && apt-get install -y python3; }
command -v curl >/dev/null  || { apt-get update -y && apt-get install -y curl; }

echo "== 2/5 install Xray-core =="
if ! command -v xray >/dev/null; then
  bash <(curl -fsSL https://github.com/XTLS/Xray-install/raw/main/install-release.sh) install
fi

echo "== 3/5 reality keypair + shortId =="
KEYS="$(xray x25519)"
PRIV="$(echo "$KEYS" | awk -F': ' '/Private/{print $2}')"
PUB="$(echo "$KEYS"  | awk -F': ' '/Public/{print $2}')"
SID="$(openssl rand -hex 4)"

echo "== 4/5 write xray config (VLESS+Reality on tcp ${VLESS_PORT}; clients filled by agent) =="
mkdir -p "$(dirname "$XRAY_CONFIG")"
cat >"$XRAY_CONFIG" <<JSON
{
  "stats": {},
  "api": { "tag": "api", "services": ["StatsService", "HandlerService"] },
  "policy": { "levels": { "0": { "statsUserUplink": true, "statsUserDownlink": true } } },
  "inbounds": [
    { "tag": "api", "listen": "127.0.0.1", "port": 10085, "protocol": "dokodemo-door",
      "settings": { "address": "127.0.0.1" } },
    { "tag": "vless-reality", "listen": "0.0.0.0", "port": ${VLESS_PORT}, "protocol": "vless",
      "settings": { "clients": [], "decryption": "none" },
      "streamSettings": { "network": "tcp", "security": "reality",
        "realitySettings": { "dest": "${REALITY_DEST}", "serverNames": ["${REALITY_SNI}"],
          "privateKey": "${PRIV}", "shortIds": ["${SID}"] } } }
  ],
  "routing": { "rules": [ { "type": "field", "inboundTag": ["api"], "outboundTag": "api" } ] },
  "outbounds": [ { "protocol": "freedom" } ]
}
JSON

echo "== 5/5 install agent + systemd units =="
install -m 0755 "${SELF_DIR}/vless-agent.sh" "$AGENT_DST"

cat >/etc/systemd/system/vless-agent-sync.service <<UNIT
[Unit]
Description=hy2board VLESS user sync
After=network-online.target xray.service
[Service]
Type=oneshot
Environment=PANEL_CLIENTS_URL=${PANEL_CLIENTS_URL}
Environment=NODE_SECRET=${NODE_SECRET}
ExecStart=${AGENT_DST} sync
UNIT

cat >/etc/systemd/system/vless-agent-sync.timer <<UNIT
[Unit]
Description=hy2board VLESS user sync every 30s
[Timer]
OnBootSec=20
OnUnitActiveSec=30
[Install]
WantedBy=timers.target
UNIT

cat >/etc/systemd/system/vless-agent-stats.service <<UNIT
[Unit]
Description=hy2board VLESS stats endpoint
After=network-online.target xray.service
[Service]
Environment=STATS_SECRET=${STATS_SECRET}
Environment=STATS_PORT=${STATS_PORT}
ExecStart=${AGENT_DST} stats
Restart=always
RestartSec=5
[Install]
WantedBy=multi-user.target
UNIT

systemctl daemon-reload
systemctl enable --now xray.service vless-agent-stats.service vless-agent-sync.timer
systemctl start vless-agent-sync.service || true

PUBIP="$(curl -4 -fsS --max-time 6 ifconfig.me || echo '<THIS_NODE_IP>')"
cat <<DONE

================= VLESS pilot installed on this node =================
OPEN cloud firewall (security group):
  - TCP ${VLESS_PORT}   (VLESS clients)
  - TCP ${STATS_PORT}   FROM THE PANEL IP ONLY (stats)

REGISTER on the panel for the HK1 node (SQL or admin API):
  vless_enabled    = 1
  vless_port       = ${VLESS_PORT}
  reality_pubkey   = ${PUB}
  reality_shortid  = ${SID}
  reality_sni      = ${REALITY_SNI}
  vless_stats_api  = http://${PUBIP}:${STATS_PORT}/
  vless_stats_secret = ${STATS_SECRET}

VERIFY:
  systemctl status xray vless-agent-stats
  ss -lntp | grep -E ':${VLESS_PORT}|:${STATS_PORT}'
  journalctl -u vless-agent-sync -n 20 --no-pager
=====================================================================
DONE
