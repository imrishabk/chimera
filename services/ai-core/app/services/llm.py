from functools import lru_cache

from langchain_openai import ChatOpenAI

from app.core.config import get_settings


@lru_cache
def get_llm() -> ChatOpenAI:
    setting = get_settings()
    return ChatOpenAI(
        base_url=setting.model_provider,
        api_key=setting.provider_api_key,
        model=setting.chat_model,
        streaming=True,
    )
