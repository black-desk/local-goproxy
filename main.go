package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

var gopath = "/usr/share/gocode/src"

func main() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	dir, err := os.MkdirTemp("", "goproxy")
	if err != nil {
		log.Fatal(fmt.Errorf("fail to create go proxy dir: %w", err))
	}

	defer func() {
		err = os.RemoveAll(dir)
		if err != nil {
			log.Print(fmt.Errorf("fail to remove go proxy dir: %w", err))
		}
	}()

	log.Printf("goproxy path: %s", dir)

	p := proxy{
		dir:    filepath.Join(dir),
		gopath: gopath,
	}

	http.Handle("/cache/", http.FileServer(http.Dir(dir)))
	http.HandleFunc("/mod/", p.handle)

	go http.ListenAndServe(":8080", nil)
	<-c
}
