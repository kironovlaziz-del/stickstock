# API Reference

All endpoints (except public ones) require an **Authorization: Bearer <token>** header.

---

## Authentication

### Register

`POST /api/auth/register`

Request body:
```json
{
  "username": "myuser",
  "password": "mypass"
}
Response:
{
  "token": "jwt_token",
  "id": "user_uuid",
  "username": "myuser"
}
Login

POST /api/auth/login

Request body:
{
  "username": "myuser",
  "password": "mypass"
}
Response: same as register.
Connections
List all connections

GET /api/datasources
Create a connection
POST /api/datasources

Request body:
{
  "name": "My DB",
  "kind": "postgres",
  "dsn": "postgres://user:pass@host:5432/db"
}
Delete a connection

DELETE /api/datasources/{id}
Queries
Run ad‑hoc query

POST /api/queries/run

Request body:
{
  "data_source_id": "ds_uuid",
  "sql": "SELECT * FROM users"
}
Saved queries

    GET /api/queries – list all saved queries.

    POST /api/queries – create a saved query.

    GET /api/queries/{id} – get a query.

    PUT /api/queries/{id} – update query (increments version).

    DELETE /api/queries/{id} – delete query.

    POST /api/queries/{id}/run – execute saved query.

    GET /api/queries/{id}/versions – list version history.

Dashboards

    POST /api/dashboards – create.

    GET /api/dashboards – list.

    GET /api/dashboards/{id}?batch=true – get dashboard with all widget data (cached).

    PUT /api/dashboards/{id} – update name/layout.

    DELETE /api/dashboards/{id} – delete.

Widgets

    POST /api/dashboards/{id}/widgets – add widget.

    PUT /api/dashboards/{id}/widgets/{widget_id} – update widget.

    DELETE /api/dashboards/{id}/widgets/{widget_id} – delete widget.

Sharing & Collaboration

    POST /api/dashboards/{id}/share – generate shareable link.

    DELETE /api/dashboards/{id}/share – revoke link.

    GET /api/dashboards/{id}/collaborators – list collaborators.

    POST /api/dashboards/{id}/collaborators – add collaborator.

    DELETE /api/dashboards/{id}/collaborators/{user_id} – remove collaborator.

Comments

    GET /api/dashboards/{id}/widgets/{widget_id}/comments

    POST /api/dashboards/{id}/widgets/{widget_id}/comments

    DELETE /api/comments/{comment_id}

Scheduled Reports

    POST /api/reports – create report.

    GET /api/reports – list reports.

    PUT /api/reports/{id} – update.

    DELETE /api/reports/{id} – delete.

Admin

    GET /api/admin/users – list users.

    PUT /api/admin/users/{id} – update user (block/unblock, grant admin).

    GET /api/admin/stats – system statistics.

    GET /api/admin/audit – audit logs (requires admin).

Async Analytics

    POST /api/analytics/tasks – start a task (regression, forecast, etc.).

    GET /api/analytics/tasks/{id} – get task status and result.

Public endpoints (no auth)

    GET /api/health – health check.

    GET /api/public/dashboards/{token} – view shared dashboard.

    POST /api/public/dashboards/{token}/widgets/{widget_id}/run – run widget query.
    EOF
