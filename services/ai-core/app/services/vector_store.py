from functools import lru_cache

from langchain_openai import OpenAIEmbeddings
from langchain_postgres import Column, PGEngine, PGVectorStore

from app.core.config import get_settings

RAG_TABLE = "rag_vec"
VECTOR_SIZE = 2048


@lru_cache
def get_embeddings() -> OpenAIEmbeddings:
    settings = get_settings()
    return OpenAIEmbeddings(
        base_url=settings.model_provider,
        model=settings.embed_model,
        api_key=settings.provider_api_key,
        check_embedding_ctx_length=False,
        model_kwargs={"encoding_format": "float"},
    )


@lru_cache
def get_engine() -> PGEngine:
    settings = get_settings()
    connection: str = f"postgresql+psycopg://{settings.db_username}:{settings.db_password}@{settings.db_hostname}:{settings.db_port}/{settings.db_database}"
    return PGEngine.from_connection_string(url=connection)


async def ensure_docs_table() -> None:
    engine = get_engine()
    await engine.ainit_vectorstore_table(
        table_name=RAG_TABLE,
        vector_size=VECTOR_SIZE,
        metadata_columns=[
            Column("session_id", "VARCHAR"),
            Column("source", "VARCHAR"),
            Column("source_type", "VARCHAR"),
            Column("doc_id", "VARCHAR"),
            Column("chunk_index", "INTEGER"),
        ],
        overwrite_existing=False,
    )


async def get_rag_store() -> PGVectorStore:
    return await PGVectorStore.create(
        engine=get_engine(),
        table_name=RAG_TABLE,
        embedding_service=get_embeddings(),
        metadata_columns=[
            "session_id",
            "source",
            "source_type",
            "doc_id",
            "chunk_index",
        ],
    )
