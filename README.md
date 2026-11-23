# core-hosts-build

vernesong/mihomo `Alpha` 分支（Smart）源码加补丁后的 linux-amd64 构建。

## 补丁

| 文件 | 内容 |
|---|---|
| `patches/smart-hosts.patch` | `dialer-proxy` 拨号（`proxydialer.DialContext`）、健康检查（`Proxy.URLTest`）与 Smart 探测（`Proxy.statusRequest`）前按 `hosts` 改写目标：IP 记录改为 IP，域名记录改为映射后的域名 |
| `patches/resolver-hosts-domain.patch` | `LookupIP*WithResolver`（节点 `server`、DIRECT 出站等的解析）：hosts 域名记录链末端不是 IP 时，改为解析映射后的域名 |
| `patches/healthcheck-stagger.patch` | 健康检查：启动检查照常立即执行，定时检查起点在 `[0, interval)` 内随机偏移；所有 provider 共用节点测试并发上限，默认 10，环境变量 `HEALTH_CHECK_CONCURRENCY` 可调，等待名额的时间不计入超时 |
| `scripts/apply.py <补丁名>` | 补丁上下文不符时按锚点替换为相同改动，锚点不唯一时失败；补丁可直接应用时在干净副本上执行并与补丁结果比对，不一致时给出警告 |
| `tests/` | 单元测试，按源码路径存放，构建时复制到源码中执行 |

各补丁修改的文件不重叠。

构建时另将 `component/updater/update_core.go` 的 Alpha 下载地址改为本仓库 `Prerelease-Alpha`，内核自更新取本仓库文件。

## 限制

`smart-hosts.patch` 只改写交给前置节点的目标地址。TLS 类落地节点（VLESS、Trojan、VMess、AnyTLS 等开启 TLS 时）的 SNI 以明文经过前置节点，SNI 取 `servername` / `sni`，未设置时为 `server` 域名；非 TLS 的 WebSocket、HTTP 传输中 `Host` 头同样为明文。前置节点服务端开启嗅探并覆盖目标时，以该域名替换 IP 后自行解析，hosts 中的自定义域名在前置节点无法解析，连接失败。客户端无法关闭服务端嗅探。Shadowsocks 等不含明文域名的协议不受影响。

前置节点服务端覆盖目标的配置：mihomo `sniffer` 启用且目标端口在嗅探端口内（`parse-pure-ip`、`override-destination` 默认均为 `true`）；Xray `sniffing` 启用 `destOverride` 且未设置 `routeOnly: true`。

| 处理方法 | 条件 |
|---|---|
| 前置节点服务端不覆盖目标：mihomo `sniffer.override-destination: false`，或落地 IP 加入 `skip-dst-address`；Xray `sniffing.routeOnly: true` | 可修改前置节点服务端配置 |
| 落地节点 `servername` / `sni` 写落地 IP，TLS 不发送 SNI | 落地服务端在无 SNI 时返回该节点证书；证书校验需证书含该 IP，或配置 `fingerprint`、`skip-cert-verify`；不适用于 Reality 与按 SNI 分流的服务端 |

## 发布

标签 `Prerelease-Alpha`（预发布），每次构建替换。

| 文件 | 说明 |
|---|---|
| `mihomo-linux-amd64-v1-<版本>.gz` | `GOAMD64=v1`、`CGO_ENABLED=0`、`-tags with_gvisor`，Go 1.26（MetaCubeX/go） |
| `version.txt` | `alpha-smart-<上游短提交>`，与上游版本号格式相同；上游提交不变时补丁变更不改变版本号，内核自更新（按字符串比较版本号）不会拉取该次构建 |
| `checksums.txt` | SHA-256 |

```sh
v=$(curl -fsSL https://github.com/yuekde/core-hosts-build/releases/download/Prerelease-Alpha/version.txt)
curl -fL -o mihomo.gz "https://github.com/yuekde/core-hosts-build/releases/download/Prerelease-Alpha/mihomo-linux-amd64-v1-$v.gz"
```

## 构建

`.github/workflows/build.yml`：

- 每天 UTC 17:00（北京时间 01:00）检查上游 `Alpha`，提交与 `upstream` 记录不同时构建；GitHub 定时任务可能延迟数分钟至更久；
- `patches/`、`scripts/`、`tests/` 或工作流变更时构建；
- 步骤：逐个打补丁 → 按本次源码重新生成各补丁 → 计数与 gofmt 检查 → `go vet`、`go test` → 编译 → 发布 → 提交 `upstream` 与补丁；
- 仓库中的补丁与 `upstream` 始终对应最近一次发布的二进制，补丁无变化时不提交。

## 许可证

GPL-3.0，与上游相同。
