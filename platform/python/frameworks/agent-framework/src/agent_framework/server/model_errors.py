"""Turns a failure raised while an agent runs into the contract a client can act on without
reading any text: an HTTP status, a stable machine-readable `code`, and whether re-sending the same
message is worth offering (`retryable`). Pure - no I/O, no framework state - so every mapping is a
table test (tests/test_model_errors.py).

Only failures that come from the model provider or the network are classified; anything else (a bug
in a tool, a programming error) returns None so the route re-raises it and the caller still gets a
plain 500 - the same as before this module existed. `message` is always a fixed generic sentence:
the provider's own error body can contain request details and is never forwarded to a client (it is
still logged by the route)."""

import asyncio
from dataclasses import dataclass

import httpx
from pydantic_ai.exceptions import ContentFilterError, ModelAPIError, ModelHTTPError, UnexpectedModelBehavior

try:
    # pydantic-ai 2 and the provider SDKs (google-genai, openai, anthropic) send requests with `httpx2`, a fork of httpx under
    # its own import name: its errors are not instances of httpx's. google-genai lets them out unwrapped, so both are matched.
    import httpx2
except ImportError:  # pydantic-ai 1.x
    httpx2 = None  # type: ignore[assignment]

_HTTP_LIBRARIES = tuple(library for library in (httpx, httpx2) if library is not None)
_TIMEOUTS = (TimeoutError, asyncio.TimeoutError, *(library.TimeoutException for library in _HTTP_LIBRARIES))
_TRANSPORT_ERRORS = tuple(library.TransportError for library in _HTTP_LIBRARIES)
_STATUS_ERRORS = tuple(library.HTTPStatusError for library in _HTTP_LIBRARIES)


@dataclass(frozen=True)
class ModelFailure:
    status: int
    code: str
    retryable: bool
    message: str
    retry_after: int | None = None


_UNAVAILABLE_MESSAGE = "The assistant's model is busy or unreachable right now. Please try again in a moment."
_TIMEOUT_MESSAGE = "The assistant took too long to answer. Please try again."
_REJECTED_MESSAGE = "The model provider rejected the request. Check the API key and model settings on the server."
_FILTERED_MESSAGE = "The model declined to answer this message."
_BAD_OUTPUT_MESSAGE = "The assistant returned an unusable answer. Please try again."

# Used by the stream route for a failure that is not a model/network one: the response has already
# started, so there is no 500 to send - the client gets this as an `error` event instead.
INTERNAL = ModelFailure(status=500, code="internal", retryable=True, message="Something went wrong on the server. Please try again.")

_MAX_RETRY_AFTER_SECONDS = 60


def classify_model_error(exc: BaseException) -> ModelFailure | None:
    """The ModelFailure for `exc`, or None when it is not a model/network failure."""
    if isinstance(exc, ExceptionGroup):
        return _classify_group(exc)
    if isinstance(exc, ModelHTTPError):
        return _classify_http_status(exc)
    if isinstance(exc, ContentFilterError):
        return ModelFailure(status=422, code="content_filtered", retryable=False, message=_FILTERED_MESSAGE)
    if isinstance(exc, UnexpectedModelBehavior):
        return ModelFailure(status=502, code="bad_output", retryable=True, message=_BAD_OUTPUT_MESSAGE)
    if isinstance(exc, ModelAPIError):
        # A provider SDK's own connection/timeout error, wrapped by pydantic-ai without a status.
        return _timeout() if _caused_by_timeout(exc) else _unavailable()
    if isinstance(exc, _TIMEOUTS):
        return _timeout()
    if isinstance(exc, _TRANSPORT_ERRORS):
        return _unavailable()
    if isinstance(exc, _STATUS_ERRORS) and exc.response.status_code >= 500:
        return _unavailable()
    return None


def _classify_http_status(exc: ModelHTTPError) -> ModelFailure:
    status = exc.status_code
    if status in (408, 504):
        return _timeout()
    if status == 429 or status >= 500:
        return _unavailable(retry_after=_clamp_retry_after(exc.retry_after))
    return ModelFailure(status=502, code="rejected", retryable=False, message=_REJECTED_MESSAGE)


def _classify_group(group: ExceptionGroup) -> ModelFailure | None:
    # pydantic-ai raises a FallbackExceptionGroup when every model in a fallback chain failed. A
    # retryable member wins: if any model was merely busy, offering a resend is right.
    failures = [failure for member in group.exceptions if (failure := classify_model_error(member)) is not None]
    if not failures:
        return None
    return next((failure for failure in failures if failure.retryable), failures[0])


def _caused_by_timeout(exc: BaseException) -> bool:
    cause = exc.__cause__
    while cause is not None:
        if isinstance(cause, _TIMEOUTS):
            return True
        cause = cause.__cause__
    return False


def _clamp_retry_after(seconds: float | None) -> int | None:
    if seconds is None:
        return None
    return max(1, min(int(seconds), _MAX_RETRY_AFTER_SECONDS))


def _unavailable(retry_after: int | None = None) -> ModelFailure:
    return ModelFailure(status=503, code="unavailable", retryable=True, message=_UNAVAILABLE_MESSAGE, retry_after=retry_after)


def _timeout() -> ModelFailure:
    return ModelFailure(status=504, code="timeout", retryable=True, message=_TIMEOUT_MESSAGE)
