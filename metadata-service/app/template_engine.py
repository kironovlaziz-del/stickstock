from jinja2 import Environment, BaseLoader, TemplateSyntaxError, UndefinedError
from typing import Dict, Any
import re
import logging

logger = logging.getLogger(__name__)

class SQLTemplateEngine:
    def __init__(self):
        self.env = Environment(
            loader=BaseLoader(),
            autoescape=False,
            trim_blocks=True,
            lstrip_blocks=True,
        )
        self.env.filters['quote'] = self._quote_filter
        self.env.filters['default'] = self._default_filter
        self.env.filters['sql_identifier'] = self._sql_identifier_filter

    def _quote_filter(self, value: str) -> str:
        if value is None:
            return "NULL"
        escaped = str(value).replace("'", "''")
        return f"'{escaped}'"

    def _default_filter(self, value: Any, default: Any = '') -> Any:
        if value is None or value == '':
            return default
        return value

    def _sql_identifier_filter(self, value: str) -> str:
        if value is None:
            return ""
        if not re.match(r'^[a-zA-Z_][a-zA-Z0-9_]*$', value):
            raise ValueError(f"Invalid identifier: {value}")
        return f'"{value}"'

    def render(self, template: str, context: Dict[str, Any]) -> str:
        if not template or not template.strip():
            raise ValueError("Template cannot be empty")
        try:
            jinja_template = self.env.from_string(template)
            rendered = jinja_template.render(**context)
            return rendered.strip()
        except TemplateSyntaxError as e:
            raise ValueError(f"Template syntax error: {e}")
        except UndefinedError as e:
            raise ValueError(f"Missing variable: {e}")
        except Exception as e:
            raise ValueError(f"Template error: {e}")

    def validate(self, template: str) -> Dict[str, Any]:
        if not template or not template.strip():
            return {"valid": False, "error": "Template cannot be empty"}
        try:
            self.env.parse(template)
            if 'import' in template or 'exec' in template:
                return {"valid": False, "error": "Dangerous keywords not allowed"}
            return {"valid": True}
        except TemplateSyntaxError as e:
            return {"valid": False, "error": f"Syntax error: {e.message}"}
        except Exception as e:
            return {"valid": False, "error": str(e)}

template_engine = SQLTemplateEngine()
