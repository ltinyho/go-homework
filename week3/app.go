package main

import (
	"context"
	"errors"
	"golang.org/x/sync/errgroup"
	"os"
	"os/signal"
	"syscall"
)



type App struct {
	opts   options
	ctx    context.Context
	cancel func()
}

func New(opts ...Option) *App {
	appOpts := options{
		ctx:  context.Background(),
		sigs: []os.Signal{syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT},
	}
	for _, o := range opts {
		o(&appOpts)
	}
	ctx, cancel := context.WithCancel(appOpts.ctx)
	return &App{
		opts:   appOpts,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (a *App) Run() error {
	eg, ctx := errgroup.WithContext(a.ctx)
	for _, srv := range a.opts.servers {
		eg.Go(func() error {
			<-ctx.Done() // 等待调用a.Stop 方法取消 a.ctx,退出
			return srv.Stop(ctx)
		})
		eg.Go(func() error {
			return srv.Start(ctx)
		})
	}
	c := make(chan os.Signal, 1)
	signal.Notify(c, a.opts.sigs...)
	eg.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-c:
				a.Stop()
			}
		}
	})
	if err := eg.Wait(); err != nil && errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}
func (a *App) Stop() error {
	if a.cancel != nil {
		a.cancel()
	}
	return nil
}

