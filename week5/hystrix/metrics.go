package hystrix

import (
	metricCollector "github.com/ltinyho/go-homework/week5/hystrix/metric_collector"
	"github.com/ltinyho/go-homework/week5/hystrix/rolling"
	"sync"
	"time"
)

// 传递command执行情况
type commandExecution struct {
	Types            []event
	Start            time.Time
	RunDuration      time.Duration
	ConcurrencyInUse float64
}

type metricExchange struct {
	Name             string
	mutex            *sync.RWMutex
	metricCollectors []metricCollector.MetricCollector
	Updates          chan *commandExecution // 执行情况
}

// newMetricExchange
func newMetricExchange(name string) *metricExchange {
	m := &metricExchange{}
	m.Name = name
	m.Updates = make(chan *commandExecution, 2000)
	m.mutex = &sync.RWMutex{}
	m.metricCollectors = metricCollector.Registry.InitializeMetricCollectors(name)
	m.Reset()

	go m.monitor()

	return m
}

func (m *metricExchange) monitor() {
	for update := range m.Updates {
		// 读锁确保 Reset 不改变数量
		m.mutex.RLock()
		totalDuration := time.Since(update.Start)
		wg := &sync.WaitGroup{}
		for _, collector := range m.metricCollectors {
			wg.Add(1)
			go m.IncrementMetrics(wg, collector, update, totalDuration)
		}
		wg.Wait()
		m.mutex.RUnlock()
	}
}

// The Default Collector function will panic if collectors are not setup to specification.
// 默认的收集器必须设置,用来收集请求的分布情况
func (m *metricExchange) defaultCollector() *metricCollector.DefaultMetricCollector {
	if len(m.metricCollectors) < 1 {
		panic("No Metric Collectors Registered.")
	}
	collection, ok := m.metricCollectors[0].(*metricCollector.DefaultMetricCollector)
	if !ok {
		panic("Default metric collector is not registered correctly. The default metric collector must be registered first.")
	}
	return collection
}

// IncrementMetrics 增加统计信息
func (m *metricExchange) IncrementMetrics(wg *sync.WaitGroup, collector metricCollector.MetricCollector, update *commandExecution, totalDuration time.Duration) {
	r := metricCollector.MetricResult{
		Attempts:         1,
		TotalDuration:    totalDuration,
		RunDuration:      update.RunDuration,
		ConcurrencyInUse: update.ConcurrencyInUse,
	}
	switch update.Types[0] {
	case success:
		r.Successes = 1
	case failure:
		r.Failures = 1
		r.Errors = 1
	case rejected:
		r.Rejects = 1
		r.Errors = 1
	case shortCircuit:
		r.ShortCircuits = 1
		r.Errors = 1
	case timeout:
		r.Timeouts = 1
		r.Errors = 1
	case contextCanceled:
		r.ContextCanceled = 1
	case contextDeadlineExceeded:
		r.ContextDeadlineExceeded = 1
	}
	if len(update.Types) > 1 {
		// fallback metrics
		switch update.Types[1] {
		case fallbackSuccess:
			r.FallbackSuccesses = 1
		case fallbackFailure:
			r.FallbackFailures = 1
		}
	}
	collector.Update(r)
	wg.Done()
}

// Requests 返回当前请求数
func (m *metricExchange) Requests() *rolling.Number {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.requestsLocked()
}

// requestsLocked 返回当前请求数
func (m *metricExchange) requestsLocked() *rolling.Number {
	return m.defaultCollector().NumRequests()
}

// ErrorPercent 返回 metrics 错误率
func (m *metricExchange) errorPercent(now time.Time) int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	var errorPercent float64
	reqs := m.requestsLocked().Sum(now)
	errs := m.defaultCollector().Errors().Sum(now)
	// 防止分母为 0
	if reqs > 0 {
		errorPercent = (errs / reqs) * 100
	}
	// 加 0.5 四舍五入
	return int(errorPercent + 0.5)
}

// IsHealthy 判断到某一段时间是否健康
func (m *metricExchange) IsHealthy(now time.Time) bool {
	return m.errorPercent(now) < getSettings(m.Name).ErrorPercentThreshold
}

func (m *metricExchange) Reset() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for _, collector := range m.metricCollectors {
		collector.Reset()
	}
}
