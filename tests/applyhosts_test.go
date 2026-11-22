// 构建时复制到 component/proxydialer 执行
package proxydialer

import (
	"net/netip"
	"testing"

	"github.com/metacubex/mihomo/component/resolver"
	"github.com/metacubex/mihomo/component/trie"
	C "github.com/metacubex/mihomo/constant"
)

func TestApplyHosts(t *testing.T) {
	ip := netip.MustParseAddr("1.2.3.4")
	tr := trie.New[resolver.HostValue]()
	add := func(domain string, v resolver.HostValue, err error) {
		if err != nil {
			t.Fatal(err)
		}
		if err := tr.Insert(domain, v); err != nil {
			t.Fatal(err)
		}
	}
	add("test.com", resolver.NewHostValueByIPs([]netip.Addr{ip}))
	add("a.com", resolver.NewHostValueByDomain("b.com"))
	add("c.com", resolver.NewHostValueByDomain("d.com"))
	add("d.com", resolver.NewHostValueByIPs([]netip.Addr{ip}))
	add("+.w.com", resolver.NewHostValueByIPs([]netip.Addr{ip}))

	old := resolver.DefaultHosts
	resolver.DefaultHosts = resolver.NewHosts(tr)
	defer func() { resolver.DefaultHosts = old }()

	cases := []struct {
		addr string
		host string
		ip   netip.Addr
	}{
		{"test.com:443", "", ip},             // IP 记录
		{"a.com:443", "b.com", netip.Addr{}}, // 域名记录，链末端无 IP
		{"c.com:443", "", ip},                // 域名记录，链末端为 IP
		{"x.w.com:443", "", ip},              // 通配记录
		{"none.com:443", "none.com", netip.Addr{}},
		{"5.6.7.8:443", "", netip.MustParseAddr("5.6.7.8")},
	}
	for _, c := range cases {
		var m C.Metadata
		if err := m.SetRemoteAddress(c.addr); err != nil {
			t.Fatal(err)
		}
		ApplyHosts(&m)
		if m.Host != c.host || m.DstIP != c.ip || m.DstPort != 443 {
			t.Errorf("%s: host=%q ip=%v port=%d", c.addr, m.Host, m.DstIP, m.DstPort)
		}
	}
}
