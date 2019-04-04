package proxy

import (
	"testing"
	"net"
	"log"
	"time"
)

func TestStart(t *testing.T) {
	connected := 0
	failed := 0
	for {
		c, err := net.Dial("tcp","192.168.0.139:1090")
		if err != nil {
			log.Println(err)
			failed ++
		}else {
			connected++
			c.Write([]byte("ok"))
		}
		log.Printf("已连接 %d, 失败 %d", connected, failed)
		time.Sleep(time.Millisecond * 50)
		if failed >0 {
			break
		}
	}
}
