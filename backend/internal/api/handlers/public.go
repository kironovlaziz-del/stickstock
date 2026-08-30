package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"stickstock/backend/internal/cache"
)

// PublicHandler serves dashboards via their share_token, with no auth —
// see DashboardHandler.CreateShareLink. The token itself is the
// credential (24 random bytes, hex-encoded), so anyone holding a valid
// link can view; nothing here checks a Supabase session.
type PublicHandler struct {
	DB *sql.DB
}

type publicDashboardResponse struct {
	ID      string           `json:"id"`
	Name    string           `json:"name"`
	Layout  json.RawMessage  `json:"layout"`
	Widgets []widgetResponse `json:"widgets"`
}

// rateLimiter —  in-memory limiter - IP or token
type rateLimiter struct {
	mu     sync.Mutex
	store  map[string][]time.Time // key: ip+token
	limit  int
	window time.Duration
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		store:  make(map[string][]time.Time),
		limit:  limit,
		window: window,
	}
}

func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	// clean
	if timestamps, ok := rl.store[key]; ok {
		cutoff := now.Add(-rl.window)
		valid := make([]time.Time, 0, len(timestamps))
		for _, t := range timestamps {
			if t.After(cutoff) {
				valid = append(valid, t)
			}
		}
		if len(valid) >= rl.limit {
			return false
		}
		rl.store[key] = append(valid, now)
	} else {
		rl.store[key] = []time.Time{now}
	}
	return true
}

// global limiter ( Redis,  in-memory)
var publicRateLimiter = newRateLimiter(10, 1*time.Minute) // 10 rate - 1 min

// GetDashboard: GET /api/public/dashboards/{token}
func (h *PublicHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	ip := r.RemoteAddr
	key := ip + ":" + token
	if !publicRateLimiter.allow(key) {
		http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
		return
	}


	var resp publicDashboardResponse
	var dashboardID string
	var layoutBytes []byte
	var expiresAt sql.NullTime
	if err := h.DB.QueryRowContext(r.Context(),
		`SELECT id, name, layout, share_expires_at FROM dashboards WHERE share_token = $1`, token,
	).Scan(&dashboardID, &resp.Name, &layoutBytes, &expiresAt); err != nil {
		writeJSONError(w, http.StatusNotFound, "shared dashboard not found")
		return
	}
	if expiresAt.Valid && time.Now().After(expiresAt.Time) {
		writeJSONError(w, http.StatusGone, "share link has expired")
		return
	}
	resp.ID = dashboardID
	resp.Layout = layoutBytes

	rows, err := h.DB.QueryContext(r.Context(),
		`SELECT id, dashboard_id, saved_query_id, chart_type, config FROM dashboard_widgets WHERE dashboard_id = $1`,
		dashboardID,
	)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not load widgets")
		return
	}
	defer rows.Close()

	resp.Widgets = []widgetResponse{}
	for rows.Next() {
		var wdg widgetResponse
		var configBytes []byte
		if err := rows.Scan(&wdg.ID, &wdg.DashboardID, &wdg.SavedQueryID, &wdg.ChartType, &configBytes); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "could not read widgets")
			return
		}
		wdg.Config = configBytes
		resp.Widgets = append(resp.Widgets, wdg)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// RunWidget executes a shared dashboard's widget query server-side and
// returns just the rows — the underlying data source's DSN/credentials
// are never sent to the (anonymous) caller.
// POST /api/public/dashboards/{token}/widgets/{widget_id}/run
func (h *PublicHandler) RunWidget(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	widgetID := r.PathValue("widget_id")
	ip := r.RemoteAddr
	key := ip + ":" + token
	if !publicRateLimiter.allow(key) {
		http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
		return
	}

	var savedQueryID string
	var expiresAt sql.NullTime
	if err := h.DB.QueryRowContext(r.Context(),
		`SELECT dw.saved_query_id, d.share_expires_at
		 FROM dashboard_widgets dw
		 JOIN dashboards d ON d.id = dw.dashboard_id
		 WHERE dw.id = $1 AND d.share_token = $2`, widgetID, token,
	).Scan(&savedQueryID, &expiresAt); err != nil {
		writeJSONError(w, http.StatusNotFound, "widget not found")
		return
	}
	if expiresAt.Valid && time.Now().After(expiresAt.Time) {
		writeJSONError(w, http.StatusGone, "share link has expired")
		return
	}


	cacheStore := cache.New(h.DB)
	var cachedResult runResponse
	cacheKey := "public_widget:" + widgetID + ":" + token
	found, err := cacheStore.Get(r.Context(), cacheKey, &cachedResult)
	if err == nil && found {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cachedResult)
		return
	}

	var sqlText, dataSourceID string
	if err := h.DB.QueryRowContext(r.Context(),
		`SELECT sql_text, data_source_id FROM saved_queries WHERE id = $1`, savedQueryID,
	).Scan(&sqlText, &dataSourceID); err != nil {
		writeJSONError(w, http.StatusNotFound, "saved query not found")
		return
	}

	var kind, dsn string
	if err := h.DB.QueryRowContext(r.Context(),
		`SELECT kind, dsn FROM data_sources WHERE id = $1`, dataSourceID,
	).Scan(&kind, &dsn); err != nil {
		writeJSONError(w, http.StatusNotFound, "data source not found")
		return
	}

	// timeout 30sec
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	result, err := execute(ctx, kind, dsn, sqlText, nil)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	// cesh save 5 min.
	if len(result.Rows) > 0 {
		_ = cacheStore.Set(r.Context(), cacheKey, result, 5*time.Minute)
	}

	//log audit
	log.Printf("[AUDIT] public widget run: token=%s, widget=%s, ip=%s, rows=%d", token, widgetID, ip, len(result.Rows))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}