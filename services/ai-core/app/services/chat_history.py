from langchain_core.messages import BaseMessage
from langchain_postgres import PostgresChatMessageHistory
from psycopg import AsyncConnection

from app.core.config import get_settings

CHAT_HISTORY_TABLE = "chat_history"


def _connection_string() -> str:
    settings = get_settings()
    return f"postgresql://{settings.db_username}:{settings.db_password}@{settings.db_hostname}:{settings.db_port}/{settings.db_database}"


_connection: AsyncConnection | None = None


async def get_connection() -> AsyncConnection:
    global _connection
    if _connection is None or _connection.closed:
        _connection = await AsyncConnection.connect(_connection_string())
        return _connection


async def ensure_chat_history_table() -> None:
    conn = await get_connection()
    return PostgresChatMessageHistory.acreate_tables(conn, CHAT_HISTORY_TABLE)


async def get_chat_history(session_id: str) -> PostgresChatMessageHistory:
    conn = await get_connection()
    return PostgresChatMessageHistory(
        CHAT_HISTORY_TABLE, session_id, async_connection=conn
    )


async def add_message(session_id: str, messages: list[BaseMessage]) -> None:
    history = await get_chat_history(session_id)
    await history.aadd_messages(messages)


async def get_messages(session_id: str) -> list[BaseMessage]:
    history = await get_chat_history(session_id)
    return await history.aget_messages()


async def clear_history(session_id: str) -> None:
    history = await get_chat_history(session_id)
    await history.aclear()
