// main.go

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"golang.org/x/net/http2"
)

func main() {
	port := flag.Int("p", 8080, "Port number to run the server on")
	useHTTP2 := flag.Bool("http2", true, "Enable HTTP/2")
	flag.Parse()

	http.HandleFunc("/", handler)
	http.HandleFunc("/static/", handleStaticFile)

	if *useHTTP2 {
		http2Enabled := &http.Server{
			Addr:    fmt.Sprintf(":%d", *port),
			Handler: nil,
		}
		http2.ConfigureServer(http2Enabled, &http2.Server{})
		log.Printf("Starting server with HTTP/2 on http://localhost:%d\n", *port)
		log.Fatal(http2Enabled.ListenAndServe())
		return
	}

	fmt.Printf("Starting server on http://localhost:%d\n", *port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}
