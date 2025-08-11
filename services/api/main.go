package main

import (
	"log"
	"net/http"
	"os"

	"github.com/mk-slmn/booksmart/services/api/handlers"
)

func main() {
	db, err := handlers.OpenDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	r := handlers.NewServer(db)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8787"
	}

	log.Printf("listening on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}
