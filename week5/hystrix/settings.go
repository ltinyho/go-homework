package hystrix

import (
	"sync"
	"time"
)

var (
	// DefaultTimeout is how long to wait for command to complete, in milliseconds
	DefaultTimeout = 1000
	// DefaultMaxConcurrent is how many commands of the same type can run at the same time
	DefaultMaxConcurrent = 10
	// DefaultVolumeThreshold is the minimum number of requests needed before a circuit can be tripped due to health
	DefaultVolumeThreshold = 20
	// DefaultSleepWindow is how long, in milliseconds, to wait after a circuit opens before testing for recovery
	DefaultSleepWindow = 5000
	// DefaultErrorPercentThreshold causes circuits to open once the rolling measure of errors exceeds this percent of requests
	DefaultErrorPercentThreshold = 50
	// DefaultLogger is the default logger that will be used in the Hystrix package. By default prints nothing.
	defaultLogger = &NoopLogger{}
)

type CommandConfig struct {
	Timeout                int // 等待命令完成的时间 单位:毫秒
	MaxConcurrentRequests  int // 最大请求数
	RequestVolumeThreshold int // 请求的阈值
	SleepWindow            int // 断路器开启后,超时检测回复需要等待的时间 单位: 毫秒
	ErrorPercentThreshold  int // 导致断路器打开时错误需占全部请求的比例
}

type Settings struct {
	Timeout                time.Duration
	MaxConcurrentRequests  int
	RequestVolumeThreshold uint64
	SleepWindow            time.Duration
	ErrorPercentThreshold  int
}

var (
	// 熔断器配置 key 为熔断器的 name
	circuitSettings map[string]*Settings
	// 熔断器配置读写锁
	settingsMutex *sync.RWMutex
	// 日志
	log logger
)

// 初始化变量
func init() {
	circuitSettings = make(map[string]*Settings)
	settingsMutex = &sync.RWMutex{}
	log = defaultLogger
}

// ConfigureCommand 设置断路器配置
func ConfigureCommand(name string, config CommandConfig) {
	settingsMutex.Lock()
	defer settingsMutex.Unlock()
	if config.Timeout == 0 {
		config.Timeout = DefaultTimeout
	}

	if config.MaxConcurrentRequests == 0 {
		config.MaxConcurrentRequests = DefaultMaxConcurrent
	}

	if config.RequestVolumeThreshold == 0 {
		config.RequestVolumeThreshold = DefaultVolumeThreshold
	}

	if config.SleepWindow == 0 {
		config.SleepWindow = DefaultSleepWindow
	}

	if config.ErrorPercentThreshold == 0 {
		config.ErrorPercentThreshold = DefaultErrorPercentThreshold
	}

	circuitSettings[name] = &Settings{
		Timeout:                time.Duration(config.Timeout) * time.Millisecond,
		MaxConcurrentRequests:  config.MaxConcurrentRequests,
		RequestVolumeThreshold: uint64(config.RequestVolumeThreshold),
		SleepWindow:            time.Duration(config.SleepWindow) * time.Millisecond,
		ErrorPercentThreshold:  config.ErrorPercentThreshold,
	}
}

func getSettings(name string) *Settings {
	settingsMutex.RLock()
	s, exits := circuitSettings[name]
	settingsMutex.RUnlock()
	if !exits { // 设置默认值
		ConfigureCommand(name, CommandConfig{})
		s = getSettings(name)
	}
	return s
}
