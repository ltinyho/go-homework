package main

import (
	"context"
	"os"
)

type Option func(o *options)

type options struct {
	servers []TransportServer
	ctx     context.Context
	sigs    []os.Signal
}

func Context(ctx context.Context) Option {
	return func(o *options) {
		o.ctx = ctx
	}
}


func Server(servers ...TransportServer) Option {
	return func(o *options) {
		o.servers = servers
	}
}
