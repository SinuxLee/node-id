package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"nodeid/internal/app"
	"nodeid/pkg/log"
)

func main() {
	log.Init()
	srv, err := app.New(
		app.Conf(),
		app.LogLevel(),
		app.Named(),
		app.Dao(),
		app.UseCase(),
		app.Controller(),
		app.Router(),
		app.PProf(),
		app.HTTPServer(),
	)

	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return
	}

	log.Info().Msg("app init success")

	// 监听信号
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	_ = srv.Run(ch)

	<-ch
}
