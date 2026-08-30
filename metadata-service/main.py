from fastapi import FastAPI, HTTPException, Request
from fastapi.responses import JSONResponse
import logging
from app.config import settings
from app.models import TemplateRenderRequest, TemplateRenderResponse, TemplateValidateRequest, TemplateValidateResponse
from app.template_engine import template_engine
from app.cache import cache

logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(name)s: %(message)s")
logger = logging.getLogger(__name__)

app = FastAPI(title="Metadata Service", version="2.0.0")

@app.on_event("startup")
async def startup():
    await cache.connect()
    logger.info("Metadata Service started")

@app.on_event("shutdown")
async def shutdown():
    await cache.close()
    logger.info("Metadata Service stopped")

@app.post("/template/render", response_model=TemplateRenderResponse)
async def render_template(req: TemplateRenderRequest):
    try:
        rendered = template_engine.render(req.template, req.context)
        used_params = {k: v for k, v in req.context.items() if k in req.template}
        return TemplateRenderResponse(rendered_sql=rendered, params=used_params)
    except ValueError as e:
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:
        logger.error(f"Template render error: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@app.post("/template/validate", response_model=TemplateValidateResponse)
async def validate_template(req: TemplateValidateRequest):
    result = template_engine.validate(req.template)
    return TemplateValidateResponse(**result)

@app.get("/health")
async def health():
    return {"status": "ok", "service": settings.service_name}
