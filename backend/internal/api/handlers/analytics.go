package handlers

import (
	"bytes"
	"io"
	"net/http"
)

type AnalyticsHandler struct {
	AnalyticsServiceURL string
}

func (h *AnalyticsHandler) proxyRequest(w http.ResponseWriter, r *http.Request, path string) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "could not read request body")
		return
	}
	defer r.Body.Close()

	url := h.AnalyticsServiceURL + path
	req, err := http.NewRequestWithContext(r.Context(), r.Method, url, bytes.NewReader(body))
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to create proxy request")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "analytics service unavailable")
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (h *AnalyticsHandler) Aggregate(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, "/api/analytics/aggregate")
}

func (h *AnalyticsHandler) Query(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, "/api/analytics/query")
}

func (h *AnalyticsHandler) Regression(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, "/api/analytics/stats/regression")
}

func (h *AnalyticsHandler) TTest(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, "/api/analytics/stats/ttest")
}

func (h *AnalyticsHandler) Forecast(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, "/api/analytics/stats/forecast")
}

func (h *AnalyticsHandler) Anomalies(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, "/api/analytics/anomalies")
}