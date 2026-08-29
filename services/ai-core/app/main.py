import asyncio
import signal
import sys

import structlog
from sqlalchemy.exc import OperationalError, ProgrammingError

from app.core.config import get_settings
from app.grpc.server import serve_grpc
from app.services.chat_history import ensure_chat_history_table
from app.services.vector_store import ensure_docs_table

logger = structlog.get_logger()


async def main():
    settings = get_settings()
    logger.info("Starting AI Core service", version=settings.version)

    # setup the tables in here
    for ensure_coro in (ensure_chat_history_table(), ensure_docs_table()):
        try:
            await ensure_coro
        except (ProgrammingError, OperationalError) as e:
            if not "already exists" in str(e):
                raise
            logger.info("Table already exists, skipping", function=ensure_coro.__name__)
        except Exception as e:  # noqa
            logger.error("Failed to initialize tables/database", error=e)
            sys.exit(1)
        else:
            logger.info("Synced the table", function=ensure_coro.__name__)

    server = await serve_grpc(port=settings.grpc_port)

    stop_event = asyncio.Event()

    def shutdown():
        logger.info("Shutdown signal received")
        stop_event.set()

    loop = asyncio.get_running_loop()
    for sig in (signal.SIGTERM, signal.SIGINT):
        loop.add_signal_handler(sig, shutdown)

    await stop_event.wait()
    logger.info("Shutting down AI Core service")
    await server.stop(grace=5)
    logger.info("gRPC server stopped")


if __name__ == "__main__":
    asyncio.run(main())
