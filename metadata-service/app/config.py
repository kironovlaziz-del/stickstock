from pydantic_settings import BaseSettings

class Settings(BaseSettings):
    redis_url: str = "redis://redis:6379"
    cache_ttl: int = 300
    service_name: str = "metadata-service"

    class Config:
        env_file = ".env"
        env_prefix = "METADATA_"

settings = Settings()
