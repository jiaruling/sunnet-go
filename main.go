// +build linux

package main

import (
	"os"
	"os/signal"
	"syscall"

	. "Sunnet/core"
)

func main() {
	var quit = make(chan os.Signal)
	Inst.Start()
	sid := Inst.NewService("gateway", 10)
	Inst.Listen("0.0.0.0",8888, sid)
	// 优雅退出
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
}