// 构建时复制到源码同一路径执行
package provider

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/metacubex/mihomo/common/utils"
	C "github.com/metacubex/mihomo/constant"
)

// countProxy 记录同时进行的 URLTest 数
type countProxy struct {
	C.Proxy
	name       string
	cur, peak  *atomic.Int32
	total      *atomic.Int32
	testLength time.Duration
}

func (p *countProxy) Name() string                      { return p.name }
func (p *countProxy) AliveForTestUrl(string) bool       { return true }
func (p *countProxy) LastDelayForTestUrl(string) uint16 { return 1 }

func (p *countProxy) URLTest(context.Context, string, utils.IntRanges[uint16]) (uint16, error) {
	n := p.cur.Add(1)
	for m := p.peak.Load(); n > m && !p.peak.CompareAndSwap(m, n); m = p.peak.Load() {
	}
	time.Sleep(p.testLength)
	p.cur.Add(-1)
	p.total.Add(1)
	return 1, nil
}

func TestHealthCheckSharedLimit(t *testing.T) {
	var cur, peak, total atomic.Int32
	const providers, nodes = 3, 20
	hcs := make([]*HealthCheck, providers)
	for i := range hcs {
		proxies := make([]C.Proxy, nodes)
		for j := range proxies {
			proxies[j] = &countProxy{name: fmt.Sprintf("p%d-%d", i, j), cur: &cur, peak: &peak, total: &total, testLength: 20 * time.Millisecond}
		}
		hcs[i] = NewHealthCheck(proxies, "https://example.com/generate_204", 0, 0, false, nil)
	}

	var wg sync.WaitGroup
	for _, hc := range hcs {
		wg.Add(1)
		go func(hc *HealthCheck) {
			defer wg.Done()
			hc.check()
		}(hc)
	}
	wg.Wait()

	if got := total.Load(); got != providers*nodes {
		t.Errorf("完成 %d 次测试，应为 %d", got, providers*nodes)
	}
	// 未修改时每个 provider 各 10 个并发，3 个 provider 合计可达 30
	if got := peak.Load(); got < 1 || int(got) > cap(healthCheckSem) {
		t.Errorf("同时测试峰值 %d，上限 %d", got, cap(healthCheckSem))
	}
}

func TestHealthCheckStartOffset(t *testing.T) {
	var cur, peak, total atomic.Int32
	p := &countProxy{name: "p", cur: &cur, peak: &peak, total: &total}
	hc := NewHealthCheck([]C.Proxy{p}, "https://example.com/generate_204", 0, 1, false, nil)
	go hc.process()
	defer hc.close()

	// 启动检查立即执行；首次定时检查在 [interval, 2*interval) 内
	time.Sleep(500 * time.Millisecond)
	if got := total.Load(); got != 1 {
		t.Fatalf("启动后 0.5 秒完成 %d 次，应为 1", got)
	}
	time.Sleep(2 * time.Second)
	if got := total.Load(); got < 2 {
		t.Fatalf("启动后 2.5 秒完成 %d 次，应至少 2", got)
	}
}
