"""Default provider: OpenRouter, supported natively as a PydanticAI model string
('openrouter:<model>'). Reads OPENROUTER_API_KEY from the environment (PydanticAI's own
convention) — adding a direct provider (Anthropic, OpenAI) later is one more file like this,
not a new adapter interface."""


def resolve(model: str) -> str:
    return f"openrouter:{model}"
