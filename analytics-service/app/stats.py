"""POST /api/analytics/stats/* — deeper statistical analysis via
statsmodels and scipy: OLS regression, Welch's t-test, and ARIMA"""

import numpy as np
import statsmodels.api as sm
from fastapi import APIRouter, HTTPException
from pydantic import BaseModel
from scipy import stats as scipy_stats
from statsmodels.tsa.arima.model import ARIMA

router = APIRouter()


class RegressionRequest(BaseModel):
    y: list[float]
    x: list[list[float]]  # one or more predictor columns, row-aligned with y


class RegressionResponse(BaseModel):
    intercept: float
    coefficients: list[float]
    r_squared: float
    p_values: list[float]


@router.post("/api/analytics/stats/regression", response_model=RegressionResponse)
async def regression(req: RegressionRequest) -> RegressionResponse:
    if len(req.y) != len(req.x):
        raise HTTPException(400, "y and x must have the same number of rows")
    if len(req.y) < 3:
        raise HTTPException(400, "need at least 3 rows to fit a regression")

    X = sm.add_constant(np.array(req.x))
    y = np.array(req.y)

    try:
        model = sm.OLS(y, X).fit()
    except Exception as e:
        raise HTTPException(400, f"regression failed: {e}")

    return RegressionResponse(
        intercept=float(model.params[0]),
        coefficients=[float(c) for c in model.params[1:]],
        r_squared=float(model.rsquared),
        p_values=[float(p) for p in model.pvalues[1:]],
    )


class TTestRequest(BaseModel):
    sample_a: list[float]
    sample_b: list[float]


class TTestResponse(BaseModel):
    statistic: float
    p_value: float
    mean_a: float
    mean_b: float


@router.post("/api/analytics/stats/ttest", response_model=TTestResponse)
async def ttest(req: TTestRequest) -> TTestResponse:
    if len(req.sample_a) < 2 or len(req.sample_b) < 2:
        raise HTTPException(400, "each sample needs at least 2 values")

    try:
        statistic, p_value = scipy_stats.ttest_ind(req.sample_a, req.sample_b, equal_var=False)
    except Exception as e:
        raise HTTPException(400, f"t-test failed: {e}")

    return TTestResponse(
        statistic=float(statistic),
        p_value=float(p_value),
        mean_a=float(np.mean(req.sample_a)),
        mean_b=float(np.mean(req.sample_b)),
    )


class ForecastRequest(BaseModel):
    series: list[float]
    periods: int = 5
    order: list[int] = [1, 1, 1]  # (p, d, q)


class ForecastResponse(BaseModel):
    forecast: list[float]


@router.post("/api/analytics/stats/forecast", response_model=ForecastResponse)
async def forecast(req: ForecastRequest) -> ForecastResponse:
    if len(req.series) < 10:
        raise HTTPException(400, "need at least 10 points for a meaningful ARIMA fit")
    if len(req.order) != 3:
        raise HTTPException(400, "order must be [p, d, q]")
    if req.periods < 1:
        raise HTTPException(400, "periods must be at least 1")

    try:
        model = ARIMA(req.series, order=tuple(req.order)).fit()
        prediction = model.forecast(steps=req.periods)
    except Exception as e:
        raise HTTPException(400, f"forecast failed: {e}")

    return ForecastResponse(forecast=[float(v) for v in prediction])
