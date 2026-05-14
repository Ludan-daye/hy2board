#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT="${SCRIPT_DIR}/deploy-hy2-dual-node.sh"

if [[ ! -f "${SCRIPT}" ]]; then
  echo "missing deploy script: ${SCRIPT}" >&2
  exit 1
fi

bash -n "${SCRIPT}"

output="$(
  HY2_DRY_RUN=1 \
  HY2_SKIP_INSTALL=1 \
  HY2_SERVER_NAME="TST" \
  HY2_PUBLIC_IP="203.0.113.10" \
  HY2_PACKAGE_MANAGER="apt" \
  HY2_ASSUME_MISSING_DEPS=1 \
  HY2_PLAIN_SECRET="plainsecret000000000000000000" \
  HY2_OBFS_SECRET="obfssecret0000000000000000000" \
  HY2_OBFS_PASSWORD="obfspass00000000000000000000" \
  "${SCRIPT}" --dry-run --skip-install --no-ufw
)"

grep -q "HY2 dual-node deploy" <<<"${output}"
grep -q "TST-plain" <<<"${output}"
grep -q "TST-obfs" <<<"${output}"
grep -q "Host: 203.0.113.10" <<<"${output}"
grep -q "Traffic API URL: http://203.0.113.10:25413" <<<"${output}"
grep -q "Traffic API URL: http://203.0.113.10:25414" <<<"${output}"
grep -q "Obfs Type: salamander" <<<"${output}"
grep -q "HY2_DRY_RUN=1" <<<"${output}"
grep -q "apt-get install -y" <<<"${output}"

legacy_output="$(
  HY2_DRY_RUN=1 \
  HY2_SKIP_INSTALL=1 \
  HY2_NODE_PREFIX="LEGACY" \
  HY2_PUBLIC_IP="203.0.113.11" \
  HY2_PLAIN_SECRET="plainsecret000000000000000001" \
  HY2_OBFS_SECRET="obfssecret0000000000000000001" \
  HY2_OBFS_PASSWORD="obfspass00000000000000000001" \
  "${SCRIPT}" --dry-run --skip-install --no-ufw
)"

grep -q "LEGACY-plain" <<<"${legacy_output}"
grep -q "LEGACY-obfs" <<<"${legacy_output}"

echo "deploy-hy2-dual-node smoke test passed"
