package main

import (
	"github.com/MarlikAlmighty/analyze-it/internal/botapi"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/MarlikAlmighty/analyze-it/internal/app"
	"github.com/MarlikAlmighty/analyze-it/internal/config"
	"github.com/MarlikAlmighty/analyze-it/internal/store"
)

func main() {

	var err error

	// got config
	cnf := config.New()
	if err = cnf.GetEnv(); err != nil {
		log.Fatalf("error config: %v\n", err)
	}

	// connect to store
	var r *store.Wrapper
	if r, err = r.New(); err != nil {
		log.Fatalf("error store: %v\n", err)
	}

	// init bot api
	var api *botapi.TgAPI
	if api, err = botapi.New(cnf, r); err != nil {
		log.Fatalf("error botAPI: %s\n", err)
	}

	// start bot
	go func() {
		if err = api.Run(); err != nil {
			log.Fatalf("error run botAPI: %s\n", err)
			return
		}
	}()

	// init and run core
	core := app.New(cnf, r)
	go core.Run()

	stopApp := make(chan os.Signal, 1)
	signal.Notify(stopApp, syscall.SIGKILL, syscall.SIGINT, syscall.SIGTERM)

	sig := <-stopApp
	log.Printf("Catch signal %s, exit app...", sig)
	core.Stop()
}
