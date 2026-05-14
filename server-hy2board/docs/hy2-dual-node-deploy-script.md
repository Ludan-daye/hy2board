# HY2 双节点一键部署脚本

脚本位置：

```bash
server-hy2board/scripts/deploy-hy2-dual-node.sh
```

## 默认部署内容

同一台服务器部署两个 Hysteria 2 节点：

| 节点 | HY2 端口 | traffic API | 说明 |
|---|---:|---:|---|
| `plain` | `443/udp` | `25413/tcp` | 不启用 Salamander 混淆 |
| `obfs` | `8443/udp` | `25414/tcp` | 启用 Salamander 混淆 |

脚本会输出可直接填入 hy2board 后台的字段：

- Name
- Host
- Port
- SNI
- Password：留空
- Skip Cert Verify：勾选
- Obfs Type / Obfs Password
- Traffic API URL / Traffic API Secret
- Sort Order

## 使用方式

在新服务器上执行：

```bash
curl -fsSL https://raw.githubusercontent.com/Ludan-daye/hy2board/main/server-hy2board/scripts/deploy-hy2-dual-node.sh -o deploy-hy2-dual-node.sh
sudo HY2_NODE_PREFIX=JP4 bash deploy-hy2-dual-node.sh
```

如果本地已有脚本：

```bash
sudo HY2_NODE_PREFIX=JP4 bash server-hy2board/scripts/deploy-hy2-dual-node.sh
```

## 常用参数

```bash
sudo bash deploy-hy2-dual-node.sh \
  --prefix JP4 \
  --auth-url https://vpn.linkbyfree.com/api/auth/hy2 \
  --sni bing.com \
  --sort-base 30
```

## 环境变量

| 变量 | 默认值 | 说明 |
|---|---|---|
| `HY2_NODE_PREFIX` | `NODE` | 节点名前缀，例如 `JP4` |
| `HY2_AUTH_URL` | `https://vpn.linkbyfree.com/api/auth/hy2` | hy2board HY2 HTTP 鉴权地址 |
| `HY2_SNI` | `bing.com` | 证书 CN 和客户端 SNI |
| `HY2_MASQUERADE_URL` | `https://www.bing.com` | HY2 masquerade 代理地址 |
| `HY2_PUBLIC_IP` | 自动探测 | 输出到 hy2board 的 Host |
| `HY2_SORT_BASE` | `10` | plain 排序；obfs 自动使用 `+1` |
| `HY2_FORCE` | `0` | `1` 时备份并覆盖旧配置 |
| `HY2_SKIP_INSTALL` | `0` | `1` 时不执行官方安装脚本 |
| `HY2_APPLY_UFW` | `1` | UFW active 时自动放行端口 |
| `HY2_DRY_RUN` | `0` | `1` 时只打印动作，不修改服务器 |

可复用 secret：

```bash
HY2_PLAIN_SECRET=...
HY2_OBFS_SECRET=...
HY2_OBFS_PASSWORD=...
```

## 干跑检查

```bash
HY2_DRY_RUN=1 HY2_NODE_PREFIX=TEST HY2_PUBLIC_IP=203.0.113.10 \
  bash server-hy2board/scripts/deploy-hy2-dual-node.sh --dry-run --skip-install --no-ufw
```

## 防火墙

云厂商安全组必须放行：

```text
443/udp
8443/udp
25413/tcp
25414/tcp
22/tcp
```

服务器使用 UFW 且已启用时，脚本会自动执行：

```bash
ufw allow 443/udp
ufw allow 8443/udp
ufw allow 25413/tcp
ufw allow 25414/tcp
```

## 验证命令

```bash
systemctl status hysteria-server.service
systemctl status hysteria-server@config-obfs.service

ss -lunpt | grep -E ':443|:8443'
ss -lntp | grep -E ':25413|:25414'

curl -H "Authorization: Plain traffic secret" http://公网IP:25413/online
curl -H "Authorization: Obfs traffic secret" http://公网IP:25414/online
```
