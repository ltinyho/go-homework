package hystrix

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type CircuitBreaker struct {
	Name                   string
	open                   bool
	forceOpen              bool
	mutex                  *sync.RWMutex
	metrics                *metricExchange
	executorPool           *executorPool
	openedOrLastTestedTime int64 // 最新打开或者测试的时间 单位: 纳秒
}

var (
	circuitBreakersMutex *sync.RWMutex
	circuitBreakers      map[string]*CircuitBreaker
)

func init() {
	circuitBreakersMutex = &sync.RWMutex{}
	circuitBreakers = make(map[string]*CircuitBreaker)
}

// getCircuit 返回command对应的Circuit和是否是当前调用者创建的
func getCircuit(name string) (*CircuitBreaker, bool) {
	circuitBreakersMutex.RLock()
	_, ok := circuitBreakers[name]
	if !ok {
		circuitBreakersMutex.RUnlock()
		circuitBreakersMutex.Lock()
		defer circuitBreakersMutex.Unlock()
		// 为什么要再次检查呢?
		// RUnlock已经释放了,防止其他goroutine又创建了circuit
		if cb, ok2 := circuitBreakers[name]; ok2 {
			return cb, false
		}
		circuitBreakers[name] = newCircuitBreaker(name)
	} else {
		// 为什么使用 defer释放锁呢?
		// 因为返回时使用circuitBreakers[name]返回circuit,防止并发读写
		defer circuitBreakersMutex.RUnlock()
	}
	return circuitBreakers[name], !ok
}

// newCircuitBreaker 创建熔断器,关联健康检查
func newCircuitBreaker(name string) *CircuitBreaker {
	c := &CircuitBreaker{}
	c.Name = name
	c.metrics = newMetricExchange(name)
	c.executorPool = newExecutorPool(name)
	c.mutex = &sync.RWMutex{}
	return c
}

// toggleForceOpen 强制开启或关闭熔断器,主要用来写测试
func (circuit *CircuitBreaker) toggleForceOpen(toggle bool) {
	c, _ := getCircuit(circuit.Name)
	c.forceOpen = toggle
}

// IsOpen 熔断器是否打开
func (circuit *CircuitBreaker) IsOpen() bool {
	circuit.mutex.RLock()
	o := circuit.forceOpen || circuit.open
	circuit.mutex.RUnlock()
	if o {
		return true
	}
	// 判断请求数是否达到阈值
	if uint64(circuit.metrics.Requests().Sum(time.Now())) < getSettings(circuit.Name).
		RequestVolumeThreshold {
		return false
	}

	// 判断指标是否正常
	if !circuit.metrics.IsHealthy(time.Now()) {
		// 太多错误,打开熔断器
		circuit.setOpen()
		return true
	}
	return false
}

// AllowRequest 在命令执行之前检测,确保熔断器的状态和健康指标允许执行
// 当熔断器打开时,尝试探测服务是否恢复
func (circuit *CircuitBreaker) AllowRequest() bool {
	return !circuit.IsOpen() || circuit.allowSingleTest()
}

// allowSingleTest 是否允许释放一个流量测试
func (circuit *CircuitBreaker) allowSingleTest() bool {
	circuit.mutex.RLock()
	defer circuit.mutex.RUnlock()
	now := time.Now().UnixNano()
	openedOrLastTestedTime := atomic.LoadInt64(&circuit.openedOrLastTestedTime)
	sleep := getSettings(circuit.Name).SleepWindow.Nanoseconds()
	if circuit.open && now > openedOrLastTestedTime+sleep {
		swapped := atomic.CompareAndSwapInt64(&circuit.openedOrLastTestedTime, openedOrLastTestedTime, now)
		if swapped {
			log.Printf("放一个流量测试服务是否恢复 circuit %v", circuit.Name)
		}
		return swapped
	}
	return false
}

// setOpen 打开熔断器,限制流量进入
func (circuit *CircuitBreaker) setOpen() {
	circuit.mutex.Lock()
	defer circuit.mutex.Unlock()
	if circuit.open {
		return
	}
	circuit.openedOrLastTestedTime = time.Now().UnixNano()
	circuit.open = true
}

// setClose 关闭熔断器,放流量进来
func (circuit *CircuitBreaker) setClose() {
	circuit.mutex.Lock()
	defer circuit.mutex.Unlock()
	if !circuit.open {
		return
	}
	circuit.open = false
	circuit.metrics.Reset() // 重置统计
}

// reportEvent 统计
func (circuit *CircuitBreaker) reportEvent(eventTypes []event, start time.Time, runDuration time.Duration) error {
	if len(eventTypes) == 0 {
		return fmt.Errorf("no event types send for metrics")
	}

	circuit.mutex.RLock()
	o := circuit.open
	circuit.mutex.RUnlock()
	// 请求成功,关闭熔断器
	if eventTypes[0] == success && o {
		circuit.setClose()
	}

	concurrencyInUse := circuit.executorPool.ConcurrencyInUse()

	select {
	case circuit.metrics.Updates <- &commandExecution{
		Types:            eventTypes,
		Start:            start,
		RunDuration:      runDuration,
		ConcurrencyInUse: concurrencyInUse,
	}:
	default:
		// channel 已满
		return fmt.Errorf("metrics channel (%v) is at capacity", circuit.Name)
	}
	return nil
}
