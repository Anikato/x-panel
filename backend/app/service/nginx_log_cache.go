package service

import (
	"context"
	"sync"
	"time"

	"xpanel/app/dto"
	"xpanel/buserr"
	"xpanel/constant"
)

var (
	logCacheTTL              = 30 * time.Second
	logCacheMaxEntries       = 16
	logCacheMaxBytes   int64 = 8 << 20
	logScanMaxWaiters        = 32
	logScanMaxQueue          = 8
)

type analysisCacheItem struct {
	key     string
	result  *dto.NginxLogAnalysis
	drill   *dto.NginxLogDrilldownResp
	size    int64
	expires time.Time
}

type analysisCache struct {
	mu    sync.Mutex
	items map[string]*analysisCacheItem
	order []string
	bytes int64
}

func newAnalysisCache() *analysisCache {
	return &analysisCache{items: map[string]*analysisCacheItem{}}
}

var logAnalysisCache = newAnalysisCache()

func resetLogAnalysisStateForTest() {
	logScanMaxBytes = 64 << 20
	logScanMaxLines = 200000
	logScanMaxLineBytes = 64 * 1024
	logScanMaxKeys = 50000
	logScanTimeout = 10 * time.Second
	logScanDelay = 0
	logScanTestHook = nil
	logScanChunkSize = 64 * 1024
	logScanTestReadHook = nil
	logCacheTTL = 30 * time.Second
	logCacheMaxEntries = 16
	logCacheMaxBytes = 8 << 20
	logScanMaxWaiters = 32
	logScanMaxQueue = 8
	logAnalysisCache = newAnalysisCache()
	logAnalysisFlights = newFlightGroup()
}

func logAnalysisCacheSize() int {
	logAnalysisCache.mu.Lock()
	defer logAnalysisCache.mu.Unlock()
	return len(logAnalysisCache.items)
}

func logAnalysisInFlight() int {
	logAnalysisFlights.mu.Lock()
	defer logAnalysisFlights.mu.Unlock()
	return len(logAnalysisFlights.flights)
}

func (c *analysisCache) get(key string) (*analysisCacheItem, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	item, ok := c.items[key]
	if !ok {
		return nil, false
	}
	if time.Now().After(item.expires) {
		c.removeLocked(key)
		return nil, false
	}
	return item, true
}

func (c *analysisCache) put(item *analysisCacheItem) {
	if item == nil || item.size <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if existing, ok := c.items[item.key]; ok {
		c.bytes -= existing.size
		c.removeOrderLocked(item.key)
	}
	for (logCacheMaxEntries > 0 && len(c.items) >= logCacheMaxEntries) || (logCacheMaxBytes > 0 && c.bytes+item.size > logCacheMaxBytes) {
		if len(c.order) == 0 {
			break
		}
		c.removeLocked(c.order[0])
	}
	c.items[item.key] = item
	c.order = append(c.order, item.key)
	c.bytes += item.size
}

func (c *analysisCache) removeLocked(key string) {
	item, ok := c.items[key]
	if !ok {
		return
	}
	c.bytes -= item.size
	delete(c.items, key)
	c.removeOrderLocked(key)
}

func (c *analysisCache) removeOrderLocked(key string) {
	for i, k := range c.order {
		if k == key {
			c.order = append(c.order[:i], c.order[i+1:]...)
			return
		}
	}
}

func estimateAnalysisSize(res *dto.NginxLogAnalysis) int64 {
	if res == nil {
		return 0
	}
	n := int64(2048)
	n += int64(len(res.TopURLs)+len(res.TopIPs)+len(res.TopUserAgents)+len(res.TopThreats)+len(res.ThreatIPs)+len(res.TopCrawlers)) * 96
	n += int64(len(res.HourlyStats)+len(res.DailyStats)) * 48
	n += int64(len(res.StatusCodes)) * 24
	n += int64(len(res.Meta.Reasons)+len(res.Meta.FailedFiles)+len(res.Meta.SkippedFiles)) * 64
	return n
}

func estimateDrillSize(res *dto.NginxLogDrilldownResp) int64 {
	if res == nil {
		return 0
	}
	return int64(1024 + (len(res.IPs)+len(res.URLs))*96)
}

type flight struct {
	waiters int
	cancel  context.CancelFunc
	done    chan struct{}
	result  *dto.NginxLogAnalysis
	drill   *dto.NginxLogDrilldownResp
	err     error
}

type flightGroup struct {
	mu      sync.Mutex
	flights map[string]*flight
}

func newFlightGroup() *flightGroup {
	return &flightGroup{flights: map[string]*flight{}}
}

var (
	logAnalysisFlights = newFlightGroup()
	logAnalysisSem     = make(chan struct{}, 1)
)

func acquireLogScan(ctx context.Context) error {
	select {
	case logAnalysisSem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func releaseLogScan() {
	select {
	case <-logAnalysisSem:
	default:
	}
}

func (g *flightGroup) doAnalysis(ctx context.Context, key string, run func(context.Context) (*dto.NginxLogAnalysis, error)) (*dto.NginxLogAnalysis, error) {
	res, _, err := g.do(ctx, key, func(runCtx context.Context) (*dto.NginxLogAnalysis, *dto.NginxLogDrilldownResp, error) {
		out, err := run(runCtx)
		return out, nil, err
	})
	return res, err
}

func (g *flightGroup) doDrill(ctx context.Context, key string, run func(context.Context) (*dto.NginxLogDrilldownResp, error)) (*dto.NginxLogDrilldownResp, error) {
	_, drill, err := g.do(ctx, key, func(runCtx context.Context) (*dto.NginxLogAnalysis, *dto.NginxLogDrilldownResp, error) {
		out, err := run(runCtx)
		return nil, out, err
	})
	return drill, err
}

func (g *flightGroup) queuedLocked() int {
	n := 0
	for _, f := range g.flights {
		n += f.waiters
	}
	return n
}

func (g *flightGroup) finish(key string, f *flight) {
	g.mu.Lock()
	if g.flights[key] == f {
		delete(g.flights, key)
	}
	g.mu.Unlock()
}

func (g *flightGroup) do(ctx context.Context, key string, run func(context.Context) (*dto.NginxLogAnalysis, *dto.NginxLogDrilldownResp, error)) (*dto.NginxLogAnalysis, *dto.NginxLogDrilldownResp, error) {
	g.mu.Lock()
	if logScanMaxQueue > 0 && g.queuedLocked() >= logScanMaxQueue {
		g.mu.Unlock()
		return nil, nil, buserr.New(constant.ErrNginxLogBusy)
	}
	f := g.flights[key]
	if f == nil || f.waiters <= 0 {
		runCtx, cancel := context.WithCancel(context.Background())
		f = &flight{waiters: 1, cancel: cancel, done: make(chan struct{})}
		g.flights[key] = f
		g.mu.Unlock()
		go func() {
			if err := acquireLogScan(runCtx); err != nil {
				f.err = err
				close(f.done)
				g.finish(key, f)
				return
			}
			res, drill, err := run(runCtx)
			releaseLogScan()
			f.result, f.drill, f.err = res, drill, err
			close(f.done)
			g.finish(key, f)
		}()
	} else {
		if f.waiters >= logScanMaxWaiters {
			g.mu.Unlock()
			return nil, nil, buserr.New(constant.ErrNginxLogBusy)
		}
		f.waiters++
		g.mu.Unlock()
	}

	select {
	case <-f.done:
		return f.result, f.drill, f.err
	case <-ctx.Done():
		g.mu.Lock()
		f.waiters--
		if f.waiters <= 0 {
			f.cancel()
		}
		g.mu.Unlock()
		return nil, nil, ctx.Err()
	}
}
