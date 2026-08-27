import polars as pl
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from typing import List, Any, Optional

app = FastAPI(title="Data Profiler Service")

class ProfileRequest(BaseModel):
    columns: List[str]
    rows: List[List[Any]]

class ColumnProfile(BaseModel):
    name: str
    data_type: str
    null_count: int
    null_percentage: float
    unique_count: int
    unique_percentage: float
    min: Optional[Any] = None
    max: Optional[Any] = None
    mean: Optional[float] = None
    std: Optional[float] = None

@app.post("/profile")
async def profile(req: ProfileRequest):
    if not req.columns or not req.rows:
        raise HTTPException(400, "No data")
    df = pl.DataFrame(req.rows, schema=req.columns, orient="row")
    result = []
    for col in df.columns:
        series = df[col]
        null_count = series.null_count()
        unique_count = series.n_unique()
        profile = ColumnProfile(
            name=col,
            data_type=str(series.dtype),
            null_count=null_count,
            null_percentage=(null_count / len(df)) * 100,
            unique_count=unique_count,
            unique_percentage=(unique_count / len(df)) * 100,
        )
        if series.dtype in (pl.Float64, pl.Float32, pl.Int64, pl.Int32):
            non_null = series.drop_nulls()
            if len(non_null) > 0:
                profile.min = float(non_null.min())
                profile.max = float(non_null.max())
                profile.mean = float(non_null.mean())
                profile.std = float(non_null.std())
        result.append(profile)
    return {"columns": result, "total_rows": len(df)}
