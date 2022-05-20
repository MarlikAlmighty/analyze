package main

import (
	"github.com/MarlikAlmighty/analyze-it/internal/app"
	"github.com/MarlikAlmighty/analyze-it/internal/config"
	"github.com/MarlikAlmighty/analyze-it/internal/store"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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

	stopServer := make(chan os.Signal, 1)
	signal.Notify(stopServer, syscall.SIGKILL, syscall.SIGINT, syscall.SIGTERM)

	router := mux.NewRouter()
	router.HandleFunc("/", homeHandler)

	srv := &http.Server{
		Addr:    ":" + os.Getenv("PORT"),
		Handler: router,
	}

	core := app.New(cnf, s, srv)
	go core.Run()

	log.Println("http: Server start")

	go func() {
		if err = core.Server.ListenAndServe(); err != nil {
			log.Fatalln(err)
		}
	}()

	sig := <-stopServer
	log.Printf("Catch signal %s, exit app...", sig)
	core.Stop()
}

func homeHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(200)
	if _, err := w.Write([]byte("I'm alive!")); err != nil {
		log.Fatalln(err)
	}
}
