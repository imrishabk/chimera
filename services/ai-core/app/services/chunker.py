from langchain_text_splitters import RecursiveCharacterText

from app.core.config import get_settings
from app.schemas.rag import DocumentBase, DocumentChunk


def chunk_documents(documents: list[DocumentBase]) -> list[DocumentChunk]:
    settings = get_settings()
    splitter = RecursiveCharacterText(
        chunk_size=settings.chunk_size,
        chunk_overlap=settings.chunk_overlap,
        separators=["\n\n", "\n", ". ", " ", ""],
        keep_separator=True,
    )
    chunks = []
    for doc in documents:
        texts = splitter.split_text(doc.content)
        for i, text in enumerate(texts):
            chunk = DocumentChunk(
                content=text,
                source=doc.source,
                source_type=doc.source_type,
                doc_id=doc.doc_id,
                page_title=doc.page_title,
                chapter=doc.chapter,
                chunk_index=i,
                metadata={**doc.metadata, "chunk_index": i},
            )
            chunks.append(chunk)
    return chunks
