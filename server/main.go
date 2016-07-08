package main

import (
	"flag"
	"html/template"
	"log"
	"net/http"
	"runtime"

	"goji.io"

	"goji.io/pat"

	"github.com/unchartedsoftware/rannu/server/api"
)

var (
	tmpl *template.Template
	addr = flag.CommandLine.String("addr",
		":7900", "<address>:<port> to bind HTTP server")
)

func indexHandler(w http.ResponseWriter, r *http.Request) {
	err := tmpl.Execute(w, nil)
	if err != nil {
		log.Println("Failed to execute template:", err)
		http.Error(w, "Failed to execute template", http.StatusInternalServerError)
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	flag.Parse()

	var err error
	tmpl, err = template.ParseFiles("templates/index.tmpl")
	if err != nil {
		log.Fatal(err)
	}

	mux := goji.NewMux()
	mux.HandleFunc(pat.Get("/"), indexHandler)

	apiMux := api.New()
	mux.Handle(pat.Get("/api/*"), apiMux)

	mux.Handle(pat.Get("/*"), http.FileServer(http.Dir("assets")))

	log.Printf("Rannu server listening on %s", *addr)
	http.ListenAndServe(*addr, mux)
}
