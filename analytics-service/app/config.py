from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    """No external API keys needed — this service does everything locally
    with Polars/DuckDB/statsmodels/PyOD, no LLM calls."""

    app_env: str = "development"
    max_rows_in: int = 200_000  # guard against being sent an enormous inline payload

    class Config:
        env_prefix = ""
        case_sensitive = False


settings = Settings()
