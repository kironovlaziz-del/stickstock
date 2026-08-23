"""POST /api/analytics/anomalies — flag outliers in numeric tabular data
using PyOD's Isolation Forest, for fraud/error detection

import numpy as np
from fastapi import APIRouter, HTTPException
from pydantic import BaseModel
from pyod.models.iforest import IForest

router = APIRouter()


class AnomalyRequest(BaseModel):
    columns: list[str]
    rows: list[list[float]]
    contamination: float = 0.05  # expected proportion of outliers, 0 < x < 0.5


class AnomalyResponse(BaseModel):
    is_outlier: list[bool]
    scores: list[float]


@router.post("/api/analytics/anomalies", response_model=AnomalyResponse)
async def detect_anomalies(req: AnomalyRequest) -> AnomalyResponse:
    if len(req.rows) < 10:
        raise HTTPException(400, "need at least 10 rows to fit an outlier model")
    if not (0 < req.contamination < 0.5):
        raise HTTPException(400, "contamination must be between 0 and 0.5")

    try:
        X = np.array(req.rows, dtype=float)
    except Exception as e:
        raise HTTPException(400, f"rows must be purely numeric: {e}")

    if X.ndim != 2 or X.shape[1] == 0:
        raise HTTPException(400, "rows must be a non-empty 2D numeric array")

    model = IForest(contamination=req.contamination, random_state=42)
    try:
        model.fit(X)
    except Exception as e:
        raise HTTPException(400, f"anomaly detection failed: {e}")

    return AnomalyResponse(
        is_outlier=[bool(v) for v in model.labels_],
        scores=[float(v) for v in model.decision_scores_],
    )
