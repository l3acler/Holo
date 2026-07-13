package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
)

func main() {
	var (
		filePath   = flag.String("file", "db.json", "Path to the source JSON file")
		port       = flag.String("port", "8080", "HTTP port to listen on")
		memoryOnly = flag.Bool("memory-only", false, "If true, mutations are not written back to the file")
	)
	flag.Parse()

	store, err := NewStore(*filePath, *memoryOnly)
	if err != nil {
		log.Fatalf("Failed to initialize store: %v", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/{resource}", handleCollection(store))
	mux.HandleFunc("/{resource}/{id}", handleItem(store))

	handler := withLogger(withCORS(mux))

	addr := fmt.Sprintf(":%s", *port)

	fmt.Println("=====================================")
	fmt.Println(" 🌌 HOLO - Zero-Config Mock Server ")
	fmt.Println("=====================================")
	fmt.Printf(" 🚀 Port        : %s\n", *port)
	fmt.Printf(" 📁 Database    : %s\n", *filePath)
	if *memoryOnly {
		fmt.Printf(" 🧠 Mode        : Memory-Only\n")
	} else {
		fmt.Printf(" 💾 Mode        : File Persistence\n")
	}
	fmt.Println("=====================================")
	fmt.Printf("Listening on http://localhost%s\n\n", addr)

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
