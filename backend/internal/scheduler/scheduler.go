// Package scheduler implements the "run due scheduled reports" batch job
// invoked by cmd/worker. It's deliberately a single-shot function, not a
// long-running loop with its own ticker — the recommended architecture
// wants this cron-invoked (host crontab running `docker compose run
// --rm backend /app/worker` every few minutes), not an always-on process
// competing for RAM on a small VPS.
package scheduler

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/robfig/cron/v3"

	"stickstock/backend/internal/connectors"
	"stickstock/backend/internal/delivery"
	"stickstock/backend/internal/queryengine"
)

type Config struct {
	SMTP     delivery.SMTPConfig
	Telegram delivery.TelegramConfig
}

// cronParser accepts standard 5-field cron expressions (minute hour dom
// month dow) — no seconds field, matching what most people expect from
// "cron syntax". Shared with handlers.ParseCron so the API validates
// exactly what the worker will later parse.
var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

func ParseCron(expr string) (cron.Schedule, error) {
	return cronParser.Parse(expr)
}

type dueReport struct {
	ID             string
	SavedQueryID   string
	CronExpr       string
	DeliveryKind   string
	DeliveryTarget string
	LastRunAt      sql.NullTime
	NextRunAt      sql.NullTime // новое поле
}

// RunDue finds every active scheduled report whose cron schedule says
// it's due (next run after last_run_at is <= now), executes its saved
// query, delivers the result, and stamps last_run_at. One bad report
// (invalid cron, query failure, delivery failure) is logged and skipped
// rather than aborting the whole batch.
func RunDue(ctx context.Context, db *sql.DB, cfg Config) error {
	now := time.Now()

	// Выбираем отчёты, у которых next_run_at <= now (или NULL, если ещё не запускались)
	// Используем FOR UPDATE SKIP LOCKED для безопасной блокировки.
	rows, err := db.QueryContext(ctx,
		`SELECT id, saved_query_id, cron_expr, delivery_kind, delivery_target, last_run_at, next_run_at
		 FROM scheduled_reports
		 WHERE is_active = true
		   AND (next_run_at IS NULL OR next_run_at <= $1)
		 ORDER BY next_run_at NULLS FIRST
		 LIMIT 10
		 FOR UPDATE SKIP LOCKED`,
		now)
	if err != nil {
		return fmt.Errorf("could not list scheduled reports: %w", err)
	}
	defer rows.Close()

	var due []dueReport
	for rows.Next() {
		var r dueReport
		if err := rows.Scan(&r.ID, &r.SavedQueryID, &r.CronExpr, &r.DeliveryKind, &r.DeliveryTarget, &r.LastRunAt, &r.NextRunAt); err != nil {
			return fmt.Errorf("could not read scheduled report: %w", err)
		}
		due = append(due, r)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	ran, skipped := 0, 0

	for _, r := range due {
		// Парсим cron-выражение
		schedule, err := cronParser.Parse(r.CronExpr)
		if err != nil {
			log.Printf("scheduler: report %s has invalid cron_expr %q: %v", r.ID, r.CronExpr, err)
			continue
		}

		// Вычисляем следующее время запуска на основе last_run_at
		var nextRun time.Time
		if r.LastRunAt.Valid {
			nextRun = schedule.Next(r.LastRunAt.Time)
		} else {
			// Если никогда не запускался, то запускаем сразу (или с учётом времени создания)
			nextRun = now
		}
		// Обновляем next_run_at в базе атомарно (только если оно не изменилось)
		res, err := db.ExecContext(ctx,
			`UPDATE scheduled_reports
			 SET next_run_at = $1
			 WHERE id = $2 AND (next_run_at IS NULL OR next_run_at = $3)`,
			nextRun, r.ID, r.NextRunAt.Time)
		if err != nil {
			log.Printf("scheduler: could not update next_run_at for report %s: %v", r.ID, err)
			continue
		}
		if n, _ := res.RowsAffected(); n == 0 {
			// Кто-то другой уже обработал этот отчёт
			skipped++
			continue
		}

		// Выполняем отчёт
		if err := runOne(ctx, db, cfg, r); err != nil {
			log.Printf("scheduler: report %s failed: %v", r.ID, err)
			// Не помечаем как запущенный, чтобы повторить позже (можно добавить счётчик попыток)
			continue
		}
		ran++

		// Обновляем last_run_at и next_run_at после успешного выполнения
		// next_run_at вычисляем заново, чтобы избежать дрейфа времени
		nextRunAfter := schedule.Next(time.Now())
		if _, err := db.ExecContext(ctx,
			`UPDATE scheduled_reports
			 SET last_run_at = $1, next_run_at = $2
			 WHERE id = $3`,
			time.Now(), nextRunAfter, r.ID,
		); err != nil {
			log.Printf("scheduler: report %s ran but could not update timestamps: %v", r.ID, err)
		}
	}

	log.Printf("scheduler: %d report(s) ran, %d skipped/claimed, total %d due", ran, skipped, len(due))
	return nil
}

func runOne(ctx context.Context, db *sql.DB, cfg Config, r dueReport) error {
	var sqlText, dataSourceID string
	if err := db.QueryRowContext(ctx,
		`SELECT sql_text, data_source_id FROM saved_queries WHERE id = $1`, r.SavedQueryID,
	).Scan(&sqlText, &dataSourceID); err != nil {
		return fmt.Errorf("saved query lookup: %w", err)
	}

	var kind, dsn string
	if err := db.QueryRowContext(ctx,
		`SELECT kind, dsn FROM data_sources WHERE id = $1`, dataSourceID,
	).Scan(&kind, &dsn); err != nil {
		return fmt.Errorf("data source lookup: %w", err)
	}

	queryText := sqlText
	if connectors.IsSQLKind(kind) {
		// Saved queries were validated at creation/update time (see
		// handlers/queries.go), so this is really just binding — but a
		// query with unfilled :params fails here with a clear error
		// rather than sending malformed SQL, since scheduled reports
		// don't have anywhere to source parameter values from yet.
		bound, _, err := queryengine.BindParams(sqlText, map[string]interface{}{})
		if err != nil {
			return fmt.Errorf("scheduled reports don't support parameterized queries yet: %w", err)
		}
		queryText = bound
	}

	conn, err := connectors.New(kind, dsn)
	if err != nil {
		return fmt.Errorf("connector: %w", err)
	}
	defer conn.Close()

	result, err := conn.Query(ctx, queryText)
	if err != nil {
		return fmt.Errorf("query execution: %w", err)
	}

	body := formatResultAsText(result.Columns, result.Rows, 50)

	switch r.DeliveryKind {
	case "email":
		return delivery.SendEmail(cfg.SMTP, r.DeliveryTarget, "StickStock scheduled report", body)
	case "telegram":
		return delivery.SendTelegramMessage(cfg.Telegram, r.DeliveryTarget, body)
	default:
		return fmt.Errorf("unknown delivery_kind %q", r.DeliveryKind)
	}
}

// formatResultAsText renders a query result as a plain-text table —
// simple and safe to email or message, at the cost of not being pretty.
// A proper HTML email / rendered table is a reasonable future upgrade
// (see the export work planned after this).
func formatResultAsText(columns []string, rows [][]interface{}, maxRows int) string {
	var b strings.Builder
	b.WriteString(strings.Join(columns, " | "))
	b.WriteString("\n")

	for i, row := range rows {
		if i >= maxRows {
			fmt.Fprintf(&b, "... (%d more rows)\n", len(rows)-maxRows)
			break
		}
		cells := make([]string, len(row))
		for j, v := range row {
			cells[j] = fmt.Sprintf("%v", v)
		}
		b.WriteString(strings.Join(cells, " | "))
		b.WriteString("\n")
	}

	return b.String()
}