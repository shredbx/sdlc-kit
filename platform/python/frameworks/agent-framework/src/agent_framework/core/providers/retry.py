"""One retry setting for every model provider.

A model call that fails for a moment (the provider is overloaded, the connection drops) is tried again before the
person watching the box sees an error. The count is "retries after the first try": 2 means up to 3 tries, 0 turns
retrying off. Two is the default of both the Anthropic and OpenAI SDKs - more mostly adds waiting in front of a person.

Each provider uses its own SDK's retrying, so nothing here re-implements it:
- OpenRouter goes through the OpenAI SDK, which retries connection errors, timeouts and 408/409/429/5xx with a backoff
  of 0.5 s doubling to 8 s and honours `Retry-After` (up to two minutes; a longer one is not retried).
- Google goes through google-genai's `HttpRetryOptions`, built in google.py from the constants below.
Retries happen before the first byte of the reply arrives, so streamed text is never repeated, and they run per model:
when they are used up the error reaches `FallbackModel`, which moves on to the next model in the chain."""

DEFAULT_MAX_RETRIES = 2
MAX_RETRIES_ALLOWED = 5  # a person is waiting; beyond this a retry loop is just a longer outage

# Google only (the OpenAI SDK has its own list): worth trying again, never a request the provider rejected as wrong.
RETRYABLE_STATUS_CODES = (408, 429, 500, 502, 503, 504)
INITIAL_DELAY_SECONDS = 1.0
MAX_DELAY_SECONDS = 8.0
JITTER_SECONDS = 0.5


def check_max_retries(value: int) -> int:
    """`value` unchanged if it is a whole number from 0 to MAX_RETRIES_ALLOWED, else ValueError."""
    if isinstance(value, bool) or not isinstance(value, int) or not 0 <= value <= MAX_RETRIES_ALLOWED:
        raise ValueError(f"max_retries must be a whole number from 0 to {MAX_RETRIES_ALLOWED}, got {value!r}")
    return value
