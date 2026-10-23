package ports

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stackmesh/go-media-gateway/internal/domain"
)

func TestPutObjectAndFetch(t *testing.T) {
	svc, _ := testService(t)
	handler := NewHandler(svc, nil)
	body := []byte("clip")

	rec := putObject(t, handler, body, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d body %s", rec.Code, rec.Body)
	}
	var created struct {
		ID      string `json:"id"`
		Created bool   `json:"created"`
		Region  string `json:"region"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if !created.Created || created.Region != "us-east-1" || created.ID == "" {
		t.Fatalf("payload: %+v", created)
	}

	again := putObject(t, handler, body, "")
	if again.Code != http.StatusOK {
		t.Fatalf("replay status %d", again.Code)
	}

	get := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/objects/"+created.ID, nil)
	handler.ServeHTTP(get, req)
	if get.Code != http.StatusOK || !bytes.Contains(get.Body.Bytes(), []byte(created.ID)) {
		t.Fatalf("get %d %s", get.Code, get.Body)
	}

	sum := sha256.Sum256(body)
	route := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v1/route?hash="+hex.EncodeToString(sum[:])+"&bytes=4&mime=video/mp4", nil)
	handler.ServeHTTP(route, req)
	if route.Code != http.StatusOK || !bytes.Contains(route.Body.Bytes(), []byte("edge-")) {
		t.Fatalf("route %d %s", route.Code, route.Body)
	}
}

func TestPutObjectRejectsMismatchAndSize(t *testing.T) {
	svc, _ := testService(t)
	svc.Policy.MaxBytes = 4
	handler := NewHandler(svc, nil)

	rec := putObject(t, handler, []byte("12345"), "")
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("size status %d %s", rec.Code, rec.Body)
	}

	svc.Policy.MaxBytes = 1 << 20
	rec = putObject(t, handler, []byte("clip"), strings.Repeat("0", 64))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("hash status %d %s", rec.Code, rec.Body)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(domain.ErrHashMismatch.Error())) {
		t.Fatalf("body %s", rec.Body)
	}
}

func putObject(t *testing.T, handler http.Handler, body []byte, hash string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/v1/objects", bytes.NewReader(body))
	req.Header.Set("X-Tenant-Id", "demo")
	req.Header.Set("Content-Type", "video/mp4")
	if hash != "" {
		req.Header.Set("X-Content-Sha256", hash)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestReadyzPing(t *testing.T) {
	svc, _ := testService(t)
	handler := NewHandler(svc, func(context.Context) error { return io.EOF })
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status %d", rec.Code)
	}
}
