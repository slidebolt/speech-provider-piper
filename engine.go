package piperprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	provider "github.com/slidebolt/sb-speech-provider-sdk"
)

type Engine struct {
	Upstream string
	HTTP     *http.Client
	slot     chan struct{}
}

func New(upstream string) *Engine {
	return &Engine{Upstream: strings.TrimRight(upstream, "/"), HTTP: &http.Client{Timeout: 2 * time.Minute}, slot: make(chan struct{}, 1)}
}

func (e *Engine) Capabilities(context.Context) provider.Capabilities {
	return provider.Capabilities{ContractVersion: provider.ContractVersion, Capabilities: []provider.Capability{provider.CapabilitySynthesis}, AudioTypes: []string{provider.ContentWAV}}
}

func (e *Engine) Health(ctx context.Context) provider.Health {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, e.Upstream+"/info", nil)
	resp, err := e.HTTP.Do(req)
	if err != nil {
		return provider.Health{Error: err.Error()}
	}
	resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return provider.Health{Error: fmt.Sprintf("Piper returned HTTP %d", resp.StatusCode)}
	}
	return provider.Health{Ready: true}
}

func (e *Engine) Transcribe(context.Context, []byte) (provider.Transcription, error) {
	return provider.Transcription{}, provider.Problem{Title: "Not supported", Status: http.StatusNotImplemented, Detail: "this provider does not transcribe speech"}
}

func (e *Engine) Synthesize(ctx context.Context, request provider.SynthesisRequest) ([]byte, error) {
	select {
	case e.slot <- struct{}{}:
		defer func() { <-e.slot }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	body, _ := json.Marshal(request)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.Upstream+"/synthesize", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<10))
		return nil, provider.Problem{Title: "Piper synthesis failed", Status: http.StatusBadGateway, Detail: fmt.Sprintf("upstream HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(detail))), Retryable: true}
	}
	return io.ReadAll(io.LimitReader(resp.Body, 32<<20))
}
