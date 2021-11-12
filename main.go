package main

import (
	"net/http"

	"github.com/black-desk/goproxy"
)

func main() {
	http.ListenAndServe("localhost:8080", &goproxy.Goproxy{})
}
