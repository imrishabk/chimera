from functools import lru_cache

from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_file=".env", env_file_encoding="utf-8", extra="ignore"
    )

    model_provider: str = "https://openrouter.ai/api/v1"
    provider_api_key: str = ""
    chat_model: str = "nvidia/nemotron-3-ultra-550b-a55b:free"
    embed_model: str = "nvidia/nemotron-3-embed-1b:free"

    db_hostname: str = "127.0.0.1"
    db_username: str = "chimera"
    db_password: str = "chimera"
    db_database: str = "chimera"
    db_port: str = "5432"


@lru_cache
def get_settings() -> Settings:
    return Settings()
