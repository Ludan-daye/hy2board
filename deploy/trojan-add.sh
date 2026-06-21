#!/usr/bin/env bash
# Add a Trojan inbound (TCP) to an already-VLESS hy2board node. Idempotent.
# Run on the node as root, in the dir holding the updated vless-agent.sh:
#   TROJAN_SNI=www.apple.com bash trojan-add.sh
set -euo pipefail
TROJAN_SNI="${TROJAN_SNI:-www.apple.com}"
TROJAN_PORT="${TROJAN_PORT:-8443}"
CFG="/usr/local/etc/xray/config.json"
CRT="/usr/local/etc/xray/trojan.crt"; KEY="/usr/local/etc/xray/trojan.key"
SELF_DIR="$(cd "$(dirname "$0")" && pwd)"
command -v jq >/dev/null || { apt-get update -y && apt-get install -y jq; }

# 1) self-signed cert (CN = SNI), readable by the 'nobody' xray user
if [ ! -f "$CRT" ]; then
  openssl ecparam -genkey -name prime256v1 -out "$KEY"
  openssl req -new -x509 -days 36500 -key "$KEY" -out "$CRT" -subj "/CN=${TROJAN_SNI}"
fi
chmod 644 "$KEY" "$CRT"

# 2) add/replace the trojan inbound, preserving everything else (reality keys, clients)
tmp="$(jq --arg crt "$CRT" --arg key "$KEY" --argjson port "$TROJAN_PORT" '
  .inbounds = ((.inbounds | map(select(.tag != "trojan"))) + [{
    tag:"trojan", listen:"0.0.0.0", port:$port, protocol:"trojan",
    settings:{clients:[]},
    streamSettings:{network:"tcp", security:"tls",
      tlsSettings:{certificates:[{certificateFile:$crt, keyFile:$key}]}}
  }])' "$CFG")"
echo "$tmp" > "$CFG"

# 3) updated agent (syncs trojan too) + ufw + restart + sync
install -m 0755 "${SELF_DIR}/vless-agent.sh" /usr/local/bin/vless-agent.sh
if command -v ufw >/dev/null && ufw status 2>/dev/null | grep -q "Status: active"; then
  ufw allow "${TROJAN_PORT}/tcp" >/dev/null 2>&1 || true
  ufw reload >/dev/null 2>&1 || true
fi
systemctl restart xray
systemctl start vless-agent-sync.service || true
sleep 2
echo "xray: $(systemctl is-active xray)  trojan port: $(ss -lntp | grep -c ":${TROJAN_PORT} ")"
echo "trojan inbound clients: $(jq '.inbounds[]|select(.tag=="trojan")|.settings.clients|length' "$CFG")"
echo "REGISTER on panel for this node: trojan_enabled=1 trojan_port=${TROJAN_PORT} trojan_sni=${TROJAN_SNI}"
