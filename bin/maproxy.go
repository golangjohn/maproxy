package main

import "slotserver/proxy"

func main() {
	proxy.LoadConfig()
	proxy.Start()
}
