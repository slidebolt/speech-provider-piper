package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	provider "github.com/slidebolt/sb-speech-provider-sdk"
	piper "github.com/slidebolt/speech-provider-piper"
)

func main() {
	if len(os.Args) == 3 && os.Args[1] == "--healthcheck" {
		healthcheck(os.Args[2])
		return
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	upstream := os.Getenv("PIPER_UPSTREAM_URL")
	if upstream == "" {
		port, err := freePort()
		if err != nil {
			log.Fatal(err)
		}
		upstream = "http://127.0.0.1:" + strconv.Itoa(port)
		voice := value("PIPER_VOICE", "en_US-lessac-medium")
		dataDir := value("PIPER_DATA_DIR", "/models")
		python := value("PIPER_PYTHON_PATH", "python3")
		command := exec.CommandContext(ctx, python, "-m", "piper.http_server", "--host", "127.0.0.1", "--port", strconv.Itoa(port), "--data-dir", dataDir, "-m", voice)
		command.Dir = filepath.Clean(dataDir)
		command.Stdout, command.Stderr = os.Stdout, os.Stderr
		if err := command.Start(); err != nil {
			log.Fatalf("start Piper: %v", err)
		}
		go func() {
			if err := command.Wait(); err != nil && ctx.Err() == nil {
				log.Printf("Piper stopped: %v", err)
				cancel()
			}
		}()
	}
	server := &http.Server{Addr: value("PROVIDER_ADDR", ":8080"), Handler: provider.Handler(piper.New(upstream), os.Getenv("PROVIDER_TOKEN")), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		shutdown, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		_ = server.Shutdown(shutdown)
	}()
	log.Printf("speech Piper provider listening on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func healthcheck(base string) {
	client := &http.Client{Timeout: 3 * time.Second}
	req, _ := http.NewRequest(http.MethodGet, base+provider.PathHealth, nil)
	if token := os.Getenv("PROVIDER_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		if err != nil {
			log.Print(err)
		}
		os.Exit(1)
	}
	resp.Body.Close()
}

func freePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port
	if port == 0 {
		return 0, fmt.Errorf("no free port")
	}
	return port, nil
}

func value(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
