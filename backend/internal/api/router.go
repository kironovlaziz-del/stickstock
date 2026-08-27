package api

import (
	"database/sql"
	"net/http"
	"os"

	"stickstock/backend/internal/api/handlers"
	"stickstock/backend/internal/api/middleware"
	"stickstock/backend/internal/config"
)

func NewRouter(cfg *config.Config, database *sql.DB) http.Handler {
	mux := http.NewServeMux()

	// ── Handlers ──────────────────────────────────────────────────
	dsHandler := &handlers.DataSourceHandler{DB: database, EncryptionKey: cfg.EncryptionKey}
	queryHandler := &handlers.QueryHandler{DB: database, EncryptionKey: cfg.EncryptionKey}
	dashboardHandler := &handlers.DashboardHandler{DB: database}
	profileHandler := &handlers.ProfileHandler{DB: database, EncryptionKey: cfg.EncryptionKey, ProfilerURL: "http://data-profiler:8082"}
    metricHandler := &handlers.MetricHandler{DB: database, EncryptionKey: cfg.EncryptionKey, SemanticURL: "http://semantic-layer:8083"}
    semanticHandler := &handlers.SemanticHandler{SemanticURL: "http://semantic-layer:8083"}
	fileHandler := &handlers.FileHandler{DB: database, DatabaseURL: cfg.DatabaseURL}
	reportHandler := &handlers.ReportHandler{DB: database}
	exportHandler := &handlers.ExportHandler{DB: database, EncryptionKey: cfg.EncryptionKey}
	schemaHandler := &handlers.SchemaHandler{DB: database, EncryptionKey: cfg.EncryptionKey}
	lineageHandler := &handlers.LineageHandler{DB: database}
	commentHandler := &handlers.CommentHandler{DB: database}
	authHandler := &handlers.AuthHandler{
		DB:        database,
		JWTSecret: cfg.JWTSecret,
		AppURL:    getEnvOrDefault("APP_URL", "https://stickstock.lol"),
	}
	publicHandler := &handlers.PublicHandler{DB: database}
	analyticsHandler := &handlers.AnalyticsHandler{AnalyticsServiceURL: cfg.AnalyticsServiceURL}
	analyticsTaskHandler := &handlers.AnalyticsTaskHandler{
		DB:                  database,
		AnalyticsServiceURL: cfg.AnalyticsServiceURL,
	}

	// ── Middleware factories ─────────────────────────────────────
	auth := middleware.RequireAuth(cfg.JWTSecret, cfg.SupabaseURL, database)

	// ── Public routes ────────────────────────────────────────────
	mux.HandleFunc("POST /api/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/auth/login", authHandler.Login)
	mux.HandleFunc("POST /api/auth/logout", authHandler.Logout)
	mux.HandleFunc("GET /api/health", handlers.Health(database))
	mux.HandleFunc("GET /api/public/dashboards/{token}", publicHandler.GetDashboard)
	mux.HandleFunc("POST /api/public/dashboards/{token}/widgets/{widget_id}/run", publicHandler.RunWidget)

	// ── Protected routes (require auth) ─────────────────────────

	mux.Handle("GET /api/datasources", auth(http.HandlerFunc(dsHandler.List)))
	mux.Handle("POST /api/datasources", auth(http.HandlerFunc(dsHandler.Create)))
	mux.Handle("DELETE /api/datasources/{id}", auth(http.HandlerFunc(dsHandler.Delete)))
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


	mux.Handle("POST /api/reports", auth(http.HandlerFunc(reportHandler.Create)))
	mux.Handle("GET /api/reports", auth(http.HandlerFunc(reportHandler.List)))
	mux.Handle("PUT /api/reports/{id}", auth(http.HandlerFunc(reportHandler.Update)))
	mux.Handle("DELETE /api/reports/{id}", auth(http.HandlerFunc(reportHandler.Delete)))

	mux.Handle("GET /api/queries/{id}/export", auth(http.HandlerFunc(exportHandler.ExportSaved)))
	mux.Handle("POST /api/export/query", auth(http.HandlerFunc(exportHandler.ExportAdHoc)))

	// ── Analytics routes (proxy to Python service) ──────────────
	mux.Handle("POST /api/analytics/aggregate", auth(http.HandlerFunc(analyticsHandler.Aggregate)))
	mux.Handle("POST /api/analytics/query", auth(http.HandlerFunc(analyticsHandler.Query)))
	mux.Handle("POST /api/analytics/stats/regression", auth(http.HandlerFunc(analyticsHandler.Regression)))
	mux.Handle("POST /api/analytics/stats/ttest", auth(http.HandlerFunc(analyticsHandler.TTest)))
	mux.Handle("POST /api/analytics/stats/forecast", auth(http.HandlerFunc(analyticsHandler.Forecast)))
	mux.Handle("POST /api/analytics/anomalies", auth(http.HandlerFunc(analyticsHandler.Anomalies)))

	// ── Async analytics tasks ────────────────────────────────────
	mux.Handle("POST /api/analytics/tasks", auth(http.HandlerFunc(analyticsTaskHandler.CreateTask)))
	mux.Handle("GET /api/analytics/tasks/{id}", auth(http.HandlerFunc(analyticsTaskHandler.GetTask)))

	// ── Global middleware chain ──────────────────────────────────
    mux.Handle("POST /api/datasources/{id}/profile", auth(http.HandlerFunc(profileHandler.ProfileDataSource)))
    mux.Handle("GET /api/semantic/metrics", auth(http.HandlerFunc(semanticHandler.ListMetrics)))
    mux.Handle("GET /api/semantic/metrics/{id}/run", auth(http.HandlerFunc(metricHandler.RunMetric)))
    mux.Handle("POST /api/semantic/metrics", auth(http.HandlerFunc(semanticHandler.CreateMetric)))
    mux.Handle("GET /api/semantic/metrics/{id}", auth(http.HandlerFunc(semanticHandler.GetMetric)))
    mux.Handle("DELETE /api/semantic/metrics/{id}", auth(http.HandlerFunc(semanticHandler.DeleteMetric)))
    mux.Handle("GET /api/semantic/datasets", auth(http.HandlerFunc(semanticHandler.ListDatasets)))
    mux.Handle("POST /api/semantic/datasets", auth(http.HandlerFunc(semanticHandler.CreateDataset)))
    mux.Handle("GET /api/semantic/datasets/{id}", auth(http.HandlerFunc(semanticHandler.GetDataset)))
    mux.Handle("DELETE /api/semantic/datasets/{id}", auth(http.HandlerFunc(semanticHandler.DeleteDataset)))
	handler := middleware.Locale(cfg.DefaultLocale)(mux)
	handler = middleware.AuditMiddleware(database)(handler)
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

func getEnvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}