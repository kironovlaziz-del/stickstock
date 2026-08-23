"""POST /api/analytics/query — run a read-only SQL query directly against
CSV content via DuckDB, with no need to load it into Postgres first
file inside the container because DuckDB's CSV reader wants a file path,
not an in-memory buffer; the file is deleted before the response returns.
"""

import os
import re
import tempfile

import duckdb
from fastapi import APIRouter, HTTPException
from pydantic import BaseModel

router = APIRouter()

_ALLOWED_DELIMS = {",", "\t", ";", "|"}
_FORBIDDEN_KEYWORDS = (
    "insert", "update", "delete", "drop", "alter", "create",
    "copy", "attach", "install", "load", "export", "pragma",
)
_DEFAULT_LIMIT = 1000

class FileQueryRequest(BaseModel):
    csv_content: str
    sql: str  # query the data via the table alias "data", e.g. "SELECT region, sum(revenue) FROM data GROUP BY region"
    delimiter: str = ","
    limit: int = _DEFAULT_LIMIT


class FileQueryResponse(BaseModel):
    columns: list[str]
    rows: list[list]
    truncated: bool = False


def _validate_read_only(sql_text: str) -> None:
    body = sql_text.strip().rstrip(";").strip()
    if not body:
        raise HTTPException(400, "query is empty")
    if ";" in body:
        raise HTTPException(400, "only a single statement is allowed")
    lowered = body.lower()
    if not (lowered.startswith("select") or lowered.startswith("with")):
        raise HTTPException(400, "only SELECT statements are allowed")
    for kw in _FORBIDDEN_KEYWORDS:
        if kw in lowered:
            raise HTTPException(400, f"query contains a disallowed keyword: {kw}")


def _ensure_limit(sql: str, limit: int) -> str:

    
    sql = sql.strip().rstrip(";").strip()
    
    if re.search(r"\bLIMIT\s+\d+", sql, re.IGNORECASE):
        return sql
    
    return f"{sql} LIMIT {limit}"


@router.post("/api/analytics/query", response_model=FileQueryResponse)
async def query_csv(req: FileQueryRequest) -> FileQueryResponse:
    _validate_read_only(req.sql)

    if req.delimiter not in _ALLOWED_DELIMS:
        raise HTTPException(400, f"unsupported delimiter {req.delimiter!r}")

    if len(req.csv_content.encode("utf-8")) > 50 * 1024 * 1024:
        raise HTTPException(400, "csv_content exceeds the 50MB limit for ad-hoc queries")

    

    tmp_path = None
    con = None
    try:
        with tempfile.NamedTemporaryFile(mode="w", suffix=".csv", delete=False) as tmp:
            tmp.write(req.csv_content)
            tmp_path = tmp.name

        con = duckdb.connect(database=":memory:")
        con.execute(
            f"CREATE VIEW data AS SELECT * FROM read_csv_auto('{tmp_path}', delim='{req.delimiter}')"
        )
        cursor = con.execute(sql_with_limit)
        
        rows = cursor.fetchall()
        columns = [c[0] for c in cursor.description] if cursor.description else []
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(400, f"query failed: {e}")
    finally:
        if con is not None:
            con.close()
        if tmp_path is not None and os.path.exists(tmp_path):
            os.unlink(tmp_path)

    
    truncated = len(rows) > req.limit
    if truncated:
        rows = rows[:req.limit]

    return FileQueryResponse(columns=columns, rows=rows, truncated=truncated)