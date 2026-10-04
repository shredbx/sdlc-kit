"""Default provider: OpenRouter, supported natively by PydanticAI. Reads OPENROUTER_API_KEY from the environment
(PydanticAI's own convention) - adding a direct provider (Anthropic, OpenAI) later is one more file like this, not
a new adapter interface.

OpenRouter speaks the OpenAI protocol, so retrying is the OpenAI SDK's own (see retry.py); this only sets how many."""

from pydantic_ai.models import Model
from pydantic_ai.models.openrouter import OpenRouterModel
from pydantic_ai.providers.openrouter import OpenRouterProvider

from agent_framework.core.providers.retry import DEFAULT_MAX_RETRIES, check_max_retries


def resolve(model: str, max_retries: int = DEFAULT_MAX_RETRIES, *, http_client: object | None = None) -> Model:
    """The OpenRouter model `model` (a bare OpenRouter name such as 'google/gemini-2.5-flash'), retrying up to
    `max_retries` times. `http_client` is for tests (a client with a mock transport)."""
    check_max_retries(max_retries)
    client = OpenRouterProvider(http_client=http_client).client.with_options(max_retries=max_retries)  # type: ignore[arg-type]
    return OpenRouterModel(model, provider=OpenRouterProvider(openai_client=client))
