// 构建时复制到源码同一路径执行
package resolver

import (
	"context"
	"errors"
	"net/netip"
	"slices"
	"testing"

	"github.com/metacubex/mihomo/component/trie"

	"github.com/miekg/dns"
)

// stubResolver 只解析 ips 中的域名
type stubResolver struct {
	ips map[string][]netip.Addr
}

func (r stubResolver) lookup(host string, keep func(netip.Addr) bool) ([]netip.Addr, error) {
	var out []netip.Addr
	for _, ip := range r.ips[host] {
		if keep(ip) {
			out = append(out, ip)
		}
	}
	if len(out) == 0 {
		return nil, errors.New("not found: " + host)
	}
	return out, nil
}

func (r stubResolver) LookupIP(_ context.Context, host string) ([]netip.Addr, error) {
	return r.lookup(host, func(netip.Addr) bool { return true })
}

func (r stubResolver) LookupIPv4(_ context.Context, host string) ([]netip.Addr, error) {
	return r.lookup(host, netip.Addr.Is4)
}

func (r stubResolver) LookupIPv6(_ context.Context, host string) ([]netip.Addr, error) {
	return r.lookup(host, netip.Addr.Is6)
}

func (stubResolver) ResolveECH(context.Context, string) ([]byte, error)           { return nil, nil }
func (stubResolver) ExchangeContext(context.Context, *dns.Msg) (*dns.Msg, error) { return nil, nil }
func (stubResolver) Invalid() bool                                               { return true }
func (stubResolver) ClearCache()                                                 {}
func (stubResolver) ResetConnection()                                            {}

func TestLookupHostsDomain(t *testing.T) {
	v4, v6, fixed := netip.MustParseAddr("5.6.7.8"), netip.MustParseAddr("2001:db8::1"), netip.MustParseAddr("1.2.3.4")
	must := func(v HostValue, err error) HostValue {
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	tr := trie.New[HostValue]()
	for domain, v := range map[string]HostValue{
		"a.com":   must(NewHostValueByDomain("b.com")),
		"+.w.com": must(NewHostValueByDomain("b.com")),
		"c.com":   must(NewHostValueByDomain("d.com")),
		"d.com":   must(NewHostValueByIPs([]netip.Addr{fixed})),
	} {
		if err := tr.Insert(domain, v); err != nil {
			t.Fatal(err)
		}
	}
	oldHosts, oldV6 := DefaultHosts, DisableIPv6
	DefaultHosts, DisableIPv6 = NewHosts(tr), false
	defer func() { DefaultHosts, DisableIPv6 = oldHosts, oldV6 }()

	r := stubResolver{ips: map[string][]netip.Addr{"b.com": {v4, v6}, "none.com": {v4}}}
	ctx := context.Background()
	lookups := map[string]func(context.Context, string, Resolver) ([]netip.Addr, error){
		"LookupIP":   LookupIPWithResolver,
		"LookupIPv4": LookupIPv4WithResolver,
		"LookupIPv6": LookupIPv6WithResolver,
	}
	cases := []struct {
		lookup string
		host   string
		want   []netip.Addr
	}{
		{"LookupIP", "a.com", []netip.Addr{v4, v6}},   // 链末端为域名，解析映射后的域名
		{"LookupIPv4", "a.com", []netip.Addr{v4}},     // 同上，仅 IPv4
		{"LookupIPv6", "a.com", []netip.Addr{v6}},     // 同上，仅 IPv6
		{"LookupIPv4", "x.w.com", []netip.Addr{v4}},   // 通配记录
		{"LookupIP", "c.com", []netip.Addr{fixed}},    // 链末端为 IP，行为不变
		{"LookupIPv4", "none.com", []netip.Addr{v4}},  // 无 hosts 记录，行为不变
	}
	for _, c := range cases {
		got, err := lookups[c.lookup](ctx, c.host, r)
		if err != nil || !slices.Equal(got, c.want) {
			t.Errorf("%s(%s) = %v, %v; want %v", c.lookup, c.host, got, err, c.want)
		}
	}
	if _, err := LookupIPv4WithResolver(ctx, "e.com", r); err == nil {
		t.Error("LookupIPv4(e.com) 应失败")
	}
}
