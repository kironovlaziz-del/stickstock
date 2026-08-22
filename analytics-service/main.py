from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import duckdb
import os
from typing import List, Any, Optional

from app.aggregate import router as aggregate_router
from app.anomalies import router as anomalies_router
from app.query import router as query_router
from app.stats import router as stats_router

app = FastAPI(
    title="StickStock Analytics Service",
    description=(
        "Data processing for StickStock: DuckDB SQL over files, Polars "
        "aggregation, statsmodels regression/t-tests/forecasting, and "
        "PyOD anomaly detection. No LLM calls, no external API keys."
    ),
    version="0.2.0",
)

app.include_router(query_router, tags=["analytics"])
app.include_router(aggregate_router, tags=["analytics"])
app.include_router(stats_router, tags=["analytics"])
app.include_router(anomalies_router, tags=["analytics"])

@app.get("/health")
async def health() -> dict[str, str]:
    return {"status": "ok", "service": "stickstock-analytics"}

# ---------- new endpoints for DuckDB ----------
class DuckDBQueryRequest(BaseModel):
    file_id: str
    sql: str

class DuckDBQueryResponse(BaseModel):
    columns: List[str]
    rows: List[List[Any]]


UPLOAD_DIR = os.getenv("UPLOAD_DIR", "/data/uploads")

@app.post("/duckdb/query", response_model=DuckDBQueryResponse)
async def duckdb_query(req: DuckDBQueryRequest):
    file_path = os.path.join(UPLOAD_DIR, req.file_id + ".csv")
    if not os.path.exists(file_path):
        raise HTTPException(status_code=404, detail="File not found")

    
    con = duckdb.connect(database=':memory:', config={'enable_progress_bar': 'false'})
    try:
        
        con.execute(f"CREATE OR REPLACE VIEW data AS SELECT * FROM read_csv_auto('{file_path}')")
        
        sql_lower = req.sql.strip().lower()
        if not sql_lower.startswith('select'):
            raise HTTPException(status_code=400, detail="Only SELECT queries are allowed")
        
        result = con.execute(req.sql).fetchall()
        
        columns = [desc[0] for desc in con.description] if con.description else []
        return DuckDBQueryResponse(columns=columns, rows=result)
    except duckdb.Error as e:
        raise HTTPException(status_code=400, detail=str(e))
    finally:
        con.close()