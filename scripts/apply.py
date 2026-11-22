#!/usr/bin/env python3
# 补丁上下文不符时按锚点插入；在源码根目录执行，锚点须唯一
import subprocess
import sys

PD = "component/proxydialer/proxydialer.go"
AD = "adapter/adapter.go"

FUNC = """// ApplyHosts 按 hosts 改写目标：IP 记录改为 IP，域名记录改为映射后的域名
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

# (文件, 锚点, 插入内容, 是否插在锚点之后)
EDITS = [
    (PD, '\tN "github.com/metacubex/mihomo/common/net"\n',
     '\t"github.com/metacubex/mihomo/component/resolver"\n', True),
    (PD, "func (p proxyDialer) DialContext(", FUNC, False),
    (PD, "\tif err := currentMeta.SetRemoteAddress(address); err != nil {\n\t\treturn nil, err\n\t}\n",
     "\tApplyHosts(currentMeta)\n", True),
    (AD, '\t"github.com/metacubex/mihomo/component/ca"\n',
     '\t"github.com/metacubex/mihomo/component/proxydialer"\n', True),
    (AD, "\taddr, err := urlToMetadata(url)\n\tif err != nil {\n\t\treturn\n\t}\n",
     "\tproxydialer.ApplyHosts(&addr)\n", True),
    (AD, "\t\tif err := metadata.SetRemoteAddress(targetAddr); err != nil {\n\t\t\treturn nil, err\n\t\t}\n",
     "\t\tproxydialer.ApplyHosts(&metadata)\n", True),
]

src = {}
for path, anchor, text, after in EDITS:
    if path not in src:
        with open(path, encoding="utf-8") as f:
            src[path] = f.read()
    if text in src[path]:
        continue
    n = src[path].count(anchor)
    if n != 1:
        sys.exit(f"{path}: 锚点出现 {n} 次: {anchor.strip()[:60]}")
    src[path] = src[path].replace(anchor, anchor + text if after else text + anchor)

for path, s in src.items():
    with open(path, "w", encoding="utf-8") as f:
        f.write(s)
# gofmt 对导入排序
subprocess.run(["gofmt", "-w", *src], check=True)
