package handlers

import (
	"bytes"
	"io"
	"net/http"
	"time"
)

type MetadataProxy struct {
	MetadataURL string
}

func (p *MetadataProxy) proxyRequest(w http.ResponseWriter, r *http.Request, path string) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "could not read request body")
		return
	}
	defer r.Body.Close()

	url := p.MetadataURL + path
	req, err := http.NewRequestWithContext(r.Context(), r.Method, url, bytes.NewReader(body))
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to create request")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	if auth := r.Header.Get("Authorization"); auth != "" {
		req.Header.Set("Authorization", auth)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "metadata service unavailable: "+err.Error())
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (p *MetadataProxy) RenderTemplate(w http.ResponseWriter, r *http.Request) {
	p.proxyRequest(w, r, "/template/render")
}

func (p *MetadataProxy) ValidateTemplate(w http.ResponseWriter, r *http.Request) {
	p.proxyRequest(w, r, "/template/validate")
}
