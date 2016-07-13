package main

import (
	"flag"
	"fmt"
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
	host = flag.CommandLine.String("host",
		"", "HTTP server host")
	port = flag.CommandLine.Int("addr",
		7900, "HTTP server port")
	addr1 = flag.CommandLine.String("addr1",
		"worker1:7901", "Worker 1 address")
	addr2 = flag.CommandLine.String("addr2",
		"worker2:7901", "Worker 2 address")
	addr3 = flag.CommandLine.String("addr3",
		"worker3:7901", "Worker 3 address")
	addr4 = flag.CommandLine.String("addr4",
		"worker4:7901", "Worker 4 address")
	addr5 = flag.CommandLine.String("addr5",
		"worker5:7901", "Worker 5 address")
	addr6 = flag.CommandLine.String("addr6",
		"worker6:7901", "Worker 6 address")
	addr7 = flag.CommandLine.String("addr7",
		"worker7:7901", "Worker 7 address")
	addr8 = flag.CommandLine.String("addr8",
		"worker8:7901", "Worker 8 address")
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

	addrs := []string{
		*addr1,
		*addr2,
		*addr3,
		*addr4,
		*addr5,
		*addr6,
		*addr7,
		*addr8,
	}
	apiMux, err := api.New(addrs)
	if err != nil {
		log.Fatal(err)
	}
	mux.Handle(pat.Get("/api/*"), apiMux)

	mux.Handle(pat.Get("/*"), http.FileServer(http.Dir("assets")))

	addr := fmt.Sprintf("%s:%d", *host, *port)
	log.Printf("Rannu server listening on %s", addr)
	http.ListenAndServe(addr, mux)
}
