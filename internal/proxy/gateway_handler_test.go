package proxy

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGatewayHandlerForwardsRequest(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/hello" {
				t.Errorf("path = %q, want %q", r.URL.Path, "/hello")
			}

			if r.URL.Query().Get("name") != "gateway" {
				t.Errorf(
					"query parameter name = %q, want %q",
					r.URL.Query().Get("name"),
					"gateway",
				)
			}

			fmt.Fprint(w, "hello from upstream")
		},
	))
	defer upstream.Close()

	handler, err := NewGatewayHandler(upstream.URL)
	if err != nil {
		t.Fatalf("NewGatewayHandler() error = %v", err)
	}

	request := httptest.NewRequest(
		http.MethodGet,
		"http://gateway.local/hello?name=gateway",
		nil,
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"status code = %d, want %d",
			recorder.Code,
			http.StatusOK,
		)
	}

	if body := recorder.Body.String(); body != "hello from upstream" {
		t.Fatalf(
			"body = %q, want %q",
			body,
			"hello from upstream",
		)
	}
}

func TestGatewayHandlerPreservesResponseHeaders(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Upstream", "flux-test")
			w.WriteHeader(http.StatusCreated)
		},
	))
	defer upstream.Close()

	handler, err := NewGatewayHandler(upstream.URL)
	if err != nil {
		t.Fatalf("NewGatewayHandler() error = %v", err)
	}

	request := httptest.NewRequest(
		http.MethodPost,
		"http://gateway.local/resource",
		nil,
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf(
			"status code = %d, want %d",
			recorder.Code,
			http.StatusCreated,
		)
	}

	if got := recorder.Header().Get("X-Upstream"); got != "flux-test" {
		t.Fatalf(
			"X-Upstream = %q, want %q",
			got,
			"flux-test",
		)
	}
}

func TestNewGatewayHandlerRejectsInvalidURL(t *testing.T) {
	_, err := NewGatewayHandler("://invalid-url")

	if err == nil {
		t.Fatal("NewGatewayHandler() error = nil, want error")
	}
}
