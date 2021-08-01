package main

import (
	"log"
	"net/http"
)

func main() {
	srv := NewServer(Addr(":333"))
	srv.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})
	app := New(Server(srv))
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
