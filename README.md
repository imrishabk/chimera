# Chimera

**Chimera** is a Multi-Agent Research & Automation
Platform.

> [!NOTE]
> Name **Chimera** comes from greek mythology creature which front body of a lion
> a goat's head from the back and a tail that ends in a snake's head. Just like
> the project which uses Go, Python and Typescript together.

This project covers a **RAG** pipeline in `python` which is used to query with an
LLM. A service, job queue to be specific in `golang` which handles document
ingestion tasks, fetching URLs, cleaning texts, batching embeddings and calling
the python service via gRPC. And a simple chat UI that streams tokens from Python
API and a dashboard which hits the Go service to show ingestion job status in
real time in `typescript`.

### Stack

- **Language**: Python, Go, Typescript
- **Core Framework/Library**: FastAPI, Langchain, OpenAI SDK, Chi, Astro
- **Database**: PostgreSQL with PGVector
- **API**: RESTful API, gRPC
- **DevOps**: GitHub Actions, Docker

### Ingestion pipeline

`worker` owns file extraction, `ai-core` owns chunking and embedding.

Flow: `POST /api/ingestion/upload` (multipart) -> staged to disk ->
`pending` job (202) -> background extract to Markdown -> existing
`RAGService.IngestDocuments` (gRPC, batch 50) -> `ai-core`
`IngestDocuments` -> chunk -> pgvector. Poll `GET /api/ingestion/{jobId}`.

- Supported: `.txt`, `.md`, `.pdf` (text-based, `## Page N` separators),
  `.docx` (headings, bold/italic, lists, GFM tables), `.pptx`
  (`## Slide N` + bullets), `.csv` (delimiter-sniffed GFM table), `.html`
  (readability-lite: `main`/`article` preferred, nav/footer/script dropped).
  Scanned-image PDFs (OCR) and encrypted PDFs are rejected with a clear
  job error.
- Everything is normalized to a Markdown string sent as
  `Document.content` with proto-enum `source_type`
  (`pdf|text|markdown|url|api`); original filename, MIME, checksum and
  page count travel in `Document.metadata`.
- `POST /api/ingestion/fetch` (`{session_id, urls[], page_title?}`)
  fetches URLs server-side behind `ALLOW_URL_FETCH` (default off) with
  SSRF guards: http/https only, private/loopback/link-local/multicast
  blocked at resolve and dial time, 3-redirect cap with re-validation,
  10s timeout, 5MB cap.
- Idempotent re-ingest: `UNIQUE(session_id, checksum)` partial index;
  re-uploading identical content returns the existing job without
  re-embedding.
- Limits (env, see `services/worker/.env.example`):
  `MAX_UPLOAD_MB_PER_FILE=25`, `MAX_FILES_PER_REQUEST=10`,
  `MAX_TOTAL_MB_PER_REQUEST=100`, `MAX_PAGES_PDF=500`,
  `MAX_CHARS_PER_FILE=1000000`, `INGEST_WORKER_TIMEOUT_MIN=10`,
  `INGEST_QUEUE_SIZE=100`, `URL_FETCH_TIMEOUT_SEC=10`,
  `URL_FETCH_MAX_MB=5`, `MAX_URLS_PER_REQUEST=10`.
- Golden extractor tests live in
  `services/worker/internal/ingest/testdata/` (`sample.{txt,md,pdf,docx,html,csv,pptx}`
  + `.expected.md`); run `go test ./internal/ingest/...`.
- Spec: `docs/ingestion-pipeline-spec.md`.
