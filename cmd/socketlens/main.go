package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/kamil-ginter/socketlens/internal/api"
	"github.com/kamil-ginter/socketlens/internal/store"
	"github.com/kamil-ginter/socketlens/internal/webui"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:8790", "HTTP listen address")
	dataDir := flag.String("data", "./data", "SocketLens data directory")
	flag.Parse()

	dataPath, err := filepath.Abs(*dataDir)
	if err != nil {
		log.Fatal(err)
	}

	db, err := store.Open(filepath.Join(dataPath, "socketlens.db"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	server := api.New(db, webui.Handler())

	httpServer := &http.Server{
		Addr:              *listen,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	fmt.Println("SocketLens 0.1.0")
	fmt.Printf("Dashboard: http://%s\n", *listen)
	fmt.Printf("Data: %s\n", dataPath)
	fmt.Println("Scope: private and loopback targets only")

	log.Fatal(httpServer.ListenAndServe())
}
