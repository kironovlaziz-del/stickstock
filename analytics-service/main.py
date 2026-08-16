from fastapi import FastAPI

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
