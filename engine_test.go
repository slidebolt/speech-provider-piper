package piperprovider

import (
	"net/http"
	"net/http/httptest"
	"testing"

	provider "github.com/slidebolt/sb-speech-provider-sdk"
)

func TestSynthesize(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/info" {
			return
		}
		w.Header().Set("Content-Type", provider.ContentWAV)
		_, _ = w.Write([]byte("RIFFaudio"))
	}))
	defer upstream.Close()
	audio, err := New(upstream.URL).Synthesize(t.Context(), provider.SynthesisRequest{Text: "hello"})
	if err != nil || string(audio) != "RIFFaudio" {
		t.Fatalf("Synthesize: %q, %v", audio, err)
	}
}
