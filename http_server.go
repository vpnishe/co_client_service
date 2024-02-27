package main

import (
	"net/http"
	"fmt"	

	//core "github.com/vpnishe/co_core"
)

const (
	TCP_WRITE_BUFFER_SIZE = 524288
	TCP_READ_BUFFER_SIZE  = 524288
)

type HttpServer struct {
	handler  ProcessHandler
}

func NewHttpServer(handler ProcessHandler) *HttpServer {
	return &HttpServer{handler: handler}
}

func (hs *HttpServer) defaultHandler(w http.ResponseWriter, r *http.Request) {
	hs.respError(http.StatusForbidden, w)
}

func (hs *HttpServer) check(w http.ResponseWriter, r *http.Request) {
	version := r.URL.Query().Get("version")
	if version != VERSION {
		glog.Fatal("version not equal,", version, ",", VERSION)
	} else {
		w.Write([]byte("ok"))
	}
}

func (hs *HttpServer) getQuery(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		fmt.Fprintf(w, "ParseForm() err: %v", err)
		return
	}
	hs.handler.OnRequest([]byte(r.FormValue("data")), w)		
}

func (hs *HttpServer) Listen(addr string) error {

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if r.URL.Path == "/" {		
			hs.getQuery(w, r)
		} else if r.URL.Path == "/check" {
			hs.check(w, r)
		} else {
			hs.defaultHandler(w, r)
		}
	})
	return http.ListenAndServe(addr, handler)
}

func (hs *HttpServer) respError(status int, w http.ResponseWriter) {
	if status == http.StatusBadRequest {
		w.Header().Add("Server", "nginx/1.10.3")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("<html>\n<head><title>400 Bad Request</title></head>\n<body bgcolor=\"white\">\n<center><h1>400 Bad Request</h1></center>\n<hr><center>nginx/1.10.3</center>\n</body>\n</html>"))
	} else if status == http.StatusForbidden {
		w.Header().Add("Server", "nginx/1.10.3")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("<html>\n<head><title>403 Forbidden</title></head>\n<body bgcolor=\"white\">\n<center><h1>403 Forbidden</h1></center>\n<hr><center>nginx/1.10.3</center>\n</body>\n</html>"))

	}
}


