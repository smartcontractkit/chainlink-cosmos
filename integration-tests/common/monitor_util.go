package common

import (
	"log"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func RunHTTPServer(t *testing.T, serverName, serverAddress string, responses map[string][]byte) *http.Server {
	handler := http.NewServeMux()

	for path, response := range responses {
		handler.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			log.Printf("%s received request for %s\n", serverName, path)
			w.Header().Set("Content-Type", "application/octet-stream")
			_, err := w.Write(response)
			require.NoError(t, err, "failed to write response")
		})
	}

	server := &http.Server{
		Addr:    serverAddress,
		Handler: handler,
	}

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("HTTP server ListenAndServe: %v", err)
		}
	}()

	return server
}
