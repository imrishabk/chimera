import typing
from collections.abc import Callable
from functools import wraps
from typing import Any

import grpc
import structlog

logger = structlog.get_logger()


class AIServiceError(Exception):
    """Base exception for AI service"""

    grpc_code = grpc.StatusCode.INTERNAL


class ValidationError(AIServiceError):
    grpc_code = grpc.StatusCode.INVALID_ARGUMENT


class NotFoundError(AIServiceError):
    grpc_code = grpc.StatusCode.NOT_FOUND


class ExternalServiceError(AIServiceError):
    grpc_code = grpc.StatusCode.UNAVAILABLE


def handle_grpc_errors(method: Callable) -> Callable:
    @wraps(method)
    async def wrapper(self, request, context: grpc.aio.ServicerContext):
        try:
            return await method(self, request, context)
        except AIServiceError as e:
            context.set_code(e.grpc_code)
            context.set_details(str(e))
            logger.warning("Service error", method=method.__name__, error=str(e))
            return _empty_response(method)
        except Exception as e:
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details("Internal server error")
            logger.exception("Unhandled error", method=method.__name__, error=str(e))
            return _empty_response(method)

    return wrapper


def _empty_response(method: Callable) -> Any:
    return_type = typing.get_type_hints(method).get("return")
    if return_type:
        try:
            return return_type()
        except Exception:  # noqa
            logger.warning(
                "Could not construct empty response",
                method=method.__name__,
                return_type=return_type,
            )
    return None
