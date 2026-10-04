"""Direct Google provider: PydanticAI's GoogleProvider talks to the Gemini Developer API via the google-genai SDK,
already a transitive dependency of the top-level `pydantic-ai` package (no separate install). Reads GOOGLE_API_KEY
(or the legacy GEMINI_API_KEY) from the environment - PydanticAI's own convention, same as openrouter.py reads
OPENROUTER_API_KEY.

Left alone, the SDK retries nothing, and the Gemini API answers 503 "overloaded" now and then; so the retries are
switched on here from the shared setting (see retry.py)."""

from google.genai.types import HttpRetryOptions
from pydantic_ai.models import Model
from pydantic_ai.models.google import GoogleModel
from pydantic_ai.providers.google import GoogleProvider

from agent_framework.core.providers.retry import (
    DEFAULT_MAX_RETRIES,
    INITIAL_DELAY_SECONDS,
    JITTER_SECONDS,
    MAX_DELAY_SECONDS,
    RETRYABLE_STATUS_CODES,
    check_max_retries,
)


def retry_options(
    max_retries: int,
    *,
    initial_delay: float = INITIAL_DELAY_SECONDS,
    max_delay: float = MAX_DELAY_SECONDS,
    jitter: float = JITTER_SECONDS,
) -> HttpRetryOptions | None:
    """The SDK's retry options for `max_retries` retries (None = never retry): the delay starts at `initial_delay`,
    doubles each time up to `max_delay`, plus up to `jitter` seconds of randomness."""
    if check_max_retries(max_retries) == 0:
        return None
    return HttpRetryOptions(
        attempts=max_retries + 1,  # the SDK counts the first try
        initial_delay=initial_delay,
        max_delay=max_delay,
        exp_base=2.0,
        jitter=jitter,
        http_status_codes=list(RETRYABLE_STATUS_CODES),
    )


def resolve(
    model: str,
    max_retries: int = DEFAULT_MAX_RETRIES,
    *,
    http_client: object | None = None,
    initial_delay: float = INITIAL_DELAY_SECONDS,
    max_delay: float = MAX_DELAY_SECONDS,
    jitter: float = JITTER_SECONDS,
) -> Model:
    """The Gemini model `model` (a bare name such as 'gemini-2.5-flash'), retrying up to `max_retries` times.
    `http_client` and the delays are for tests (a mock transport, no real waiting)."""
    options = retry_options(max_retries, initial_delay=initial_delay, max_delay=max_delay, jitter=jitter)
    return GoogleModel(model, provider=GoogleProvider(http_client=http_client, retry_options=options))  # type: ignore[arg-type]
