参考 Hystrix 实现一个滑动窗口计数器

> All of these represent failure and latency that needs to be isolated and managed so that a single failing dependency can’t take down an entire application or system

**防止局部故障导致全局故障.**

# 参考 hystrix-go
## Types
- CircuitBreaker 熔断器
- CommandConfig 命令参数配置,检测什么时候打开熔断器,什么时候关闭熔断器
- Settings 跟 CommandConfig 基本一致,将timeout等转成 time等.将 CommandConfig 转换成 Settings,通过内部 map 维护熔断器的配置
- CircuitError 熔断时的错误
- NoopLogger 实现了 log 接口的默认 logger


内部types:
- executorPool 计算最大并发
- 
## 主要的Functions

```go
// 设置熔断器配置
func ConfigureCommand(name string, config CommandConfig)
// 异步执行
func GoC(ctx context.Context, name string, run runFuncC, fallback fallbackFuncC) chan error

// 同步执行,在GoC的基础上封装成同步
func DoC(ctx context.Context, name string, run runFuncC, fallback fallbackFuncC) error
```





# 参考资料
- [hystrix-go](https://github.com/afex/hystrix-go)
- [Hystrix 官方wiki](https://github.com/netflix/hystrix/wiki)
- [Hystrix golang实践](https://www.bilibili.com/video/BV1Vt4y1Q747/)
- [hystrix-go 使用与原理](https://learnku.com/articles/53019)
- [一文彻底读懂 hystrix-go 源码](https://learnku.com/articles/53090)
