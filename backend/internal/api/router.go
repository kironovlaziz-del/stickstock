package api

import (
	"database/sql"
	"net/http"

	"stickstock/backend/internal/api/handlers"
	"stickstock/backend/internal/api/middleware"
	"stickstock/backend/internal/config"
)

func NewRouter(cfg *config.Config, database *sql.DB) http.Handler {
	mux := http.NewServeMux()

	dsHandler := &handlers.DataSourceHandler{DB: database}
	queryHandler := &handlers.QueryHandler{DB: database}
	dashboardHandler := &handlers.DashboardHandler{DB: database}
	profileHandler := &handlers.ProfileHandler{DB: database}
	fileHandler := &handlers.FileHandler{DB: database, DatabaseURL: cfg.DatabaseURL}
	reportHandler := &handlers.ReportHandler{DB: database}
	exportHandler := &handlers.ExportHandler{DB: database}
	schemaHandler := &handlers.SchemaHandler{DB: database}
	lineageHandler := &handlers.LineageHandler{DB: database}
	commentHandler := &handlers.CommentHandler{DB: database}
	publicHandler := &handlers.PublicHandler{DB: database}
	adminHandler := &handlers.AdminHandler{DB: database}
	osintHandler := &handlers.OSINTHandler{HIBPAPIKey: cfg.HIBPAPIKey}

	// Public routes. Registration/login now happen on the frontend via
	// Supabase Auth (supabase-js) — this backend only verifies the JWTs
	// Supabase issues (see middleware.RequireAuth) and never sees passwords.
	mux.HandleFunc("GET /api/health", handlers.Health(database))

	// Also public: shared dashboards. The share_token itself is the
	// credential (see DashboardHandler.CreateShareLink) — no Supabase
	// session involved, so these deliberately aren't wrapped in `auth`.
	mux.HandleFunc("GET /api/public/dashboards/{token}", publicHandler.GetDashboard)
	mux.HandleFunc("POST /api/public/dashboards/{token}/widgets/{widget_id}/run", publicHandler.RunWidget)

	// Protected routes. cfg.JWTSecret is the Supabase project's JWT
	// secret (Settings → API in the Supabase dashboard), not a secret
	// this backend generates itself.
	auth := middleware.RequireAuth(cfg.JWTSecret, cfg.SupabaseURL, database)
	admin := middleware.RequireAdmin(database)
	mux.Handle("GET /api/me", auth(http.HandlerFunc(profileHandler.Get)))
	mux.Handle("PUT /api/me", auth(http.HandlerFunc(profileHandler.Update)))

	mux.Handle("GET /api/datasources", auth(http.HandlerFunc(dsHandler.List)))
	mux.Handle("POST /api/datasources", auth(http.HandlerFunc(dsHandler.Create)))
	mux.Handle("POST /api/datasources/upload", auth(http.HandlerFunc(fileHandler.Upload)))
	mux.Handle("GET /api/datasources/{id}/schema", auth(http.HandlerFunc(schemaHandler.Get)))

	mux.Handle("POST /api/queries/run", auth(http.HandlerFunc(queryHandler.RunAdHoc)))
	mux.Handle("POST /api/queries", auth(http.HandlerFunc(queryHandler.Create)))
	mux.Handle("GET /api/queries", auth(http.HandlerFunc(queryHandler.List)))
	mux.Handle("GET /api/queries/{id}", auth(http.HandlerFunc(queryHandler.Get)))
	mux.Handle("PUT /api/queries/{id}", auth(http.HandlerFunc(queryHandler.Update)))
	mux.Handle("DELETE /api/queries/{id}", auth(http.HandlerFunc(queryHandler.Delete)))
	mux.Handle("GET /api/queries/{id}/versions", auth(http.HandlerFunc(queryHandler.Versions)))
	mux.Handle("POST /api/queries/{id}/run", auth(http.HandlerFunc(queryHandler.RunSaved)))

	mux.Handle("POST /api/dashboards", auth(http.HandlerFunc(dashboardHandler.Create)))
	mux.Handle("GET /api/dashboards", auth(http.HandlerFunc(dashboardHandler.List)))
	mux.Handle("GET /api/dashboards/{id}", auth(http.HandlerFunc(dashboardHandler.Get)))
	mux.Handle("PUT /api/dashboards/{id}", auth(http.HandlerFunc(dashboardHandler.Update)))
	mux.Handle("DELETE /api/dashboards/{id}", auth(http.HandlerFunc(dashboardHandler.Delete)))
	mux.Handle("POST /api/dashboards/{id}/widgets", auth(http.HandlerFunc(dashboardHandler.AddWidget)))
	mux.Handle("PUT /api/dashboards/{id}/widgets/{widget_id}", auth(http.HandlerFunc(dashboardHandler.UpdateWidget)))
	mux.Handle("DELETE /api/dashboards/{id}/widgets/{widget_id}", auth(http.HandlerFunc(dashboardHandler.DeleteWidget)))

	mux.Handle("POST /api/dashboards/{id}/share", auth(http.HandlerFunc(dashboardHandler.CreateShareLink)))
	mux.Handle("DELETE /api/dashboards/{id}/share", auth(http.HandlerFunc(dashboardHandler.RevokeShareLink)))
	mux.Handle("GET /api/dashboards/{id}/collaborators", auth(http.HandlerFunc(dashboardHandler.ListCollaborators)))
	mux.Handle("POST /api/dashboards/{id}/collaborators", auth(http.HandlerFunc(dashboardHandler.AddCollaborator)))
	mux.Handle("DELETE /api/dashboards/{id}/collaborators/{user_id}", auth(http.HandlerFunc(dashboardHandler.RemoveCollaborator)))

	mux.Handle("GET /api/dashboards/{id}/widgets/{widget_id}/comments", auth(http.HandlerFunc(commentHandler.List)))
	mux.Handle("POST /api/dashboards/{id}/widgets/{widget_id}/comments", auth(http.HandlerFunc(commentHandler.Create)))
	mux.Handle("DELETE /api/comments/{comment_id}", auth(http.HandlerFunc(commentHandler.Delete)))

	mux.Handle("GET /api/lineage", auth(http.HandlerFunc(lineageHandler.Get)))

	mux.Handle("GET /api/admin/users", auth(admin(http.HandlerFunc(adminHandler.ListUsers))))
	mux.Handle("PUT /api/admin/users/{id}", auth(admin(http.HandlerFunc(adminHandler.UpdateUser))))
	mux.Handle("GET /api/admin/stats", auth(admin(http.HandlerFunc(adminHandler.Stats))))

	mux.Handle("POST /api/osint/lookup", auth(http.HandlerFunc(osintHandler.Lookup)))

	mux.Handle("POST /api/reports", auth(http.HandlerFunc(reportHandler.Create)))
	mux.Handle("GET /api/reports", auth(http.HandlerFunc(reportHandler.List)))
	mux.Handle("PUT /api/reports/{id}", auth(http.HandlerFunc(reportHandler.Update)))
	mux.Handle("DELETE /api/reports/{id}", auth(http.HandlerFunc(reportHandler.Delete)))

	mux.Handle("GET /api/queries/{id}/export", auth(http.HandlerFunc(exportHandler.ExportSaved)))
	mux.Handle("POST /api/export/query", auth(http.HandlerFunc(exportHandler.ExportAdHoc)))

	// Global middleware chain: locale resolution wraps everything so both
	// public and protected handlers can read the caller's language.
	var handler http.Handler = mux
	handler = middleware.Locale(cfg.DefaultLocale)(handler)
	handler = corsMiddleware(cfg.AllowedOrigins)(handler)

	return handler
}

func corsMiddleware(allowedOrigins string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigins)
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept-Language")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
