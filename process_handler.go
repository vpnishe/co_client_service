package main

import
	"net/http"

type ProcessHandler interface {
	OnRequest(pkg []byte, w http.ResponseWriter)
}
