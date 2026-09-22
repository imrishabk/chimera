"""Regression test for IngestDocuments metadata mapping.

Covers the swapped-fields bug where `source_type` received `doc_id` and
`page_title` received `chapter`. Uses mocked vector store + chunker so it
runs without DB, embeddings, or LLM:

    uv run python tests/test_ingest_metadata.py
"""

import asyncio
import sys

sys.path.insert(0, ".")
sys.path.insert(0, "grpc_gen/generated")

from ai_core.v1 import ai_core_pb2  # noqa: E402

from app.grpc.server import AIServiceServicer  # noqa: E402
from app.schemas.rag import DocumentChunk  # noqa: E402

captured: dict = {}


async def fake_vector_store():
    class FakeStore:
        async def aadd_documents(self, docs):
            captured["docs"] = docs
            return [f"id-{i}" for i, _ in enumerate(docs)]

    return FakeStore()


def fake_chunker(documents):
    return [
        DocumentChunk(
            content=d.content,
            source=d.source,
            source_type=d.source_type,
            doc_id=d.doc_id,
            page_title=d.page_title,
            chapter=d.chapter,
            chunk_index=0,
            metadata=dict(d.metadata),
        )
        for d in documents
    ]


async def main() -> None:
    svc = AIServiceServicer(vector_store=fake_vector_store, chunker_func=fake_chunker)
    req = ai_core_pb2.IngestRequest(
        session_id="00000000-0000-0000-0000-000000000001",
        documents=[
            ai_core_pb2.Document(
                content="hello markdown",
                source="report.pdf",
                source_type="pdf",
                doc_id="doc-123",
                page_title="Intro",
                chapter="Ch1",
                metadata={"original_filename": "report.pdf"},
            ),
            ai_core_pb2.Document(
                content="second doc body",
                source="notes.md",
                source_type="markdown",
                doc_id="doc-456",
                metadata={},
            ),
        ],
    )
    resp = await svc.IngestDocuments(req, None)
    assert resp.success is True, resp
    assert resp.count == 2, resp
    first, second = captured["docs"]
    md = first.metadata
    assert md["source"] == "report.pdf", md
    assert md["source_type"] == "pdf", md
    assert md["doc_id"] == "doc-123", md
    assert md["page_title"] == "Intro", md
    assert md["chapter"] == "Ch1", md
    assert md["session_id"] == "00000000-0000-0000-0000-000000000001", md
    assert md["original_filename"] == "report.pdf", md
    md2 = second.metadata
    assert md2["source_type"] == "markdown", md2
    assert md2["doc_id"] == "doc-456", md2
    print("PASS test_ingest_metadata: source_type/page_title/chapter mapping correct")


if __name__ == "__main__":
    asyncio.run(main())
