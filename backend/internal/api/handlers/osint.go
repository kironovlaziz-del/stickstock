package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"stickstock/backend/internal/osint"
)

type OSINTHandler struct {
	HIBPAPIKey string
}

type osintLookupRequest struct {
	Type   string `json:"type"`   // "whois" | "dns" | "breach"
	Target string `json:"target"` // domain for whois/dns, email for breach
}

// Lookup: POST /api/osint/lookup
func (h *OSINTHandler) Lookup(w http.ResponseWriter, r *http.Request) {
	var req osintLookupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Target == "" {
		writeJSONError(w, http.StatusBadRequest, "type and target are required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()

	switch req.Type {
	case "whois":
		result, err := osint.WHOIS(req.Target)
		if err != nil {
			writeJSONError(w, http.StatusBadGateway, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"raw": result})

	case "dns":
		result, err := osint.DNSLookup(ctx, req.Target)
		if err != nil {
			writeJSONError(w, http.StatusBadGateway, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)

	case "breach":
		names, err := osint.CheckBreaches(h.HIBPAPIKey, req.Target)
		if err != nil {
			writeJSONError(w, http.StatusBadGateway, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"breaches": names})

	default:
		writeJSONError(w, http.StatusBadRequest, `type must be "whois", "dns", or "breach"`)
	}
}
