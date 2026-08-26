# User Guide

## Getting Started

1. Open the platform in your browser.
2. Sign up using **username** and **password**.
3. You will be redirected to the **Dashboards** page.

---

## Connections

### Adding a data source

1. Go to **Connections** from the sidebar.
2. Click **Add Connection**.
3. Fill in:
   - Name – a friendly name (e.g., "Production DB")
   - Kind – choose one: `postgres`, `mysql`, `mongodb`, or `rest`
   - DSN – connection string (e.g., `postgres://user:pass@host:5432/db`)
4. Click **Save**. The connection will be tested automatically.

### Deleting a connection

- Click **Delete** next to the connection.
- Confirm the action.

---

## Queries

### Running a query

1. Go to **Queries** → **New Query**.
2. Write your SQL or JSON (for MongoDB/REST).
3. Click **Run**.
4. Results appear below.

### Saving a query

1. Enter a name in the "New query" field.
2. Click **Save**.
3. The query appears in the left sidebar.

### Using the visual query builder

1. Go to **Queries** → **Visual query builder**.
2. Select a connection and table.
3. Pick columns, add filters, and choose aggregations.
4. Click **Run** to see results.

---

## Dashboards

### Creating a dashboard

1. Go to **Dashboards** → **Create Dashboard**.
2. Enter a name and click **Save**.

### Adding widgets

1. Open a dashboard and click **Edit**.
2. Choose a saved query, select a chart type, and click **Add Widget**.
3. Drag and resize widgets on the grid.

### Sharing a dashboard

1. Open the dashboard and click **Share**.
2. Generate a shareable link (valid for 30 days).
3. Copy the link and send it to anyone – they can view the dashboard without logging in.

### Collaborating

1. In the **Share** panel, invite users by email.
2. Assign roles: `editor` (can edit widgets) or `viewer` (read-only).

---

## Analytics

StickStock includes advanced statistical analysis:
- **Regression** – linear regression with p‑values.
- **Forecast** – ARIMA forecasting.
- **Anomaly Detection** – Isolation Forest.
- **T-test** – Welch's t‑test.

### Using analytics on a query result

1. Run a query.
2. In the results panel, click **Analytics**.
3. Choose the type of analysis and configure parameters.
4. Run it – results appear below.

---

## Scheduled Reports

1. Go to **Reports** → **Schedule Report**.
2. Select a saved query.
3. Choose a cron schedule (e.g., `0 9 * * *` for daily at 9 AM).
4. Set delivery method (email or Telegram) and target address.
5. The report will be sent automatically.

---

## Admin Panel

If you are an admin, you can:
- View system stats.
- Manage users (block/unblock, grant admin rights).

---

## Tips

- Use `:param` placeholders in SQL to make queries parameterized.
- Export results to CSV, Excel, or PDF from any query result.
- Comment on widgets to collaborate with teammates.
