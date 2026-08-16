"""POST /api/analytics/aggregate — filter, group, and aggregate tabular
data with Polars. Takes the same {columns, rows} shape the Go backend
already returns from query execution, so results can be piped straight in
without a round trip through a file."""

import polars as pl
from fastapi import APIRouter, HTTPException
from pydantic import BaseModel

router = APIRouter()

_FUNC_MAP = {
    "sum": lambda c: pl.col(c).sum(),
    "mean": lambda c: pl.col(c).mean(),
    "min": lambda c: pl.col(c).min(),
    "max": lambda c: pl.col(c).max(),
    "count": lambda c: pl.col(c).count(),
    "median": lambda c: pl.col(c).median(),
    "std": lambda c: pl.col(c).std(),
}

_OP_MAP = {
    "eq": lambda c, v: pl.col(c) == v,
    "ne": lambda c, v: pl.col(c) != v,
    "gt": lambda c, v: pl.col(c) > v,
    "gte": lambda c, v: pl.col(c) >= v,
    "lt": lambda c, v: pl.col(c) < v,
    "lte": lambda c, v: pl.col(c) <= v,
}


class FilterSpec(BaseModel):
    column: str
    op: str  # eq | ne | gt | gte | lt | lte
    value: float | int | str | bool


class AggSpec(BaseModel):
    column: str
    func: str  # sum | mean | min | max | count | median | std
    alias: str | None = None


class AggregateRequest(BaseModel):
    columns: list[str]
    rows: list[list]
    filters: list[FilterSpec] = []
    group_by: list[str] = []
    aggregations: list[AggSpec] = []


class AggregateResponse(BaseModel):
    columns: list[str]
    rows: list[list]


@router.post("/api/analytics/aggregate", response_model=AggregateResponse)
async def aggregate(req: AggregateRequest) -> AggregateResponse:
    if not req.aggregations:
        raise HTTPException(400, "at least one aggregation is required")

    try:
        data = {col: [row[i] for row in req.rows] for i, col in enumerate(req.columns)}
        df = pl.DataFrame(data)
    except Exception as e:
        raise HTTPException(400, f"could not build dataframe from columns/rows: {e}")

    for f in req.filters:
        if f.column not in df.columns:
            raise HTTPException(400, f"unknown filter column {f.column!r}")
        if f.op not in _OP_MAP:
            raise HTTPException(400, f"unsupported filter op {f.op!r}")
        try:
            df = df.filter(_OP_MAP[f.op](f.column, f.value))
        except Exception as e:
            raise HTTPException(400, f"filter on {f.column!r} failed: {e}")

    agg_exprs = []
    for spec in req.aggregations:
        if spec.column not in df.columns:
            raise HTTPException(400, f"unknown aggregation column {spec.column!r}")
        if spec.func not in _FUNC_MAP:
            raise HTTPException(400, f"unsupported aggregation function {spec.func!r}")
        expr = _FUNC_MAP[spec.func](spec.column)
        if spec.alias:
            expr = expr.alias(spec.alias)
        agg_exprs.append(expr)

    try:
        if req.group_by:
            for g in req.group_by:
                if g not in df.columns:
                    raise HTTPException(400, f"unknown group_by column {g!r}")
            result = df.group_by(req.group_by, maintain_order=True).agg(agg_exprs)
        else:
            result = df.select(agg_exprs)
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(400, f"aggregation failed: {e}")

    return AggregateResponse(columns=result.columns, rows=[list(r) for r in result.rows()])
