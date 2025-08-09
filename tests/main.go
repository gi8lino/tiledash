package main

import (
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/containeroo/tinyflags"
)

// Conifg is the mock server configuration.
type Config struct {
	RandomDelay bool
	Port        int
	DataDir     string
}

func main() {
	cfg := Config{}
	tf := tinyflags.NewFlagSet("mock-server", tinyflags.ExitOnError)
	tf.BoolVar(&cfg.RandomDelay, "random-delay", false, "Add random delay (0-2s) to section responses")
	tf.IntVar(&cfg.Port, "port", 8081, "Port to run mock server on")
	tf.StringVar(&cfg.DataDir, "data-dir", "./data", "Directory to serve JSON section data from")
	if err := tf.Parse(os.Args[1:]); err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/rest/api/2/search", handleSection(&cfg))

	log.Printf("Mock server listening on :%d", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(cfg.Port), nil))
}

// handleSection returns fake JSON data from a file like data/0.json, data/1.json, etc.
func handleSection(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract jql=filter=xxxx
		jql := r.URL.Query().Get("jql")
		if !strings.HasPrefix(jql, "filter=") {
			http.Error(w, "unsupported jql expression", http.StatusBadRequest)
			return
		}

		filterID := strings.TrimPrefix(jql, "filter=")
		if _, err := strconv.Atoi(filterID); err != nil {
			http.Error(w, "invalid filter id", http.StatusBadRequest)
			return
		}

		if cfg.RandomDelay {
			// Add a delay of 200-1000ms so the Spinner can be seen
			time.Sleep(time.Duration(rand.Intn(800)+200) * time.Millisecond)
		}

		path := filepath.Join(cfg.DataDir, filterID+".json")
		data, err := os.ReadFile(path)
		if err != nil {
			http.Error(w, "mock data not found for filter="+filterID, http.StatusNotFound)
			return
		}

		log.Printf("serving %s", path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(data) // nolint:errcheck
	}
}
