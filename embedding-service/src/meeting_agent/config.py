from functools import lru_cache
from typing import Literal

from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_file=".env", strict=False, extra="ignore")

    log_level: int | Literal["trace", "debug", "info", "notice", "warn", "error"] = (
        "debug"
    )
    openrouter_api_key: str
    logfire_api_key: str


@lru_cache
def get_settings() -> Settings:
    settings = Settings()
    return settings
