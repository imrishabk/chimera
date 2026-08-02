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
- **Core Framework/Library**: FastAPI, Langchain, OpenAI SDK, GoFiber, Astro
- **Database**: PostgreSQL with PGVector
- **API**: RESTful API, gRPC
- **DevOps**: GitHub Actions, Docker
