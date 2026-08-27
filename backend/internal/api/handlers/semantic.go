package handlers

import (
	"bytes"
	"io"
	"net/http"
	"time"
)

type SemanticHandler struct {
	SemanticURL string
}

func (h *SemanticHandler) proxyRequest(w http.ResponseWriter, r *http.Request, path string) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "could not read body")
		return
	}
	defer r.Body.Close()

	url := h.SemanticURL + path
	req, err := http.NewRequestWithContext(r.Context(), r.Method, url, bytes.NewReader(body))
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to create request")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "semantic service unreachable")
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (h *SemanticHandler) ListMetrics(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, "/metrics")
}

func (h *SemanticHandler) CreateMetric(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, "/metrics")
}

func (h *SemanticHandler) GetMetric(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, "/metrics/"+r.PathValue("id"))
}

func (h *SemanticHandler) DeleteMetric(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, "/metrics/"+r.PathValue("id"))
}

func (h *SemanticHandler) ListDatasets(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, "/datasets")
}

func (h *SemanticHandler) CreateDataset(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, "/datasets")
}

func (h *SemanticHandler) GetDataset(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, "/datasets/"+r.PathValue("id"))
}

func (h *SemanticHandler) DeleteDataset(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, "/datasets/"+r.PathValue("id"))
}
