# core-hosts-build

vernesong/mihomo `Alpha` 分支（Smart）源码加 hosts 补丁后的 linux-amd64 构建。

## 补丁

| 文件 | 内容 |
|---|---|
| `patches/smart-hosts.patch` | `dialer-proxy` 拨号（`proxydialer.DialContext`）、健康检查（`Proxy.URLTest`）与 Smart 探测（`Proxy.statusRequest`）前按 `hosts` 改写目标：IP 记录改为 IP，域名记录改为映射后的域名 |
| `scripts/apply.py` | 补丁上下文不符时按锚点插入相同改动，锚点不唯一时失败；补丁可直接应用时在干净副本上执行并与补丁结果比对，不一致时给出警告 |
| `tests/applyhosts_test.go` | `ApplyHosts` 单元测试，构建时复制到 `component/proxydialer` |

构建时另将 `component/updater/update_core.go` 的 Alpha 下载地址改为本仓库 `Prerelease-Alpha`，内核自更新取本仓库文件。

## 发布

标签 `Prerelease-Alpha`（预发布），每次构建替换。

| 文件 | 说明 |
|---|---|
| `mihomo-linux-amd64-v1-<版本>.gz` | `GOAMD64=v1`、`CGO_ENABLED=0`、`-tags with_gvisor`，Go 1.26（MetaCubeX/go） |
| `version.txt` | `alpha-smart-<上游短提交>`，与上游版本号格式相同 |
| `checksums.txt` | SHA-256 |

```sh
v=$(curl -fsSL https://github.com/yuekde/core-hosts-build/releases/download/Prerelease-Alpha/version.txt)
curl -fL -o mihomo.gz "https://github.com/yuekde/core-hosts-build/releases/download/Prerelease-Alpha/mihomo-linux-amd64-v1-$v.gz"
```

## 构建

`.github/workflows/build.yml`：

- 每 6 小时检查上游 `Alpha`，提交与 `upstream` 记录不同时构建；
- `patches/`、`scripts/`、`tests/` 或工作流变更时构建；
- 步骤：打补丁 → 按本次源码重新生成 `patches/smart-hosts.patch` → 计数与 gofmt 检查 → `go vet`、`go test` → 编译 → 发布 → 提交 `upstream` 与补丁；
- 仓库中的补丁与 `upstream` 始终对应最近一次发布的二进制，补丁无变化时不提交。

## 许可证

GPL-3.0，与上游相同。
