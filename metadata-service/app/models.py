from typing import List, Optional, Any, Dict
from pydantic import BaseModel

class TemplateRenderRequest(BaseModel):
    template: str
    context: Dict[str, Any]

class TemplateRenderResponse(BaseModel):
    rendered_sql: str
    params: Dict[str, Any]

class TemplateValidateRequest(BaseModel):
    template: str

class TemplateValidateResponse(BaseModel):
    valid: bool
    error: Optional[str] = None
