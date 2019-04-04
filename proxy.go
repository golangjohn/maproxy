package proxy

import (
	"os"
	"net"
	"time"
	"sync"
	"strings"
	"runtime/debug"
	"slotserver/mlog"
)

// Notice: a service must has been started on toport
var record *recordMap

type recordMap struct {
	sync.RWMutex
	data map[string]int
}

func (rmp *recordMap) store(key string, v int) {
	rmp.Lock()
	defer rmp.Unlock()
	rmp.data[key] = v
}

func (rmp *recordMap) load(key string) int {
	rmp.Lock()
	defer rmp.Unlock()
	v, _ := rmp.data[key]
	return v
}

func Start() {
	run := make(chan struct{})
	for _, cfg := range proxyConfig.Proxies {
		go RunProxy(cfg)
	}
	<-run
}

func RunProxy(cfg LocalRemote) {
	local, remoteDest := cfg.Local, cfg.Remote
	record := new(recordMap)
	record.data = make(map[string]int)
	ln, err := net.Listen("tcp", local)
	if err != nil {
		mlog.ErrorF("Unable to listen on: %s, error: %s", local, err.Error())
		os.Exit(1)
	}
	mlog.InfoF("Proxy %s listen on: %s, proxy remote : %s", cfg.ProxyName, local, remoteDest)
	defer ln.Close()

	for {
		local, err := ln.Accept()
		// check max conn for per IPAddr
		ipport := strings.Split(local.RemoteAddr().String(), ":")
		if len(ipport) != 2 {
			mlog.ErrorF("un_support remote addr :: %s ", local.RemoteAddr().String())
			local.Close()
			continue
		}
		max := record.load(ipport[0])
		// 如果连接数大于每个IP允许的最大连接数，拒绝连接
		if max >= proxyConfig.MaxConnOneIP {
			mlog.ErrorF("too much connection for ip:: %s ", ipport[0])
			local.Close()
			continue
		}
		if err != nil {
			mlog.InfoF("Unable to accept a request, error: %s", err.Error())
			continue
		}
		// handle
		go func(local net.Conn) {
			defer func() {
				r := recover()
				if r != nil {
					mlog.ErrorF("panic in handle func ::%s", debug.Stack())
				}
			}()
			// Read a header firstly in case you could have opportunity to check request
			// whether to decline or proceed the request
			buffer := make([]byte, proxyConfig.BufferSize)
			if proxyConfig.ConnTimeout > 0 {
				local.SetDeadline(time.Now().Add(time.Second * time.Duration(proxyConfig.ConnTimeout)))
			}
			n, err := local.Read(buffer)
			if err != nil {
				mlog.InfoF("Unable to read from input %s, error: %s", local.RemoteAddr(), err.Error())
				return
			}

			remote, err := net.Dial("tcp", remoteDest)
			if err != nil {
				mlog.InfoF("Unable to connect to: %s, error: %s", remoteDest, err.Error())
				local.Close()
				return
			}

			n, err = remote.Write(buffer[:n])
			if err != nil {
				mlog.InfoF("Unable to write to output, error: %s", err.Error())
				local.Close()
				remote.Close()
				return
			}

			go ioCopy(local, remote, func(local net.Conn) {
				defer func() {
					r := recover()
					if r != nil {
						mlog.ErrorF("panic onEnd func ::%s", debug.Stack())
					}
				}()
				// 断线处理
				ip := strings.Split(local.RemoteAddr().String(), ":")[0]
				now := record.load(ip)
				if now > 0 {
					record.store(ip, now-1)
				}
				mlog.InfoF("断线，当前IP %s, 连接数 %d", ip, now-1)
			})
			go ioCopy(remote, local, nil)
			// 转发正常，更新IP连接数量
			ip := ipport[0]
			ipV := max + 1
			record.store(ip, ipV)
			mlog.InfoF("新连接，当前IP %s, 连接数 %d", ip, ipV)
		}(local)
	}
}

// Forward all requests from r to w
func ioCopy(r net.Conn, w net.Conn, onEnd func(r net.Conn)) {
	defer func() {
		w.Close()
		r.Close()
		if onEnd != nil {
			onEnd(r)
		}
		r := recover()
		if r != nil {
			mlog.ErrorF("panic ioCopy func ::%s", debug.Stack())
		}
	}()
	var buffer = make([]byte, proxyConfig.BufferSize)
	for {
		if proxyConfig.ConnTimeout > 0 {
			r.SetDeadline(time.Now().Add(time.Second * time.Duration(proxyConfig.ConnTimeout)))
		}
		n, err := r.Read(buffer)
		if err != nil {
			mlog.InfoF("read from %s, error: %s", r.RemoteAddr(), err.Error())
			break
		}
		if proxyConfig.ConnTimeout > 0 {
			w.SetDeadline(time.Now().Add(time.Second * time.Duration(proxyConfig.ConnTimeout)))
		}
		n, err = w.Write(buffer[:n])
		if err != nil {
			mlog.InfoF("write to %s, error: %s", w.RemoteAddr(), err.Error())
			break
		}
	}
}
