package main

import (
	"context"
	"fmt"
	"github.com/ltinyho/go-homework/week5/hystrix"
	"net/http"
	"time"
)

type Handle struct{}

func (h *Handle) ServeHTTP(r http.ResponseWriter, request *http.Request) {
	h.Common(r, request)
}

func (h *Handle) Common(r http.ResponseWriter, request *http.Request) {
	hystrix.ConfigureCommand("mycommand", hystrix.CommandConfig{
		Timeout:                int(3 * time.Second),
		MaxConcurrentRequests:  10,
		SleepWindow:            5000,
		RequestVolumeThreshold: 20,
		ErrorPercentThreshold:  30,
	})
	msg := "success"
	output := make(chan bool, 1)
	errChan := hystrix.GoC(context.Background(), "mycommand", func(ctx context.Context) error {
		_, err := http.Get("https://www.baidu.com")
		if err != nil {
			fmt.Printf("请求失败:%v", err)
			return err
		}
		output <- true
		return nil
	}, func(ctx context.Context, err error) error {
		fmt.Printf("handle  error:%v\n", err)
		msg = "error"
		return nil
	})
	select {
	case out := <-output:
		fmt.Println("out success", out)
	case err := <-errChan:
		fmt.Println("errchan ", err)
	}
	r.Write([]byte(msg))
}

func main() {
	http.ListenAndServe(":8888", &Handle{})
}
