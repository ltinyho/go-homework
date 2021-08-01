package main

import (
	"context"
	"errors"
	"github.com/sirupsen/logrus"
	"net"
	"net/http"
)

type HttpOption func(opts *HttpServer)

type HttpServer struct {
	*http.Server
	addr   string
	log    *logrus.Entry
	router *http.ServeMux
}

func Addr(addr string) HttpOption {
	return func(s *HttpServer) {
		s.addr = addr
	}
}

func Log(logger *logrus.Entry) HttpOption {
	return func(s *HttpServer) {
		s.log = logger
	}
}

func NewServer(opts ...HttpOption) *HttpServer {
	srv := &HttpServer{
		log:  logrus.WithField("app", "server"),
		addr: ":8080",
	}

	for _, o := range opts {
		o(srv)
	}
	mux := http.NewServeMux()
	srv.router = mux
	srv.Server = &http.Server{
		Addr:    srv.addr,
		Handler: mux,
	}
	return srv
}

func (s *HttpServer) HandleFunc(path string, h func(http.ResponseWriter, *http.Request)) {
	s.router.HandleFunc(path, h)
}

func (s *HttpServer) Start(ctx context.Context) error {
	s.BaseContext = func(listener net.Listener) context.Context {
		return ctx
	}
	s.log.Infof("[HTTP] server listening on: %s", s.addr)
	if err := s.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *HttpServer) Stop(ctx context.Context) error {
	s.log.Info("stop")
	return s.Shutdown(ctx)
}
