package ports

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/stackmesh/go-media-gateway/internal/domain"
)

// Handler is the inbound HTTP adaptor for the media gateway contract.
type Handler struct {
	Service *IngestService
	Ping    func(context.Context) error
}

func NewHandler(service *IngestService, ping func(context.Context) error) http.Handler {
	h := &Handler{Service: service, Ping: ping}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.healthz)
	mux.HandleFunc("GET /readyz", h.readyz)
	mux.HandleFunc("POST /v1/objects", h.putObject)
	mux.HandleFunc("GET /v1/objects/{id}", h.getObject)
	mux.HandleFunc("GET /v1/route", h.route)
	return mux
}

func (h *Handler) healthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (h *Handler) readyz(w http.ResponseWriter, r *http.Request) {
	if h.Ping != nil {
		if err := h.Ping(r.Context()); err != nil {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (h *Handler) putObject(w http.ResponseWriter, r *http.Request) {
	asset, created, err := h.Service.Ingest(
		r.Context(),
		r.Header.Get("X-Tenant-Id"),
		r.Header.Get("Content-Type"),
		r.Header.Get("X-Content-Sha256"),
		r.Body,
	)
	if err != nil {
		writeErr(w, err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	writeAsset(w, status, asset, &created)
}

func (h *Handler) getObject(w http.ResponseWriter, r *http.Request) {
	asset, err := h.Service.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeAsset(w, http.StatusOK, asset, nil)
}

func (h *Handler) route(w http.ResponseWriter, r *http.Request) {
	size, err := strconv.ParseInt(r.URL.Query().Get("bytes"), 10, 64)
	if err != nil {
		writeErr(w, domain.ErrEmpty)
		return
	}
	placement, err := h.Service.Preview(
		r.Context(),
		r.URL.Query().Get("hash"),
		r.URL.Query().Get("mime"),
		r.URL.Query().Get("region"),
		size,
	)
	if err != nil {
		writeErr(w, err)
		return
	}
	replicas := placement.Replicas
	if replicas == nil {
		replicas = []string{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"primary":  placement.Primary,
		"replicas": replicas,
		"region":   placement.Region,
	})
}

func writeAsset(w http.ResponseWriter, status int, asset domain.Asset, created *bool) {
	replicas := asset.Replicas
	if replicas == nil {
		replicas = []string{}
	}
	payload := map[string]any{
		"id":          asset.ID,
		"tenantId":    asset.TenantID,
		"contentHash": asset.ContentHash,
		"sizeBytes":   asset.SizeBytes,
		"mime":        asset.MIME,
		"region":      asset.Region,
		"primaryNode": asset.PrimaryNode,
		"replicas":    replicas,
		"createdAt":   asset.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	if created != nil {
		payload["created"] = *created
	}
	writeJSON(w, status, payload)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(payload); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

func writeErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, domain.ErrTooLarge):
		http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
	case errors.Is(err, domain.ErrMIME):
		http.Error(w, err.Error(), http.StatusUnsupportedMediaType)
	case errors.Is(err, domain.ErrEmpty),
		errors.Is(err, domain.ErrTenant),
		errors.Is(err, domain.ErrHash),
		errors.Is(err, domain.ErrHashMismatch):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, domain.ErrNoNodes):
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	default:
		slog.Error("request failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}
