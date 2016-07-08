package api

import (
	"net/http"

	"goji.io"

	"goji.io/pat"

	q "github.com/unchartedsoftware/rannu/cluster/queue"
)

var jobc = make(chan *q.Job)

// New return an multiplexer for API endpoints
func New() http.Handler {
	q.Listen(jobc)

	mux := goji.NewMux()
	mux.HandleFuncC(pat.Get("/api/pca/:dataset/:workers"), pcaHandler)

	return mux
}
