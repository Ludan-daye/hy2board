# HY2 双节点一键部署脚本使用说明

这个目录里的 `deploy-hy2-dual-node.sh` 用于在一台新服务器上一键部署两个 Hysteria 2 节点：

| 节点 | HY2 端口 | Traffic API | 说明 |
|---|---:|---:|---|
| `plain` | `443/udp` | `25413/tcp` | 普通 HY2 节点 |
| `obfs` | `8443/udp` | `25414/tcp` | HY2 + Salamander 混淆节点 |

## 一键部署

在新服务器上执行：

```bash
curl -fsSL https://raw.githubusercontent.com/Ludan-daye/hy2board/main/server-hy2board/scripts/deploy-hy2-dual-node.sh -o deploy-hy2-dual-node.sh
sudo HY2_SERVER_NAME=JP4 bash deploy-hy2-dual-node.sh
```

部署完成后，脚本会输出两条可直接填入 hy2board 后台的节点信息：

```text
JP4-plain
JP4-obfs
```

## 已有配置时覆盖

如果服务器以前已经部署过 HY2，需要覆盖旧配置：

```bash
sudo HY2_SERVER_NAME=JP4 bash deploy-hy2-dual-node.sh --force
```

脚本会先备份旧配置，再写入新配置。

## 干跑检查

只检查流程和输出，不修改服务器：

```bash
HY2_DRY_RUN=1 HY2_SERVER_NAME=TEST HY2_PUBLIC_IP=203.0.113.10 \
  bash deploy-hy2-dual-node.sh --dry-run --skip-install --no-ufw
```

## 常用变量

| 变量 | 说明 | 默认值 |
|---|---|---|
| `HY2_SERVER_NAME` | 服务器名/线路名，例如 `JP4` | `SERVER` |
| `HY2_AUTH_URL` | hy2board HY2 鉴权地址 | `https://vpn.linkbyfree.com/api/auth/hy2` |
| `HY2_SNI` | 客户端 SNI / 自签证书 CN | `bing.com` |
| `HY2_PUBLIC_IP` | 输出给 hy2board 的 Host | 自动探测 |
| `HY2_SORT_BASE` | plain 排序，obfs 自动 +1 | `10` |
| `HY2_PACKAGE_MANAGER` | 指定包管理器：`apt` / `dnf` / `yum` / `apk` | 自动探测 |

复用已有 secret：

```bash
sudo HY2_SERVER_NAME=JP4 \
  HY2_PLAIN_SECRET=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx \
  HY2_OBFS_SECRET=yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy \
  HY2_OBFS_PASSWORD=zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz \
  bash deploy-hy2-dual-node.sh --force
```

## 云厂商防火墙

安全组必须放行：

```text
443/udp
8443/udp
25413/tcp
25414/tcp
22/tcp
```

服务器本机如果启用了 UFW，脚本会自动放行这些端口。

## 部署后验证

```bash
systemctl status hysteria-server.service
systemctl status hysteria-server@config-obfs.service

ss -lunpt | grep -E ':443|:8443'
ss -lntp | grep -E ':25413|:25414'
```

Traffic API 验证：

```bash
curl -H "Authorization: Plain traffic secret" http://公网IP:25413/online
curl -H "Authorization: Obfs traffic secret" http://公网IP:25414/online
```

正常会返回 JSON。

## 完整文档

更详细的说明在：

```text
server-hy2board/docs/hy2-dual-node-deploy-script.md
```
