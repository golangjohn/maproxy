package proxy

import (
	"flag"
	"os"
	"io/ioutil"
	"log"
	"encoding/json"
)

var proxyConfig struct {
	Proxies      []LocalRemote `json:"proxies"`
	ConnTimeout  int           `json:"conn_timeout"`
	BufferSize   int           `json:"buffer_size"`
	MaxConnOneIP int           `json:"max_conn_one_ip"`
}

type LocalRemote struct {
	Local  string `json:"local"`
	Remote string `json:"remote"`
	ProxyName string `json:"proxy_name"`
}

var CONF = flag.String("conf", "proxy.json", "setup proxy config file")

func LoadConfig() {
	flag.Parse()
	f, err := os.Open(*CONF)
	defer f.Close()
	if err != nil {
		log.Panicf("Load config err %s", err.Error())
	}
	data, err := ioutil.ReadAll(f)
	if err != nil {
		log.Panicf("Read config err %s", err.Error())
	}
	err = json.Unmarshal(data, &proxyConfig)
	if err != nil {
		log.Panicf("Config to json err %s ", err.Error())
	}
}
