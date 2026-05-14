#!/usr/bin/env bash
set -Eeuo pipefail

VERSION="1.0.0"

HY2_AUTH_URL="${HY2_AUTH_URL:-https://vpn.linkbyfree.com/api/auth/hy2}"
HY2_SNI="${HY2_SNI:-bing.com}"
HY2_MASQUERADE_URL="${HY2_MASQUERADE_URL:-https://www.bing.com}"
HY2_NODE_PREFIX="${HY2_NODE_PREFIX:-NODE}"
HY2_PUBLIC_IP="${HY2_PUBLIC_IP:-}"

HY2_PLAIN_PORT="${HY2_PLAIN_PORT:-443}"
HY2_OBFS_PORT="${HY2_OBFS_PORT:-8443}"
HY2_PLAIN_TRAFFIC_PORT="${HY2_PLAIN_TRAFFIC_PORT:-25413}"
HY2_OBFS_TRAFFIC_PORT="${HY2_OBFS_TRAFFIC_PORT:-25414}"
HY2_SORT_BASE="${HY2_SORT_BASE:-10}"

HY2_CONFIG_DIR="${HY2_CONFIG_DIR:-/etc/hysteria}"
HY2_PLAIN_CONFIG="${HY2_CONFIG_DIR}/config.yaml"
HY2_OBFS_CONFIG="${HY2_CONFIG_DIR}/config-obfs.yaml"
HY2_CERT_FILE="${HY2_CONFIG_DIR}/server.crt"
HY2_KEY_FILE="${HY2_CONFIG_DIR}/server.key"

HY2_PLAIN_SECRET="${HY2_PLAIN_SECRET:-}"
HY2_OBFS_SECRET="${HY2_OBFS_SECRET:-}"
HY2_OBFS_PASSWORD="${HY2_OBFS_PASSWORD:-}"

HY2_DRY_RUN="${HY2_DRY_RUN:-0}"
HY2_SKIP_INSTALL="${HY2_SKIP_INSTALL:-0}"
HY2_FORCE="${HY2_FORCE:-0}"
HY2_APPLY_UFW="${HY2_APPLY_UFW:-1}"
HY2_OUTPUT_FILE="${HY2_OUTPUT_FILE:-}"

BACKUP_TS="$(date +%Y%m%d%H%M%S)"

usage() {
  cat <<EOF
HY2 dual-node deploy ${VERSION}

Usage:
  sudo bash deploy-hy2-dual-node.sh [options]

Options:
  --prefix NAME          Node name prefix, e.g. JP4 -> JP4-plain / JP4-obfs
  --auth-url URL         hy2board auth URL. Default: ${HY2_AUTH_URL}
  --sni NAME             TLS SNI and self-signed certificate CN. Default: ${HY2_SNI}
  --public-ip IP         Public IP/domain to print in hy2board node info
  --sort-base N          plain sort order; obfs uses N+1. Default: ${HY2_SORT_BASE}
  --plain-port N         HY2 plain UDP port. Default: ${HY2_PLAIN_PORT}
  --obfs-port N          HY2 obfs UDP port. Default: ${HY2_OBFS_PORT}
  --plain-traffic N      plain trafficStats TCP port. Default: ${HY2_PLAIN_TRAFFIC_PORT}
  --obfs-traffic N       obfs trafficStats TCP port. Default: ${HY2_OBFS_TRAFFIC_PORT}
  --force                Backup and overwrite existing configs
  --skip-install         Do not run the official Hysteria installer
  --no-ufw               Do not add UFW allow rules
  --dry-run              Print actions and node info without changing the server
  -h, --help             Show this help

Environment overrides:
  HY2_PLAIN_SECRET       Reuse a plain traffic secret instead of generating one
  HY2_OBFS_SECRET        Reuse an obfs traffic secret instead of generating one
  HY2_OBFS_PASSWORD      Reuse a salamander password instead of generating one
  HY2_MASQUERADE_URL     Masquerade proxy URL. Default: ${HY2_MASQUERADE_URL}

Example:
  sudo HY2_NODE_PREFIX=JP4 bash deploy-hy2-dual-node.sh
EOF
}

log() {
  printf '[INFO] %s\n' "$*"
}

warn() {
  printf '[WARN] %s\n' "$*" >&2
}

fail() {
  printf '[ERROR] %s\n' "$*" >&2
  exit 1
}

run() {
  if [[ "${HY2_DRY_RUN}" == "1" ]]; then
    printf '[dry-run] %q' "$1"
    shift || true
    for arg in "$@"; do
      printf ' %q' "${arg}"
    done
    printf '\n'
    return 0
  fi
  "$@"
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --prefix)
        HY2_NODE_PREFIX="${2:-}"
        shift 2
        ;;
      --auth-url)
        HY2_AUTH_URL="${2:-}"
        shift 2
        ;;
      --sni)
        HY2_SNI="${2:-}"
        shift 2
        ;;
      --public-ip)
        HY2_PUBLIC_IP="${2:-}"
        shift 2
        ;;
      --sort-base)
        HY2_SORT_BASE="${2:-}"
        shift 2
        ;;
      --plain-port)
        HY2_PLAIN_PORT="${2:-}"
        shift 2
        ;;
      --obfs-port)
        HY2_OBFS_PORT="${2:-}"
        shift 2
        ;;
      --plain-traffic)
        HY2_PLAIN_TRAFFIC_PORT="${2:-}"
        shift 2
        ;;
      --obfs-traffic)
        HY2_OBFS_TRAFFIC_PORT="${2:-}"
        shift 2
        ;;
      --force)
        HY2_FORCE=1
        shift
        ;;
      --skip-install)
        HY2_SKIP_INSTALL=1
        shift
        ;;
      --no-ufw)
        HY2_APPLY_UFW=0
        shift
        ;;
      --dry-run)
        HY2_DRY_RUN=1
        shift
        ;;
      -h|--help)
        usage
        exit 0
        ;;
      *)
        fail "unknown option: $1"
        ;;
    esac
  done
}

require_value() {
  local name="$1"
  local value="$2"
  [[ -n "${value}" ]] || fail "${name} cannot be empty"
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "missing required command: $1"
}

validate_number() {
  local name="$1"
  local value="$2"
  [[ "${value}" =~ ^[0-9]+$ ]] || fail "${name} must be a number: ${value}"
}

preflight() {
  log "preflight checks"
  require_value "HY2_AUTH_URL" "${HY2_AUTH_URL}"
  require_value "HY2_SNI" "${HY2_SNI}"
  require_value "HY2_NODE_PREFIX" "${HY2_NODE_PREFIX}"
  validate_number "HY2_PLAIN_PORT" "${HY2_PLAIN_PORT}"
  validate_number "HY2_OBFS_PORT" "${HY2_OBFS_PORT}"
  validate_number "HY2_PLAIN_TRAFFIC_PORT" "${HY2_PLAIN_TRAFFIC_PORT}"
  validate_number "HY2_OBFS_TRAFFIC_PORT" "${HY2_OBFS_TRAFFIC_PORT}"
  validate_number "HY2_SORT_BASE" "${HY2_SORT_BASE}"

  if [[ "${HY2_DRY_RUN}" != "1" && "${EUID}" -ne 0 ]]; then
    fail "must run as root. Use: sudo bash $0"
  fi

  [[ "$(uname -s)" == "Linux" || "${HY2_DRY_RUN}" == "1" ]] || fail "Linux is required"
  need_cmd bash
  need_cmd curl
  need_cmd openssl
  need_cmd awk
  need_cmd sed
  need_cmd date

  if [[ "${HY2_DRY_RUN}" != "1" ]]; then
    need_cmd systemctl
    need_cmd ss
    need_cmd install
  fi

  case "$(uname -m)" in
    x86_64|amd64|aarch64|arm64) ;;
    *)
      if [[ "${HY2_DRY_RUN}" != "1" ]]; then
        fail "unsupported architecture: $(uname -m)"
      fi
      ;;
  esac
}

port_in_use() {
  local port="$1"
  ss -H -lntu 2>/dev/null | awk '{print $5}' | grep -Eq "[:.]${port}$"
}

check_ports() {
  [[ "${HY2_DRY_RUN}" == "1" ]] && return 0
  local ports=("${HY2_PLAIN_PORT}" "${HY2_OBFS_PORT}" "${HY2_PLAIN_TRAFFIC_PORT}" "${HY2_OBFS_TRAFFIC_PORT}")
  for port in "${ports[@]}"; do
    if port_in_use "${port}"; then
      if [[ "${HY2_FORCE}" == "1" ]]; then
        warn "port ${port} is already in use; continuing because --force is set"
      else
        fail "port ${port} is already in use. Stop the old service or rerun with --force."
      fi
    fi
  done
}

detect_public_ip() {
  if [[ -n "${HY2_PUBLIC_IP}" ]]; then
    return 0
  fi
  if [[ "${HY2_DRY_RUN}" == "1" ]]; then
    HY2_PUBLIC_IP="203.0.113.10"
    return 0
  fi
  HY2_PUBLIC_IP="$(
    curl -4 -fsS --max-time 6 https://ifconfig.me 2>/dev/null ||
      curl -4 -fsS --max-time 6 https://api.ipify.org 2>/dev/null ||
      true
  )"
  if [[ -z "${HY2_PUBLIC_IP}" ]]; then
    warn "could not detect public IPv4; output will use <PUBLIC_IP>"
    HY2_PUBLIC_IP="<PUBLIC_IP>"
  fi
}

check_auth_url() {
  [[ "${HY2_DRY_RUN}" == "1" ]] && return 0
  local status
  status="$(curl -k -sS -o /dev/null -w '%{http_code}' --max-time 8 "${HY2_AUTH_URL}" || true)"
  if [[ "${status}" == "000" ]]; then
    warn "hy2board auth URL is not reachable from this server: ${HY2_AUTH_URL}"
  else
    log "hy2board auth URL reachable, HTTP ${status}"
  fi
}

rand_hex_16() {
  openssl rand -hex 16
}

prepare_secrets() {
  [[ -n "${HY2_PLAIN_SECRET}" ]] || HY2_PLAIN_SECRET="$(rand_hex_16)"
  [[ -n "${HY2_OBFS_SECRET}" ]] || HY2_OBFS_SECRET="$(rand_hex_16)"
  [[ -n "${HY2_OBFS_PASSWORD}" ]] || HY2_OBFS_PASSWORD="$(rand_hex_16)"
}

install_hysteria() {
  if command -v hysteria >/dev/null 2>&1; then
    log "hysteria exists: $(command -v hysteria)"
    hysteria version 2>/dev/null | sed -n '1,14p' || true
    return 0
  fi
  if [[ "${HY2_SKIP_INSTALL}" == "1" ]]; then
    fail "hysteria is not installed and --skip-install is set"
  fi
  log "installing hysteria via official installer"
  if [[ "${HY2_DRY_RUN}" == "1" ]]; then
    echo "[dry-run] bash <(curl -fsSL https://get.hy2.sh/)"
  else
    bash <(curl -fsSL https://get.hy2.sh/)
  fi
}

ensure_hysteria_user() {
  [[ "${HY2_DRY_RUN}" == "1" ]] && return 0
  if ! id hysteria >/dev/null 2>&1; then
    useradd --system --no-create-home --shell /usr/sbin/nologin hysteria
  fi
}

backup_if_exists() {
  local path="$1"
  [[ -e "${path}" ]] || return 0
  local backup="${path}.bak-${BACKUP_TS}"
  if [[ "${HY2_FORCE}" != "1" ]]; then
    fail "${path} already exists. Use --force to backup and overwrite."
  fi
  run cp -a "${path}" "${backup}"
  log "backed up ${path} -> ${backup}"
}

write_cert() {
  log "preparing TLS certificate"
  if [[ -e "${HY2_KEY_FILE}" || -e "${HY2_CERT_FILE}" ]]; then
    if [[ "${HY2_FORCE}" == "1" ]]; then
      backup_if_exists "${HY2_KEY_FILE}"
      backup_if_exists "${HY2_CERT_FILE}"
    else
      log "reusing existing certificate files"
      return 0
    fi
  fi

  run mkdir -p "${HY2_CONFIG_DIR}"
  if [[ "${HY2_DRY_RUN}" == "1" ]]; then
    echo "[dry-run] openssl ecparam -genkey -name prime256v1 -out ${HY2_KEY_FILE}"
    echo "[dry-run] openssl req -new -x509 -days 36500 -key ${HY2_KEY_FILE} -out ${HY2_CERT_FILE} -subj /CN=${HY2_SNI}"
    return 0
  fi
  openssl ecparam -genkey -name prime256v1 -out "${HY2_KEY_FILE}"
  openssl req -new -x509 -days 36500 -key "${HY2_KEY_FILE}" -out "${HY2_CERT_FILE}" -subj "/CN=${HY2_SNI}"
}

write_systemd_units() {
  [[ "${HY2_DRY_RUN}" == "1" ]] && return 0
  if [[ ! -f /etc/systemd/system/hysteria-server.service ]]; then
    cat >/etc/systemd/system/hysteria-server.service <<'EOF'
[Unit]
Description=Hysteria Server Service (config.yaml)
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/hysteria server --config /etc/hysteria/config.yaml
WorkingDirectory=~
User=hysteria
Group=hysteria
Environment=HYSTERIA_LOG_LEVEL=info
CapabilityBoundingSet=CAP_NET_ADMIN CAP_NET_BIND_SERVICE CAP_NET_RAW
AmbientCapabilities=CAP_NET_ADMIN CAP_NET_BIND_SERVICE CAP_NET_RAW
NoNewPrivileges=true
Restart=on-failure
RestartSec=5
LimitNOFILE=infinity

[Install]
WantedBy=multi-user.target
EOF
  fi
  if [[ ! -f /etc/systemd/system/hysteria-server@.service ]]; then
    cat >/etc/systemd/system/hysteria-server@.service <<'EOF'
[Unit]
Description=Hysteria Server Service (%i.yaml)
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/hysteria server --config /etc/hysteria/%i.yaml
WorkingDirectory=~
User=hysteria
Group=hysteria
Environment=HYSTERIA_LOG_LEVEL=info
CapabilityBoundingSet=CAP_NET_ADMIN CAP_NET_BIND_SERVICE CAP_NET_RAW
AmbientCapabilities=CAP_NET_ADMIN CAP_NET_BIND_SERVICE CAP_NET_RAW
NoNewPrivileges=true
Restart=on-failure
RestartSec=5
LimitNOFILE=infinity

[Install]
WantedBy=multi-user.target
EOF
  fi
}

write_configs() {
  log "writing hysteria server configs"
  backup_if_exists "${HY2_PLAIN_CONFIG}"
  backup_if_exists "${HY2_OBFS_CONFIG}"
  run mkdir -p "${HY2_CONFIG_DIR}"

  if [[ "${HY2_DRY_RUN}" == "1" ]]; then
    echo "[dry-run] write ${HY2_PLAIN_CONFIG}"
    echo "[dry-run] write ${HY2_OBFS_CONFIG}"
    return 0
  fi

  cat >"${HY2_PLAIN_CONFIG}" <<EOF
listen: :${HY2_PLAIN_PORT}

tls:
  cert: ${HY2_CERT_FILE}
  key: ${HY2_KEY_FILE}

auth:
  type: http
  http:
    url: ${HY2_AUTH_URL}
    insecure: false

trafficStats:
  listen: :${HY2_PLAIN_TRAFFIC_PORT}
  secret: ${HY2_PLAIN_SECRET}

masquerade:
  type: proxy
  proxy:
    url: ${HY2_MASQUERADE_URL}
    rewriteHost: true
EOF

  cat >"${HY2_OBFS_CONFIG}" <<EOF
listen: :${HY2_OBFS_PORT}

obfs:
  type: salamander
  salamander:
    password: ${HY2_OBFS_PASSWORD}

tls:
  cert: ${HY2_CERT_FILE}
  key: ${HY2_KEY_FILE}

auth:
  type: http
  http:
    url: ${HY2_AUTH_URL}
    insecure: false

trafficStats:
  listen: :${HY2_OBFS_TRAFFIC_PORT}
  secret: ${HY2_OBFS_SECRET}

masquerade:
  type: proxy
  proxy:
    url: ${HY2_MASQUERADE_URL}
    rewriteHost: true
EOF
}

fix_permissions() {
  log "fixing permissions"
  run chown -R hysteria:hysteria "${HY2_CONFIG_DIR}"
  run chmod 750 "${HY2_CONFIG_DIR}"
  run chmod 600 "${HY2_KEY_FILE}"
  run chmod 644 "${HY2_CERT_FILE}"
  run chmod 644 "${HY2_PLAIN_CONFIG}" "${HY2_OBFS_CONFIG}"
}

configure_ufw() {
  [[ "${HY2_APPLY_UFW}" == "1" ]] || return 0
  if ! command -v ufw >/dev/null 2>&1; then
    warn "ufw not installed; remember to open cloud firewall/security group ports manually"
    return 0
  fi
  local status
  status="$(ufw status 2>/dev/null | head -1 || true)"
  if [[ "${status}" != *"active"* ]]; then
    warn "ufw is not active; cloud firewall/security group still needs these ports open"
    return 0
  fi
  log "adding UFW allow rules"
  run ufw allow "${HY2_PLAIN_PORT}/udp"
  run ufw allow "${HY2_OBFS_PORT}/udp"
  run ufw allow "${HY2_PLAIN_TRAFFIC_PORT}/tcp"
  run ufw allow "${HY2_OBFS_TRAFFIC_PORT}/tcp"
}

start_services() {
  log "starting hysteria services"
  run systemctl daemon-reload
  run systemctl enable --now hysteria-server.service
  run systemctl restart hysteria-server.service
  run systemctl enable --now hysteria-server@config-obfs.service
  run systemctl restart hysteria-server@config-obfs.service
}

verify_runtime() {
  [[ "${HY2_DRY_RUN}" == "1" ]] && return 0
  log "verifying services and traffic API"
  systemctl is-active --quiet hysteria-server.service || fail "hysteria-server.service is not active"
  systemctl is-active --quiet hysteria-server@config-obfs.service || fail "hysteria-server@config-obfs.service is not active"

  ss -lunpt | grep -Eq ":${HY2_PLAIN_PORT}\\b" || fail "UDP ${HY2_PLAIN_PORT} is not listening"
  ss -lunpt | grep -Eq ":${HY2_OBFS_PORT}\\b" || fail "UDP ${HY2_OBFS_PORT} is not listening"
  ss -lntp | grep -Eq ":${HY2_PLAIN_TRAFFIC_PORT}\\b" || fail "TCP ${HY2_PLAIN_TRAFFIC_PORT} is not listening"
  ss -lntp | grep -Eq ":${HY2_OBFS_TRAFFIC_PORT}\\b" || fail "TCP ${HY2_OBFS_TRAFFIC_PORT} is not listening"

  curl -fsS --max-time 5 -H "Authorization: ${HY2_PLAIN_SECRET}" "http://127.0.0.1:${HY2_PLAIN_TRAFFIC_PORT}/online" >/dev/null ||
    fail "plain traffic API check failed"
  curl -fsS --max-time 5 -H "Authorization: ${HY2_OBFS_SECRET}" "http://127.0.0.1:${HY2_OBFS_TRAFFIC_PORT}/online" >/dev/null ||
    fail "obfs traffic API check failed"
}

emit_summary() {
  local plain_sort="${HY2_SORT_BASE}"
  local obfs_sort=$((HY2_SORT_BASE + 1))
  local summary
  summary="$(cat <<EOF

================ HY2 dual-node deploy ================

HY2_DRY_RUN=${HY2_DRY_RUN}
Auth URL: ${HY2_AUTH_URL}
SNI: ${HY2_SNI}
Public Host: ${HY2_PUBLIC_IP}

plain node:
  Name: ${HY2_NODE_PREFIX}-plain
  Host: ${HY2_PUBLIC_IP}
  Port: ${HY2_PLAIN_PORT}
  SNI: ${HY2_SNI}
  Password: 留空
  Skip Cert Verify: 勾选
  Obfs Type: None / 留空
  Obfs Password: 留空
  Traffic API URL: http://${HY2_PUBLIC_IP}:${HY2_PLAIN_TRAFFIC_PORT}
  Traffic API Secret: ${HY2_PLAIN_SECRET}
  Sort Order: ${plain_sort}

obfs node:
  Name: ${HY2_NODE_PREFIX}-obfs
  Host: ${HY2_PUBLIC_IP}
  Port: ${HY2_OBFS_PORT}
  SNI: ${HY2_SNI}
  Password: 留空
  Skip Cert Verify: 勾选
  Obfs Type: salamander
  Obfs Password: ${HY2_OBFS_PASSWORD}
  Traffic API URL: http://${HY2_PUBLIC_IP}:${HY2_OBFS_TRAFFIC_PORT}
  Traffic API Secret: ${HY2_OBFS_SECRET}
  Sort Order: ${obfs_sort}

required firewall/security-group ports:
  ${HY2_PLAIN_PORT}/udp
  ${HY2_OBFS_PORT}/udp
  ${HY2_PLAIN_TRAFFIC_PORT}/tcp
  ${HY2_OBFS_TRAFFIC_PORT}/tcp
  22/tcp

traffic API test:
  curl -H "Authorization: ${HY2_PLAIN_SECRET}" http://${HY2_PUBLIC_IP}:${HY2_PLAIN_TRAFFIC_PORT}/online
  curl -H "Authorization: ${HY2_OBFS_SECRET}" http://${HY2_PUBLIC_IP}:${HY2_OBFS_TRAFFIC_PORT}/online

systemd:
  systemctl status hysteria-server.service
  systemctl status hysteria-server@config-obfs.service
  journalctl -u hysteria-server.service -n 80 --no-pager -o cat
  journalctl -u hysteria-server@config-obfs.service -n 80 --no-pager -o cat

======================================================
EOF
)"
  printf '%s\n' "${summary}"

  if [[ "${HY2_DRY_RUN}" == "1" ]]; then
    return 0
  fi
  if [[ -z "${HY2_OUTPUT_FILE}" ]]; then
    HY2_OUTPUT_FILE="/root/hy2-node-info-${BACKUP_TS}.txt"
  fi
  printf '%s\n' "${summary}" >"${HY2_OUTPUT_FILE}"
  chmod 600 "${HY2_OUTPUT_FILE}"
  log "node info saved to ${HY2_OUTPUT_FILE}"
}

main() {
  parse_args "$@"
  preflight
  detect_public_ip
  check_ports
  check_auth_url
  prepare_secrets
  install_hysteria
  ensure_hysteria_user
  write_cert
  write_systemd_units
  write_configs
  fix_permissions
  configure_ufw
  start_services
  verify_runtime
  emit_summary
}

main "$@"
