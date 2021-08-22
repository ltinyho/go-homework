package hystrix

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type runFuncC func(context.Context) error
type fallbackFuncC func(context.Context, error) error

// A CircuitError is an error which models various failure states of execution,
// such as the circuit being open or a timeout.
type CircuitError struct {
	Message string
}

func (e CircuitError) Error() string {
	return "hystrix: " + e.Message
}

type command struct {
	sync.Mutex
	ticket      *struct{}
	start       time.Time
	run         runFuncC
	fallback    fallbackFuncC
	circuit     *CircuitBreaker
	finished    chan bool
	errChan     chan error
	events      []event
	runDuration time.Duration
}

var (
	// ErrMaxConcurrency occurs when too many of the same named command are executed at the same time.
	ErrMaxConcurrency = CircuitError{Message: "max concurrency"}
	// ErrCircuitOpen returns when an execution attempt "short circuits". This happens due to the circuit being measured as unhealthy.
	ErrCircuitOpen = CircuitError{Message: "circuit open"}
	// ErrTimeout occurs when the provided function takes too long to execute.
	ErrTimeout = CircuitError{Message: "timeout"}
)

func GoC(ctx context.Context, name string, run runFuncC, fallback fallbackFuncC) chan error {
	cmd := &command{
		run:      run,
		fallback: fallback,
		start:    time.Now(),
		errChan:  make(chan error, 1),
		finished: make(chan bool, 1),
	}

	circuit, _ := getCircuit(name)
	cmd.circuit = circuit
	ticketCond := sync.NewCond(cmd)
	ticketChecked := false
	returnTicket := func() {
		cmd.Lock()
		for !ticketChecked {
			ticketCond.Wait()
		}
		cmd.circuit.executorPool.Return(cmd.ticket)
		cmd.Unlock()
	}
	returnOnce := &sync.Once{}
	reportAllEvent := func() {
		err := cmd.circuit.reportEvent(cmd.events, cmd.start, cmd.runDuration)
		if err != nil {
			log.Printf(err.Error())
		}
	}

	go func() {
		defer func() { cmd.finished <- true }()
		if !cmd.circuit.AllowRequest() {
			// 熔断器打开不允许放流量
			cmd.Lock()
			ticketChecked = true
			ticketCond.Signal()
			cmd.Unlock()
			returnOnce.Do(func() {
				returnTicket()
				cmd.errorWithFallback(ctx, ErrCircuitOpen)
				reportAllEvent()
			})
			return
		}
		cmd.Lock()
		select {
		case cmd.ticket = <-circuit.executorPool.Tickets:
			ticketChecked = true
			ticketCond.Signal()
			cmd.Unlock()
		default:
			// 达到最大并发量
			ticketChecked = true
			ticketCond.Signal()
			cmd.Unlock()
			returnOnce.Do(func() {
				returnTicket()
				cmd.errorWithFallback(ctx, ErrMaxConcurrency)
				reportAllEvent()
			})
			return
		}

		// 允许执行
		runStart := time.Now()
		runErr := run(ctx)
		returnOnce.Do(func() {
			defer reportAllEvent()
			cmd.runDuration = time.Since(runStart)
			returnTicket()
			if runErr != nil {
				cmd.errorWithFallback(ctx, runErr)
				return
			}
			cmd.reportEvent(success)
		})
	}()

	go func() {
		// 超时处理
		timer := time.NewTimer(getSettings(name).Timeout)
		defer timer.Stop()
		select {
		case <-cmd.finished:
			// returnOnce 已经在其他 goroutine 执行过
		case <-ctx.Done():
			// context完成,归还ticket,报错,报告event
			returnOnce.Do(func() {
				returnTicket()
				cmd.errorWithFallback(ctx, ctx.Err())
				reportAllEvent()
			})
		case <-timer.C:
			// 超时,归还ticket,报错,报告event
			returnOnce.Do(func() {
				returnTicket()
				cmd.errorWithFallback(ctx, ErrTimeout)
				reportAllEvent()
			})
		}
	}()
	return cmd.errChan
}

func (c *command) reportEvent(e event) {
	c.Lock()
	defer c.Unlock()
	c.events = append(c.events, e)
}

// errorWithFallback triggers the fallback while reporting the appropriate metric events.
func (c *command) errorWithFallback(ctx context.Context, err error) {
	eventType := failure
	if err == ErrCircuitOpen {
		eventType = shortCircuit
	} else if err == ErrMaxConcurrency {
		eventType = rejected
	} else if err == ErrTimeout {
		eventType = timeout
	} else if err == context.Canceled {
		eventType = contextCanceled
	} else if err == context.DeadlineExceeded {
		eventType = contextDeadlineExceeded
	}

	c.reportEvent(eventType)
	fallbackErr := c.tryFallback(ctx, err)
	if fallbackErr != nil {
		c.errChan <- fallbackErr
	}
}

func (c *command) tryFallback(ctx context.Context, err error) error {
	if c.fallback == nil {
		// If we don't have a fallback return the original error.
		return err
	}

	fallbackErr := c.fallback(ctx, err)
	if fallbackErr != nil {
		c.reportEvent(fallbackFailure)
		return fmt.Errorf("fallback failed with '%v'. run error was '%v'", fallbackErr, err)
	}

	c.reportEvent(fallbackSuccess)

	return nil
}
