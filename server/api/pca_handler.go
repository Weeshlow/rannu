package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"goji.io/pat"

	"golang.org/x/net/context"

	q "github.com/unchartedsoftware/rannu/cluster/queue"
)

func pcaHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	dataset := pat.Param(ctx, "dataset")
	workers, err := strconv.Atoi(pat.Param(ctx, "workers"))
	var standardize bool
	if pat.Param(ctx, "standardize") == "true" {
		standardize = true
	} else {
		standardize = false
	}
	log.Println(dataset, workers, standardize)
	if err != nil {
		log.Printf("Could not parse workers param: %s", pat.Param(ctx, "workers"))
		http.Error(w, "Could not parse workers param", http.StatusInternalServerError)
		return
	}
	if workers != 1 && workers != 2 && workers != 4 && workers != 8 {
		log.Printf("Invalid number of workers: %d", workers)
		http.Error(w, "Invalid number of workers", http.StatusInternalServerError)
		return
	}

	respc := make(chan *q.Response)
	job := &q.Job{
		Dataset:         dataset,
		Workers:         workers,
		Standardize:     standardize,
		ResponseChannel: respc,
	}
	jobc <- job

	resp := <-respc

	body, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Unable to marshal response: %v", err)
		http.Error(w, "Unable to marshal response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, string(body))
}
