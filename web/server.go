package web

import (
	"net/http"
)

func NewServer(addr string) {
	s := &http.Server{
		Addr:           addr,
		Handler:        myHandler,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

}
