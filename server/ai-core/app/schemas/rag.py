from typing import Any

from pydantic import BaseModel, Field


class DocumentBase(BaseModel):
    content: str
    source: str
    source_type: str = Field(..., pattern="^(url|pdf|text|markdown|api)$")
    doc_id: str
    page_title: str | None = ""
    chapter: str | None = ""
    metadata: dict[str, Any] = Field(default_factory=dict)


class DocumentChunk(DocumentBase):
    chunk_index: int = Field(ge=0)


class IngestRequestSchema(BaseModel):
    session_id: str
    documents: list[DocumentBase]


class IngestResponseSchema(BaseModel):
    success: bool
    count: int
    docs_ids: list[str] = Field(default_factory=list)
    error: str | None = None
