package main

import (
	"errors"
	"io/ioutil"
	"os"
	"runtime"
	"strings"
	"sync"
	"net/http"

	"github.com/vpnishe/anyvalue"
	core "github.com/vpnishe/co_core"
)

const (
	CLIENT_STOPPED  = 0
	CLIENT_STARTING = 1
	CLIENT_STARTED  = 2
)

type RequestHandler struct {
	conn       Conn
	mutex      *sync.Mutex
	status     int
	server     *anyvalue.AnyValue
	client     *core.PoleVpnClient
	networkmgr core.NetworkManager
	device     *core.TunDevice
	buffer	   string
}

func NewRequestHandler() *RequestHandler {
	return &RequestHandler{mutex: &sync.Mutex{}, status: CLIENT_STOPPED}
}


func (rh *RequestHandler) onCallback(av *anyvalue.AnyValue) {
	pkt, _ := av.EncodeJson()
	if len(rh.buffer) == 0 {
		rh.buffer = "["
	}
	rh.buffer = rh.buffer + string(pkt) + ","
}

func (rh *RequestHandler) onCallbackRequest(av *anyvalue.AnyValue, w http.ResponseWriter) {
	pkt, _ := av.EncodeJson()
	w.Write(pkt)
}


func (rh *RequestHandler) OnClientEvent(event int, client *core.PoleVpnClient, av *anyvalue.AnyValue) {

	defer core.PanicHandler()
	switch event {
	case core.CLIENT_EVENT_ADDRESS_ALLOCED:
		{
			rh.mutex.Lock()
			defer rh.mutex.Unlock()

			rh.status = CLIENT_STARTED

			var err error
			var routes []string

			if rh.server.Get("UseRemoteRouteRules").AsBool() {
				routes = append(routes, av.Get("route").AsStrArr()...)
			}

			if rh.server.Get("LocalRouteRules").AsStr() != "" {
				routes = append(routes, strings.Split(rh.server.Get("LocalRouteRules").AsStr(), "\n")...)
			}

			if rh.server.Get("ProxyDomains").AsStr() != "" {
				ips := GetRouteIpsFromDomain(strings.Split(rh.server.Get("ProxyDomains").AsStr(), "\n"))
				routes = append(routes, ips...)
			}

			glog.Info("route=", routes, ",allocated ip=", av.Get("ip").AsStr(), ",dns=", av.Get("dns").AsStr())

			if runtime.GOOS == "windows" {
				err = rh.device.GetInterface().SetTunNetwork(av.Get("ip").AsStr() + "/30")
				if err != nil {
					glog.Error("set tun network fail,", err)
					client.Stop()
					return
				}
			}
			av.Set("remoteIp", client.GetRemoteIP())
			err = rh.networkmgr.SetNetwork(rh.device.GetInterface().Name(), av.Get("ip").AsStr(), client.GetRemoteIP(), av.Get("dns").AsStr(), routes)
			if err != nil {
				glog.Error("set network fail,", err)
				client.Stop()
				return
			}
			rh.onCallback(anyvalue.New().Set("event", "allocated").Set("data", av.AsMap()))
		}
	case core.CLIENT_EVENT_STOPPED:
		{
			glog.Info("client stoped")
			rh.networkmgr.RestoreNetwork()
			rh.onCallback(anyvalue.New().Set("event", "stoped").Set("data", nil))
			rh.status = CLIENT_STOPPED
		}
	case core.CLIENT_EVENT_RECONNECTED:
		glog.Info("client reconnected")
		rh.onCallback(anyvalue.New().Set("event", "reconnected").Set("data", nil))
	case core.CLIENT_EVENT_RECONNECTING:
		err := rh.networkmgr.RefreshDefaultGateway()
		if err != nil {
			glog.Error("refresh default gateway fail,", err)
		}
		glog.Info("client reconnecting")
		rh.onCallback(anyvalue.New().Set("event", "reconnecting").Set("data", nil))
	case core.CLIENT_EVENT_STARTED:
		glog.Info("client started")

		var err error
		rh.device, err = core.NewTunDevice()
		if err != nil {
			glog.Error("create device fail,", err)
			go client.Stop()
			return
		}

		client.AttachTunDevice(rh.device)
		rh.onCallback(anyvalue.New().Set("event", "started").Set("data", nil))
	case core.CLIENT_EVENT_ERROR:
		glog.Info("Unexception error,", av.Get("error").AsStr())
		rh.onCallback(anyvalue.New().Set("event", "error").Set("data", av.AsMap()))

	default:
		glog.Error("invalid event=", event)
	}

}

func (rh *RequestHandler) OnRequest(pkt []byte, w http.ResponseWriter) {
	
	defer core.PanicHandler()

	req, err := anyvalue.NewFromJson(pkt)

	if err != nil {
		glog.Error("decode json fail,", err)
		return
	}
	event := req.Get("event").AsStr()

	if event == "start" {
		err := rh.start(req.Get("data"))
		rh.onCallbackRequest(anyvalue.New().Set("event", "ok"), w)
		if err != nil {
			rh.onCallbackRequest(anyvalue.New().Set("event", "error").Set("data.error", err.Error()), w)
		}
	} else if event == "status" {
		if (len(rh.buffer)>0){
			s := rh.buffer
			s = s[:len(s)-1]+"]"	
			w.Write([]byte(s))
			rh.buffer=""
		} else {
			rh.onCallbackRequest(anyvalue.New().Set("event", "ok"), w)			
		}
	} else if event == "stop" {
		rh.onCallbackRequest(anyvalue.New().Set("event", "ok"), w)		
		rh.stop()
	} else if event == "network" {
			if rh.client != nil && !rh.client.IsStoped() {
				rh.onCallbackRequest(anyvalue.New().Set("event", "connected"), w)
			} else {
				rh.onCallbackRequest(anyvalue.New().Set("event", "stopped"), w)
			}
	} else if event == "getlogs" {
		logFilePath := ""
		if (len(glog.GetLogPath())==0){
			logFilePath = GetAppName() + "-" + GetTimeNowDate() + ".log"		
		} else {
			logFilePath = glog.GetLogPath() + string(os.PathSeparator) + GetAppName() + "-" + GetTimeNowDate() + ".log"
		}

		data, err := ioutil.ReadFile(logFilePath)

		if err != nil {
			glog.Error("read log fail,", err)
			return
		}
		rh.onCallbackRequest(anyvalue.New().Set("event", data), w)	

	} else if event == "getbytes" {
		var upBytes, downBytes uint64
		if rh.client != nil {
			upBytes, downBytes = rh.client.GetUpDownBytes()
		}
		rh.onCallbackRequest(anyvalue.New().Set("event", "bytes").Set("data.UpBytes", upBytes).Set("data.DownBytes", downBytes), w)
	} else {
		glog.Error("invalid event,", event)
	}

}

func (rh *RequestHandler) start(server *anyvalue.AnyValue) error {
	rh.mutex.Lock()
	defer rh.mutex.Unlock()

	if rh.status != CLIENT_STOPPED {
		return errors.New("client have started")
	}

	rh.status = CLIENT_STARTING

	glog.Info("Connect to ", server.Get("Endpoint").AsStr())

	rh.server = server

	var err error
	rh.client, err = core.NewPoleVpnClient()

	if err != nil {
		return err
	}

	deviceType := "Unknown"

	if runtime.GOOS == "darwin" {
		rh.networkmgr = core.NewDarwinNetworkManager()
		deviceType = "Macos"
	} else if runtime.GOOS == "linux" {
		rh.networkmgr = core.NewLinuxNetworkManager()
		deviceType = "Linux"
	} else if runtime.GOOS == "windows" {
		rh.networkmgr = core.NewWindowsNetworkManager()
		deviceType = "Windows"
	} else {
		return errors.New("os platform not support")
	}

	rh.client.SetEventHandler(rh.OnClientEvent)

	deviceId := GetDeviceId()

	go rh.client.Start(server.Get("Endpoint").AsStr(), server.Get("User").AsStr(), server.Get("Password").AsStr(), server.Get("Sni").AsStr(), server.Get("SkipVerifySSL").AsBool(), deviceType, deviceId)

	return nil
}

func (rh *RequestHandler) stop() error {
	rh.mutex.Lock()
	defer rh.mutex.Unlock()

	if rh.status != CLIENT_STARTED {
		return errors.New("client haven't started")
	}

	if rh.client != nil {
		rh.client.Stop()
	}
	rh.status = CLIENT_STOPPED
	return nil
}
