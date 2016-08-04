package api

import (
	"net/http"

	"goji.io"

	"goji.io/pat"

	q "github.com/unchartedsoftware/rannu/cluster/queue"
)

var jobc = make(chan *q.Job)

// New return an multiplexer for API endpoints
func New(addrs []string) (http.Handler, error) {
	if err := q.Listen(addrs, jobc); err != nil {
		return nil, err
	}

	mux := goji.NewMux()
	mux.HandleFuncC(pat.Get("/api/pca/:dataset/:workers/:standardize"), pcaHandler)

	return mux, nil
}
