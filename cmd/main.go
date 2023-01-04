package main

import (
	"log"
	"microblog/internal/repo"
	"microblog/internal/service"
	"os"
)

const (
	modeServer = "SERVER"
	modeWorker = "WORKER"
)

func main() {
	var r repo.Repository

	mode, ok := os.LookupEnv("APP_MODE")
	if !ok {
		mode = modeServer
	}

	r = repo.NewRedisRepository(repo.NewMongoDatabaseRepository())

	switch mode {
	case modeServer:
		srv, err := service.NewServer(r)
		if err != nil {
			log.Fatal(err.Error())
		}

		log.Printf("Start serving HTTP at %s", srv.Addr)
		log.Fatal(srv.ListenAndServe())
	case modeWorker:
		log.Fatal(service.StartConsumer(r))
	default:
		log.Fatalf("Unexpected mode flag: %s", mode)
	}
}
