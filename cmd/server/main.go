package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/hoozecn/proxier"
)

func main() {
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGTERM, syscall.SIGINT)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-done
		cancel()
	}()

	os.Setenv("PROXIER_ETCD_ENDPOINTS", "http://10.211.55.78:2379")
	os.Setenv("PROXIER_SELF_IP", "127.0.0.1")
	os.Setenv("PROXIER_HTTP_PORT", "11087")
	proxier.RegisterAndServe(ctx)
}
