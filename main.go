package main

import (
	"log"
	"microblog/repo"
	"microblog/service"
	"os"
)

const (
	modeInMemory = "inmemory"
	modeMongo    = "mongo"
	modeCache    = "cached"
)

func main() {
	var r repo.Repository

	mode, ok := os.LookupEnv("STORAGE_MODE")
	if !ok {
		mode = modeMongo
	}

	switch mode {
	case modeInMemory:
		r = repo.NewInMemoryRepository()
	case modeMongo:
		r = repo.NewMongoDatabaseRepository()
	case modeCache:
		r = repo.NewRedisRepository(repo.NewMongoDatabaseRepository())
	default:
		log.Fatalf("Unexpected mode flag: %s", mode)
	}

	srv := service.NewServer(r)
	log.Printf("Start serving HTTP at %s", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}
