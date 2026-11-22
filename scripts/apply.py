#!/usr/bin/env python3
# 补丁上下文不符时按锚点替换：apply.py <补丁名>，在源码根目录执行，锚点须唯一
import subprocess
import sys


def after(path, anchor, text):
    return path, anchor, anchor + text


def before(path, anchor, text):
    return path, anchor, text + anchor


PD = "component/proxydialer/proxydialer.go"
AD = "adapter/adapter.go"
RS = "component/resolver/resolver.go"
HC = "adapter/provider/healthcheck.go"

APPLY_HOSTS = """// ApplyHosts 按 hosts 改写目标：IP 记录改为 IP，域名记录改为映射后的域名
func ApplyHosts(metadata *C.Metadata) {
\tif metadata.Host == "" {
\t\treturn
\t}
\tnode, ok := resolver.DefaultHosts.Search(metadata.Host, false)
\tif ok {
\t\tif ip, err := node.RandIP(); err == nil {
\t\t\tmetadata.DstIP, metadata.Host = ip, ""
\t\t}
\t} else if node != nil && node.IsDomain {
\t\tmetadata.Host = node.Domain
\t}
}

"""

MAPPED_HOST = """// mappedHost 返回 hosts 域名记录映射后的域名（链末端不是 IP 时），否则返回原值
func mappedHost(host string) string {
\tif node, ok := DefaultHosts.Search(host, false); !ok && node != nil && node.IsDomain {
\t\treturn node.Domain
\t}
\treturn host
}

"""

HC_SEM = """// healthCheckSem 为所有 provider 共用的节点测试并发上限，环境变量 HEALTH_CHECK_CONCURRENCY 可调，默认 10
var healthCheckSem = make(chan struct{}, func() int {
\tif n, err := strconv.Atoi(os.Getenv("HEALTH_CHECK_CONCURRENCY")); err == nil && n > 0 {
\t\treturn n
\t}
\treturn 10
}())

"""

EDITS = {
    "smart-hosts": [
        after(PD, '\tN "github.com/metacubex/mihomo/common/net"\n',
              '\t"github.com/metacubex/mihomo/component/resolver"\n'),
        before(PD, "func (p proxyDialer) DialContext(", APPLY_HOSTS),
        after(PD, "\tif err := currentMeta.SetRemoteAddress(address); err != nil {\n\t\treturn nil, err\n\t}\n",
              "\tApplyHosts(currentMeta)\n"),
        after(AD, '\t"github.com/metacubex/mihomo/component/ca"\n',
              '\t"github.com/metacubex/mihomo/component/proxydialer"\n'),
        after(AD, "\taddr, err := urlToMetadata(url)\n\tif err != nil {\n\t\treturn\n\t}\n",
              "\tproxydialer.ApplyHosts(&addr)\n"),
        after(AD, "\t\tif err := metadata.SetRemoteAddress(targetAddr); err != nil {\n\t\t\treturn nil, err\n\t\t}\n",
              "\t\tproxydialer.ApplyHosts(&metadata)\n"),
    ],
    "resolver-hosts-domain": [
        before(RS, "// LookupIPv4WithResolver same as LookupIPv4, but with a resolver\n", MAPPED_HOST),
        after(RS, "\t\t\treturn ip.Is4()\n\t\t}); len(addrs) > 0 {\n\t\t\treturn addrs, nil\n\t\t}\n\t}\n",
              "\thost = mappedHost(host)\n"),
        after(RS, "\t\t\treturn ip.Is6()\n\t\t}); len(addrs) > 0 {\n\t\t\treturn addrs, nil\n\t\t}\n\t}\n",
              "\thost = mappedHost(host)\n"),
        after(RS, "\tif node, ok := DefaultHosts.Search(host, false); ok {\n\t\treturn node.IPs, nil\n\t}\n",
              "\thost = mappedHost(host)\n"),
    ],
    "healthcheck-stagger": [
        after(HC, '\t"context"\n', '\t"math/rand"\n\t"os"\n\t"strconv"\n'),
        before(HC, "func (hc *HealthCheck) process() {\n", HC_SEM),
        (HC, "\tticker := time.NewTicker(hc.interval)\n\tgo hc.check()\n",
         "\tgo hc.check()\n"
         "\t// 定时检查起点随机偏移，错开各 provider 的检查时刻\n"
         "\tselect {\n"
         "\tcase <-time.After(time.Duration(rand.Int63n(int64(hc.interval)))):\n"
         "\tcase <-hc.ctx.Done():\n"
         "\t\treturn\n"
         "\t}\n"
         "\tticker := time.NewTicker(hc.interval)\n"),
        (HC, "\t\tb.Go(func() error {\n\t\t\tctx, cancel := context.WithTimeout(hc.ctx, hc.timeout)\n",
         "\t\tb.Go(func() error {\n"
         "\t\t\t// 等待并发名额的时间不计入超时\n"
         "\t\t\tselect {\n"
         "\t\t\tcase healthCheckSem <- struct{}{}:\n"
         "\t\t\tcase <-hc.ctx.Done():\n"
         "\t\t\t\treturn nil\n"
         "\t\t\t}\n"
         "\t\t\tdefer func() { <-healthCheckSem }()\n"
         "\t\t\tctx, cancel := context.WithTimeout(hc.ctx, hc.timeout)\n"),
    ],
}

if len(sys.argv) != 2 or sys.argv[1] not in EDITS:
    sys.exit("用法: apply.py <" + "|".join(EDITS) + ">")

src = {}
for path, old, new in EDITS[sys.argv[1]]:
    if path not in src:
        with open(path, encoding="utf-8") as f:
            src[path] = f.read()
    if new in src[path]:
        continue
    n = src[path].count(old)
    if n != 1:
        sys.exit(f"{path}: 锚点出现 {n} 次: {old.strip()[:60]}")
    src[path] = src[path].replace(old, new)

for path, s in src.items():
    with open(path, "w", encoding="utf-8") as f:
        f.write(s)
# gofmt 对导入排序
subprocess.run(["gofmt", "-w", *src], check=True)
