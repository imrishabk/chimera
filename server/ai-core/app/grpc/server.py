import asyncio

import structlog
from grpc import aio
from langchain_core.messages import AIMessage, HumanMessage, SystemMessage
from sqlalchemy import text

from app.core.config import get_settings
from app.core.exceptions import ValidationError, handle_grpc_errors
from app.schemas.rag import DocumentBase
from app.services.chat_history import (
    add_message,
    get_chat_history,
    get_messages,
)
from app.services.chunker import chunk_documents
from app.services.llm import get_llm
from app.services.vector_store import get_rag_store
from grpc_gen import ai_core_pb2, ai_core_pb2_grpc

logger = structlog.get_logger()


class AIServiceServicer(ai_core_pb2_grpc.AIServiceServicer):
    def __init__(
        self,
        llm=None,
        vector_store=None,
        get_chat_history_func=None,
        add_message_func=None,
        get_messages_func=None,
        chunker_func=None,
    ):
        self.llm = llm or get_llm
        self.vector_store = vector_store or get_rag_store
        self._get_chat_history = get_chat_history_func or get_chat_history
        self._add_message = add_message_func or add_message
        self._get_messages = get_messages_func or get_messages
        self._chunker = chunker_func or chunk_documents
        self.settings = get_settings()
        logger.info("AIServiceServicer initialized")

    @handle_grpc_errors
    async def Chat(
        self, request: ai_core_pb2.ChatRequest, _context: aio.ServicerContext
    ):
        session_id = request.session_id

        if not session_id or not session_id.strip():
            raise ValidationError("session_id is required")

        new_messages = [self._proto_to_lc_message(m) for m in request.messages]

        await self._add_message(session_id, new_messages)

        history = await self._get_messages(session_id)
        response = await self.llm().ainvoke(history)

        await self._add_message(session_id, [AIMessage(content=response.content)])

        return ai_core_pb2.ChatResponse(
            session_id=session_id,
            message=ai_core_pb2.Message(role="assistant", content=response.content),
            done=True,
        )

    @handle_grpc_errors
    async def ChatStream(
        self, request: ai_core_pb2.ChatRequest, _context: aio.ServicerContext
    ):
        session_id = request.session_id
        if not session_id or not session_id.strip():
            raise ValidationError("session_id is required")

        new_messages = [self._proto_to_lc_message(m) for m in request.messages]

        if not new_messages:
            raise ValidationError("messages is required")

        await self._add_message(session_id, new_messages)
        history = await self._get_messages(session_id)

        # Batch small provider tokens so we emit a steady stream of frames:
        # content frames (done=False) during generation, then one terminal
        # done=True frame. The terminal frame and history persist live
        # outside the loop — the previous code had them indented inside it,
        # which ended the stream after the first buffered token.
        per_chunk = max(1, self.settings.stream_tokens_per_chunk or 1)
        buffer: list[str] = []
        parts: list[str] = []

        def take_batch() -> str:
            text = "".join(buffer)
            buffer.clear()
            return text

        try:
            async for chunk in self.llm().astream(history):
                text = _chunk_text(chunk)
                if not text:
                    continue
                parts.append(text)
                buffer.append(text)
                if len(buffer) >= per_chunk:
                    yield ai_core_pb2.ChatResponse(
                        session_id=session_id,
                        message=ai_core_pb2.Message(
                            role="assistant", content=take_batch()
                        ),
                        done=False,
                    )
            rest = take_batch()
            if rest:
                yield ai_core_pb2.ChatResponse(
                    session_id=session_id,
                    message=ai_core_pb2.Message(role="assistant", content=rest),
                    done=False,
                )
            yield ai_core_pb2.ChatResponse(
                session_id=session_id,
                message=ai_core_pb2.Message(role="assistant", content=""),
                done=True,
            )
        finally:
            # Persist the reply (or its partial prefix on disconnect) so
            # history stays complete for follow-up turns.
            if parts:
                try:
                    await self._add_message(
                        session_id, [AIMessage(content="".join(parts))]
                    )
                except Exception:
                    logger.warning(
                        "Failed to persist streamed reply", session_id=session_id
                    )

    @handle_grpc_errors
    async def GetChatHistory(
        self, request: ai_core_pb2.ChatHistoryRequest, _context: aio.ServicerContext
    ):
        session_id = request.session_id
        if not session_id or not session_id.strip():
            raise ValidationError("session_id is required")

        history = await self._get_messages(session_id)
        return ai_core_pb2.ChatHistoryResponse(
            session_id=session_id,
            messages=[
                ai_core_pb2.Message(
                    role=_lc_role(m), content=_chunk_text(m) or ""
                )
                for m in history
            ],
        )

    @handle_grpc_errors
    async def IngestDocuments(
        self, request: ai_core_pb2.IngestRequest, _context: aio.ServicerContext
    ):
        session_id = request.session_id
        if not session_id or not session_id.strip():
            raise ValidationError("session_id is required")

        if not request.documents:
            raise ValidationError("documents list cannot be empty")

        # documents = [self._proto_to_document(d) for d in request.documents]
        documents = await asyncio.gather(
            *(self._proto_to_document(doc) for doc in request.documents)
        )

        chunks = self._chunker(documents)

        from langchain_core.documents import Document as LCDocument

        lc_documents = []
        for chunk in chunks:
            metadata = {
                "session_id": session_id,
                "source": chunk.source,
                "source_type": chunk.source_type,
                "doc_id": chunk.doc_id,
                "chunk_index": chunk.chunk_index,
                "page_title": chunk.page_title,
                "chapter": chunk.chapter,
                **chunk.metadata,
            }
            lc_documents.append(
                LCDocument(page_content=chunk.content, metadata=metadata)
            )
        store = await self.vector_store()
        ids = await store.aadd_documents(lc_documents)

        return ai_core_pb2.IngestResponse(
            success=True, count=len(lc_documents), docs_id=ids, error=""
        )

    @handle_grpc_errors
    async def QueryRAG(
        self, request: ai_core_pb2.QueryRequest, _context: aio.ServicerContext
    ):
        session_id = request.session_id
        if not session_id or not session_id.strip():
            raise ValidationError("session_id is required")

        query = request.query
        if not query or not query.strip():
            raise ValidationError("query cannot be empty")

        k = request.k if request.k > 0 else 4

        filter_dict = {"session_id": session_id}
        filter_dict.update(dict(request.filter))

        store = await self.vector_store()

        results = await store.asimilarity_search_with_score(
            query=query, k=k, filter=filter_dict
        )

        sources = []
        contexts_parts = []
        for i, (doc, score) in enumerate(results):
            source = ai_core_pb2.Source(
                content=doc.page_content,
                metadata={str(k): str(v) for k, v in (doc.metadata or {}).items()},
                score=float(score),
            )
            sources.append(source)
            contexts_parts.append(f"[Source {i + 1} {doc.page_content}")

        if not contexts_parts:
            return ai_core_pb2.QueryResponse(
                answer="I couldn't find relevant information to answer your question.",
                sources=[],
                session_id=session_id,
            )

        context = "\n\n".join(contexts_parts)
        prompt = f"""Answer the question using only the provided context. Cite sources using [Source N] format.

        Context: {context}

        Question: {query}

        Answer:"""

        llm = self.llm
        response = await llm().ainvoke(prompt)

        return ai_core_pb2.QueryResponse(
            answer=response.content, sources=sources, session_id=session_id
        )

    async def Health(
        self, _request: ai_core_pb2.HealthRequest, _context: aio.ServicerContext
    ):
        from google.protobuf.timestamp_pb2 import Timestamp

        from app.services.vector_store import get_engine

        status = "healthy"
        try:
            engine = get_engine()
            async with engine._pool.connect() as conn:
                await conn.execute(text("SELECT 1"))
        except Exception as e:  # noqa
            logger.warning("Health check failed", error=str(e))
            status = "unhealthy"

        timestamp = Timestamp()

        return ai_core_pb2.HealthResponse(
            status=status,
            version=self.settings.version,
            timestamp=timestamp.GetCurrentTime(),
        )

    async def _proto_to_document(self, proto_doc: ai_core_pb2.Document) -> DocumentBase:
        return DocumentBase(
            content=proto_doc.content,
            source=proto_doc.source,
            source_type=proto_doc.source_type,
            doc_id=proto_doc.doc_id,
            page_title=proto_doc.page_title,
            chapter=proto_doc.chapter,
            metadata=dict(proto_doc.metadata),
        )

    def _proto_to_lc_message(self, proto_msg: ai_core_pb2.Message):
        role_map = {
            "user": HumanMessage,
            "assistant": AIMessage,
            "system": SystemMessage,
        }
        msg_class = role_map.get(proto_msg.role, HumanMessage)
        return msg_class(content=proto_msg.content)


def _lc_role(msg) -> str:
    """Map a langchain message to the proto role vocabulary."""
    name = type(msg).__name__
    if "Human" in name:
        return "user"
    if "System" in name:
        return "system"
    if "AI" in name:
        return "assistant"
    return "user"


def _chunk_text(chunk) -> str:
    """Extract plain text from a streamed LLM chunk.

    Content is usually a string, but some providers return a list of
    content blocks. Coercing here avoids a TypeError surfacing as a
    gRPC INTERNAL error mid-stream.
    """
    content = getattr(chunk, "content", "")
    if isinstance(content, str):
        return content
    if isinstance(content, list):
        texts = []
        for block in content:
            if isinstance(block, dict):
                texts.append(str(block.get("text", "")))
            else:
                texts.append(str(getattr(block, "text", block) or ""))
        return "".join(texts)
    return str(content) if content else ""


async def serve_grpc(port: int = 50051) -> aio.Server:
    server = aio.server()
    ai_core_pb2_grpc.add_AIServiceServicer_to_server(AIServiceServicer(), server)
    server.add_insecure_port(f"[::]:{port}")
    await server.start()
    return server
