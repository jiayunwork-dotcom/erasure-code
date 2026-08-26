package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"erasure-code/internal/reedsolomon"
)

func runServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	addr := fs.String("addr", ":8080", "listen address")
	dir := fs.String("dir", "./shards", "base directory for shard storage")
	if err := fs.Parse(reorderArgs(args)); err != nil {
		os.Exit(2)
	}
	if err := os.MkdirAll(*dir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "serve: mkdir %s: %v\n", *dir, err)
		os.Exit(1)
	}

	srv := &ecServer{baseDir: *dir}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", srv.handleHealth)
	mux.HandleFunc("/api/encode", srv.handleEncode)
	mux.HandleFunc("/api/reconstruct", srv.handleReconstruct)
	mux.HandleFunc("/api/verify", srv.handleVerify)
	mux.HandleFunc("/api/info", srv.handleInfo)

	fmt.Printf("erasure-code serving on %s (dir=%s)\n", *addr, *dir)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		fmt.Fprintf(os.Stderr, "serve: %v\n", err)
		os.Exit(1)
	}
}

type ecServer struct {
	baseDir string
	mu      sync.Mutex
}

func (s *ecServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *ecServer) handleEncode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	dataStr := r.URL.Query().Get("data")
	parityStr := r.URL.Query().Get("parity")
	name := r.URL.Query().Get("name")
	if dataStr == "" || parityStr == "" || name == "" {
		http.Error(w, "query params required: data, parity, name", http.StatusBadRequest)
		return
	}
	dataShards, err := strconv.Atoi(dataStr)
	if err != nil || dataShards <= 0 {
		http.Error(w, "data must be a positive integer", http.StatusBadRequest)
		return
	}
	parityShards, err := strconv.Atoi(parityStr)
	if err != nil || parityShards <= 0 {
		http.Error(w, "parity must be a positive integer", http.StatusBadRequest)
		return
	}

	payload, err := io.ReadAll(io.LimitReader(r.Body, 10<<20))
	if err != nil {
		http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if len(payload) == 0 {
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}

	outdir := filepath.Join(s.baseDir, filepath.Base(name))
	s.mu.Lock()
	err = reedsolomon.EncodeDir(payload, dataShards, parityShards, outdir)
	s.mu.Unlock()
	if err != nil {
		http.Error(w, "encode: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"encoded_bytes":  len(payload),
		"data_shards":    dataShards,
		"parity_shards":  parityShards,
		"output_dir":     outdir,
	})
}

func (s *ecServer) handleReconstruct(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "query param required: name", http.StatusBadRequest)
		return
	}
	outdir := filepath.Join(s.baseDir, filepath.Base(name))

	s.mu.Lock()
	defer s.mu.Unlock()

	shards, present, m, err := reedsolomon.ReadShardsFromDir(outdir)
	if err != nil {
		http.Error(w, "read shards: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := reedsolomon.Reconstruct(shards, present, m.DataShards); err != nil {
		http.Error(w, "reconstruct: "+err.Error(), http.StatusInternalServerError)
		return
	}
	recovered, err := reedsolomon.OriginalData(shards, m.DataShards, m.OriginalSize)
	if err != nil {
		http.Error(w, "original data: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.Itoa(len(recovered)))
	w.Write(recovered)
}

func (s *ecServer) handleVerify(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "query param required: name", http.StatusBadRequest)
		return
	}
	outdir := filepath.Join(s.baseDir, filepath.Base(name))

	s.mu.Lock()
	ok, err := reedsolomon.VerifyDir(outdir)
	s.mu.Unlock()
	if err != nil {
		http.Error(w, "verify: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"consistent": ok})
}

func (s *ecServer) handleInfo(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "query param required: name", http.StatusBadRequest)
		return
	}
	outdir := filepath.Join(s.baseDir, filepath.Base(name))
	metaPath := filepath.Join(outdir, "meta.json")

	data, err := os.ReadFile(metaPath)
	if err != nil {
		http.Error(w, "read meta: "+err.Error(), http.StatusNotFound)
		return
	}
	var m reedsolomon.ShardMeta
	if err := json.Unmarshal(data, &m); err != nil {
		http.Error(w, "parse meta: "+err.Error(), http.StatusInternalServerError)
		return
	}

	total := m.DataShards + m.ParityShards
	present := 0
	for i := 0; i < total; i++ {
		p := filepath.Join(outdir, fmt.Sprintf("shard.%03d", i))
		if _, serr := os.Stat(p); serr == nil {
			present++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"original_size":   m.OriginalSize,
		"data_shards":     m.DataShards,
		"parity_shards":   m.ParityShards,
		"total_shards":    total,
		"present_shards":  present,
		"fault_tolerance": m.ParityShards,
	})
}
