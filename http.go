package main

import (
	"log"
	"net/http"
	"path/filepath"
)

func httpServer() {

	server := http.FileServer(http.Dir("www"))
	abs, _ := filepath.Abs("www")
	log.Println("[WWW   ] See", abs)
	log.Fatal(http.ListenAndServe(":8080", server))
}
