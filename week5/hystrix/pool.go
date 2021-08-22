package hystrix

type executorPool struct {
	Name    string
	Max     int
	Tickets chan *struct{} // 剩余 tickets 最大容量为Max的chan
}

func newExecutorPool(name string) *executorPool {
	p := &executorPool{}
	p.Name = name
	p.Max = getSettings(name).MaxConcurrentRequests
	p.Tickets = make(chan *struct{}, p.Max)
	// 初始化 tickets
	for i := 0; i < p.Max; i++ {
		p.Tickets <- &struct{}{}
	}
	return p
}

func (p *executorPool) Return(ticket *struct{}) {
	if ticket == nil {
		return
	}
	p.Tickets <- ticket
}

// ActiveCount 正在执行的命令
func (p *executorPool) ActiveCount() int {
	return p.Max - len(p.Tickets)
}

// ConcurrencyInUse 正在执行的命令
func (p *executorPool) ConcurrencyInUse() float64 {
	var res float64
	if p.Max > 0 {
		res = float64(p.ActiveCount()) / float64(p.Max)
	}
	return res
}
