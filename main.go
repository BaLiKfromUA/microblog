package main

import (
	"log"
	"microblog/repo"
	"microblog/service"
)

func main() {
	r := repo.NewInMemoryRepository()
	srv := service.NewServer(r)
	log.Printf("Start serving HTTP at %s", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}
