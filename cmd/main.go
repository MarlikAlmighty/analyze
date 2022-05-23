package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/MarlikAlmighty/analyze-it/internal/app"
	"github.com/MarlikAlmighty/analyze-it/internal/config"
	"github.com/MarlikAlmighty/analyze-it/internal/store"
)

func main() {

	cnf := config.New()
	if err := cnf.GetEnv(); err != nil {
		log.Fatalf("get environment keys: %v\n", err)
	}

	s := store.New()
	r, err := s.Connect(cnf.RedisUrl)
	if err != nil {
		log.Fatalf("connect: %v\n", err)
	}
	s.Client = r

	stopApp := make(chan os.Signal, 1)
	signal.Notify(stopApp, syscall.SIGKILL, syscall.SIGINT, syscall.SIGTERM)

	core := app.New(cnf, s)
	go core.Run()

	sig := <-stopApp
	log.Printf("Catch signal %s, exit app...", sig)
	core.Stop()
}
