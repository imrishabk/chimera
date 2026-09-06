# Ingestion Pipeline Spec — File Extraction to Markdown for RAG

## 1. Goal

Add missing file-ingestion stage to Chimera: accept `docx, pdf, txt, md, html/url` uploads, extract text, normalize to a Markdown string, and feed it into the existing RAG path without breaking current contracts.

Non-goals:

* No changes to chunking, embedding, vector search logic.
* No changes to `AIService` RPC shapes unless explicitly approved.
* No OCR for scanned PDFs / images in v1.

## 2. Proto Contract (source of truth)

`proto/ai_core/v1/ai_core.proto`:

```proto
service AIService {
  rpc Chat(ChatRequest) returns (ChatResponse);
  rpc ChatStream(ChatRequest) returns (stream ChatResponse);
  rpc IngestDocuments(IngestRequest) returns (IngestResponse);
  rpc QueryRAG(QueryRequest) returns (QueryResponse);
  rpc Health(HealthRequest) returns (HealthResponse);
}

message IngestRequest {
  string session_id = 1;
  repeated Document documents = 2;
}

message Document {
  string content = 1;
  string source = 2;
  string source_type = 3; // "url" | "pdf" | "text" | "markdown" | "api"
  string doc_id = 4;
  int32 chunk_index = 5;
  string page_title = 6;
  string chapter = 7;
  map<string, string> metadata = 8;
}

message IngestResponse {
  bool success = 1;
  int32 count = 2;
  repeated string docs_id = 3;
  string error = 4;
}
```

Constraints derived from proto + Python:

* `Document.content`: must be non-empty Markdown string. This is the only field `chunker.py` reads.
* `Document.source`: human-readable origin. Use original filename or URL. Example: `report.pdf`, `https://example.com/page`.
* `Document.source_type`: MUST be one of `url|pdf|text|markdown|api` until proto + `services/ai-core/app/schemas/rag.py:9 DocumentBase` regex are migrated together. Do not send `docx`, `html`, `csv`.
* `Document.doc_id`: MUST be non-empty. Python `DocumentBase.doc_id: str` is required. Use `uuidv4` per file, shared across all pages/chunks of that file.
* `Document.chunk_index`: leave `0`. Python re-chunks and overwrites it in `app/services/chunker.py:22`. Go `internal/service/rag.go:45` already defaults it by slice position; do not rely on it.
* `Document.page_title`, `chapter`: optional, pass through if known (docx heading, html `<title>`). Else empty.
* `Document.metadata`: `map<string,string>`. All values must be strings. Required keys: `session_id` is added server-side; client MUST add `original_filename`, `original_mime`, `checksum_sha256`, `page_count`. See §7.
* `IngestRequest.session_id`: UUID string. Must match an existing session.
* Batching: Go `internal/service/rag.go:59` already splits at 50 docs per gRPC call. Pipeline MUST emit one `Document` per file-page or per capped section (see §6), not one per chunk. Python does chunking.

## 3. Current Codebase (do not break)

* Entry: `services/worker/internal/handler/ingest.go:28 Push` — JSON only (`model.IngestRequest`). Keep as-is.
* Job tracking: `services/worker/internal/service/ingest_job.go:26 CreateJob`, `services/worker/internal/repo/ingest_job.go`, table `services/worker/migrations/20260828000000_add_ingest_jobs.sql`.
* RAG send: `services/worker/internal/service/rag.go:25 IngestDocuments` -> `services/worker/internal/grpc/client/client.go:52`.
* Python receive: `services/ai-core/app/grpc/server.py:112 IngestDocuments` -> `app/services/chunker.py:7` -> `app/services/vector_store.py:54`.
* Routes: `services/worker/internal/routes/routes.go:54 /ingestion`. Auth via `middleware.AuthMiddlewareValidated`.
* Validation: `services/worker/internal/model/request.go:28 IngestRequest`, `internal/validator/`.

Pre-existing bugs to fix first (separate commits, before new code):

1. `services/ai-core/app/grpc/server.py:134-140` metadata mapping is swapped (`source_type: chunk.doc_id`, `page_title: chunk.chapter`). Fix to `source_type: chunk.source_type`, `page_title: chunk.page_title`, `chapter: chunk.chapter`.
2. `services/worker/internal/service/ingest_job.go:55` is synchronous gRPC inside HTTP handler. New upload path MUST be async (202 + background worker). Optionally refactor `CreateJob` to enqueue.
3. `services/worker/cmd/worker/main.go:65-68` wires `RAG`/`IngestJob` manually instead of `service.NewServicesWithRAG`. Unify so `handler/handlers.go:22 NewHandlers` always sees both.

## 4. Architecture Decision

Extraction lives in Go `services/worker`. Rationale:

* Matches `README.md` division: Go does ingestion/fetch/clean/batching, Python does chunk/embed/retrieve.
* No new Python deps, no proto migration, no change to vector store.
* Go failure modes (bad zip, encrypted PDF) never crash `ai-core`.

```
[client] --multipart--> [worker POST /api/ingestion/upload]
                              |
                              v
                     ingest.Pipeline.Validate
                              |
                              v
                     ingest_jobs row (pending) -> 202 {job_id}
                              |
                              v (goroutine, ctx with timeout)
                     Detect MIME -> Extract -> ToMarkdown -> Clean
                              |
                              v
                     []model.Document (Content=markdown, SourceType in proto enum)
                              |
                              v
                     RAGService.IngestDocuments (reuse, batch 50)
                              |
                              v
                     ai-core IngestDocuments -> chunk -> pgvector
                              |
                              v
                     ingest_jobs UpdateStatus completed/failed
[client] --poll--> GET /api/ingestion/{jobId}
```

## 5. API Spec

### 5.1 Keep existing

`POST /api/ingestion/` JSON `{session_id, documents[]|content, source}` — unchanged.

### 5.2 New (additive)

`POST /api/ingestion/upload`

* Auth: same protected group as `/ingestion`.
* Content-Type: `multipart/form-data`.
* Fields:
  * `session_id`: string (uuid, required).
  * `files`: one or more files (required, `min=1`).
  * `source_type_hint` (optional): `url|pdf|text|markdown|api`. Ignored if MIME sniff disagrees.
  * `page_title` (optional): applied when single file.
* Limits (enforce in handler, return 400/413):
  * `MAX_UPLOAD_MB_PER_FILE=25`, `MAX_FILES_PER_REQUEST=10`, `MAX_TOTAL_MB_PER_REQUEST=100`.
  * Allowlist MIME (sniffed via `net/http.DetectContentType` + extension cross-check):
    * `text/plain (.txt)`, `text/markdown (.md,.markdown)`, `application/pdf (.pdf)`,
      `application/vnd.openxmlformats-officedocument.wordprocessingml.document (.docx)`,
      `text/html (.html,.htm)` + `url` fetch path (§6.5).
  * Reject: encrypted/password PDFs, zip-bombs (`compressed_ratio > 100` or `uncompressed > 200MB`), empty files, `NUL`-only files.
* Success: `202 Accepted` with body:
  ```json
  {"success": true, "data": {"id": "<job-uuid>", "status": "pending", "session_id": "<uuid>", "source": "<first-filename>", "source_type": "markdown"}}
  ```
* Errors: `400` validation/mime, `401` auth, `404` session not found, `413` too large, `503` queue full.
* Poll: existing `GET /api/ingestion/{jobId}`, `GET /api/ingestion/list/{sessionId}` — no change.

Example:

```bash
curl -X POST http://localhost:8000/api/ingestion/upload \
  -H "Authorization: Bearer $TOKEN" \
  -F "session_id=00000000-0000-0000-0000-000000000001" \
  -F "files=@report.pdf" \
  -F "files=@notes.docx"
```

## 6. Pipeline Stages

New package `services/worker/internal/ingest/`. Handler MUST NOT contain parsing logic; service MUST NOT import PDF/DOCX libs directly — only via `ingest` interfaces.

### 6.1 Interface

```go
type MarkdownDoc struct {
  Markdown     string
  Source      string // original filename or URL
  OrigMime    string
  PageCount   int
  PageTitle   string
  ChecksumSHA256 string
}

type Extractor interface {
  Supports(sniffedMIME, ext string) bool
  Extract(ctx context.Context, r io.Reader, filename string) (MarkdownDoc, error)
}
```

`Pipeline.Execute(ctx, sessionID uuid.UUID, files []OpenedFile) ([]model.Document, error)` orchestrates: Validate -> Extract (per file, fail-fast per file, collect per-file errors) -> Clean -> Map to `model.Document`.

### 6.2 Stage 0 — Validate + stage to disk

1. Parse multipart with `r.MultipartReader()`, stream to `os.CreateTemp("", "chimera-upload-*")`. Never hold full file in RAM.
2. `io.CopyN` up to `MAX + 1` byte to detect overflow. If overflow, delete temp, return 413.
3. Sniff first 512 bytes. Compare against allowlist. Extension/MIME mismatch -> 400.
4. Compute `sha256` while writing (io.MultiWriter). Record size.
5. Optional dedup: if checksum already completed for this `session_id` (query by checksum if column exists, else skip v1), return cached `doc_id` without re-ingest.

### 6.3 Stage 1 — Extract per format

* `txt.go` (`text/plain`): read as UTF-8, replace invalid runes, keep line breaks. If `DetectContentType` says binary, reject.
* `md.go` (`text/markdown`): pass through after sanitizing NUL bytes. Do not render.
* `pdf.go` (`application/pdf`): extract text per page in order (suggested lib: `ledongthoma/pdf` or equivalent permissive). Rules:
  * Reject encrypted/no-extract permission with clear error `pdf encrypted or extract forbidden`.
  * Emit `## Page N` separator between pages.
  * Preserve paragraph breaks; collapse >2 blank lines.
  * If page yields 0 chars and page count > 0 for all pages -> error `no extractable text (scanned image? OCR out of scope)`.
  * Cap: `MAX_PAGES=500`, `MAX_CHARS_PER_FILE=1_000_000`. Beyond -> truncate + set `metadata.truncated=true`.
* `docx.go` (`...wordprocessingml.document`): unzip `word/document.xml` (do not use generic zip-slip-vulnerable code; validate paths). Map:
  * `w:pStyle Heading1-3` -> `#/##/###`, other paragraphs -> blank-line separated.
  * `w:b` -> `**x**`, `w:i` -> `*x*`.
  * `w:numPr` -> `- ` list items.
  * `w:tbl` -> GitHub markdown table `| a | b |`.
  * `w:br`, `w:tab` -> newline / space. Drop images, footnotes, headers/footers in v1 (note in metadata `dropped_images=n` if easy).
* `html.go` / URL: v1 supports uploaded `.html` only. URL fetching (`http.Get` + readability + `html-to-markdown`) is phase 2 behind `ALLOW_URL_FETCH=false` default with SSRF guard (block private CIDRs, 10s timeout, 5MB cap). If requested in v1, return 400 `url fetch disabled`.

Each extractor returns exactly one `MarkdownDoc` per file. Page splitting into multiple `model.Document`s happens in §7 if markdown exceeds cap.

### 6.4 Stage 2 — Clean (`clean.go`)

Pure function `CleanMarkdown(s string) string`, unit-testable:

1. Unicode NFC normalize, strip `U+0000`, replace other `Cc` (except `\n\t`) with space.
2. Normalize line endings `\r\n->\n`.
3. Collapse 3+ newlines to 2. Trim trailing spaces per line. Trim file.
4. Drop empty headings / repeated boilerplate headers-footers (exact-duplicate line appearing on >50% of pdf pages -> remove after first).
5. Enforce min: after clean `< 10` non-space chars -> error `empty after cleaning`.
6. Prepend title block if missing: `# {filename}\n\n`.

Do NOT chunk, embed, summarize, or LLM-rewrite here.

### 6.5 Stage 3 — Map to proto-compatible Documents

For each cleaned `MarkdownDoc`:

* `docID = uuid.NewString()` per file.
* If `len(markdown) <= 60_000` chars: one `model.Document`.
* Else split on `\n\n` boundaries into ~60k-char sections, one `model.Document` per section with same `docID`, distinct `Source` suffix `filename#part=N/M` only if split. Keep `ChunkIndex` as section ordinal (Python re-chunks anyway).
* Field mapping:

| `model.Document` | value |
|---|---|
| `Content` | cleaned markdown |
| `Source` | original filename or URL |
| `SourceType` | proto-enum normalized: `pdf->pdf`, `txt->text`, `md->markdown`, `docx->markdown`, `html->url` |
| `DocID` | file-level uuid |
| `PageTitle` | from docx/html `<title>` or empty |
| `Chapter` | empty in v1 |
| `Metadata` | `original_filename, original_mime, original_ext, checksum_sha256, page_count, char_count, truncated ("true"/"")` |

All metadata values strings. Never put raw file bytes in metadata.

## 7. Job Lifecycle + DB

Reuse `ingest_jobs(status IN pending,processing,completed,failed)`.

1. Handler creates row `pending` with `Source=first filename`, `SourceType=markdown`, `DocCount=len(files)` then spawns `go pipeline.Run(jobID)` and returns 202.
2. Worker sets `processing`, runs §6, calls `RAGService.IngestDocuments`, sets `completed` (`doc_count=resp.Count`) or `failed` (`error=first per-file error + gRPC error`).
3. Context: `context.WithTimeout(r.Context(), 10*time.Minute)` detached from HTTP disconnect via `context.WithoutCancel`.
4. Retry: no auto-retry in v1 except gRPC `Unavailable` once after 2s. Record error verbatim.

Additive migration only (do not edit `20260828000000_add_ingest_jobs.sql`):

```sql
-- +goose Up
ALTER TABLE ingest_jobs ADD COLUMN IF NOT EXISTS file_name TEXT;
ALTER TABLE ingest_jobs ADD COLUMN IF NOT EXISTS mime_type TEXT;
ALTER TABLE ingest_jobs ADD COLUMN IF NOT EXISTS file_size BIGINT DEFAULT 0;
ALTER TABLE ingest_jobs ADD COLUMN IF NOT EXISTS checksum TEXT;
-- +goose Down
ALTER TABLE ingest_jobs DROP COLUMN IF EXISTS checksum;
ALTER TABLE ingest_jobs DROP COLUMN IF EXISTS file_size;
ALTER TABLE ingest_jobs DROP COLUMN IF EXISTS mime_type;
ALTER TABLE ingest_jobs DROP COLUMN IF EXISTS file_name;
```

Phase 2 may add `UNIQUE(session_id, checksum)` for dedup.

## 8. Config (env, `services/worker/.env.example`)

```
MAX_UPLOAD_MB_PER_FILE=25
MAX_FILES_PER_REQUEST=10
MAX_TOTAL_MB_PER_REQUEST=100
MAX_PAGES_PDF=500
MAX_CHARS_PER_FILE=1000000
INGEST_WORKER_TIMEOUT_MIN=10
INGEST_QUEUE_SIZE=100
ALLOW_URL_FETCH=false
TMPDIR=/tmp
```

## 9. Security

* Auth required; `extractUserID` + session ownership check before enqueue (reuse `ChatService` session check pattern).
* MIME sniff, not extension trust. Double-extension (`report.pdf.exe`) rejected.
* Temp files `0600`, `defer Remove`, size-capped streaming.
* docx zip: reject absolute paths, `..`, symlinks, >1000 entries, >200MB uncompressed.
* PDF: reject encrypted, external stream fetches disabled.
* URL fetch (phase 2): allowlist http/https, block `169.254.0.0/16,10/8,172.16/12,192.168/16,127/8,::1`, redirect max 3, timeout 10s.
* Log filename/mime/size/checksum, never file content.

## 10. Observability

* Structured logs per job: `job_id, session_id, filename, mime, size, pages, chars, duration_ms, status, error`.
* Job `error` column holds user-safe message; full stack in server logs only.
* `GET /health` unchanged. Consider `GET /api/ingestion/{jobId}` already covers polling.

## 11. Testing (agent must add)

* `internal/ingest/*_test.go`: golden files in `internal/ingest/testdata/` (`sample.pdf`, `sample.docx`, `sample.txt`, `sample.md` + `.expected.md`). Assert exact markdown after `Extract+Clean`.
* Edge tests: empty file, encrypted PDF (expect error), zip-slip docx, oversize (413), MIME mismatch, NUL bytes, 1M+ char split into N documents.
* Handler test: `multipart` upload -> 202, mock `IngestJobService`, assert temp cleanup.
* Service test: background run with mock `RAGService` asserting `SourceType ∈ {url,pdf,text,markdown,api}`, `DocID` non-empty, metadata keys present, batching preserved.
* Python: no new tests unless fixing §3 bug; add regression test for swapped metadata fields.
* Run `go vet ./...`, `go test ./...` in `services/worker`; `pytest` in `services/ai-core` before marking done.

## 12. Implementation Checklist (in order)

- [x] Fix Python metadata swap (`server.py:134-140`) + regression test (`services/ai-core/tests/test_ingest_metadata.py`, mocked, no DB/LLM).
- [x] Add env + config loader for §8 limits (`internal/config/config.go`).
- [x] Add `+goose` migration for `file_name/mime_type/file_size/checksum` (`20260906000000` applied).
- [x] Create `internal/ingest/` interfaces + `clean.go` + tests.
- [x] Implement `txt.go`, `md.go` (via TXTExtractor) + golden tests.
- [x] Implement `pdf.go` + golden test + encrypted/empty-image error paths. (`testdata/sample.pdf` + `.expected.md`, `TestGoldenPDFStructure`, `TestPDFRejectsNonPDF`, `TestPDFEnforcesPageLimit`; word-spacing fix via `joinRow`.)
- [x] Implement `docx.go` + golden test (headings/bold/list/table).
- [x] Implement `pipeline.go` mapping (§6.5) + split-at-60k test.
- [x] Add `POST /api/ingestion/upload` handler (streaming, 202, async goroutine with semaphore `INGEST_QUEUE_SIZE`) + route + validator.
- [x] Wire in `main.go` via `NewServicesWithRAG` (manual RAG+IngestJob wiring, `POST /ingestion/upload` route).
- [x] Update `.env.example`, this spec status, and `README.md` ingestion section. (`.env.example` + root `README.md` ingestion pipeline section done.)
- [x] Phase 2: URL fetch with SSRF guard (`internal/ingest/fetch.go`, `POST /ingestion/fetch`), `html.go` golden, `pptx`/`csv` extractors + goldens, dedup unique index (`20260907000000`, `FindBySessionChecksum`, idempotent `EnqueueUpload`).

## 13. Acceptance Criteria

* Uploading `txt/md/pdf/docx/html/csv/pptx` via `POST /api/ingestion/upload` returns 202 with pollable `job_id`. Verified (handler tests + real-DB e2e).
* Poll transitions `pending->processing->completed`; `doc_count` equals gRPC `count`. Verified (real-DB e2e).
* Python receives only proto-enum `source_type`, non-empty `content`/`doc_id`, string metadata with provenance keys. Verified (unit + live ingest).
* `QueryRAG` returns citations grounded in uploaded file content. Verified live 2026-09-06 against ai-core :50051: distinctive-fact doc ingested (count=1), query answered with exact fact + `[Source 1]`, source metadata `source_type=pdf`, `doc_id`, `page_title`, `chapter` correct. Note: free-tier LLM overloaded 1 of 3 attempts (upstream flake, retrieval unaffected).
* No regression: `go vet ./...` clean, full `go test ./...` green (incl. previously-failing handler harness tests, fixed by routing through `AppHandler` + wrapping validation errors in `ingest.go`/`query.go`); `tests/test_ingest_metadata.py` passes.
