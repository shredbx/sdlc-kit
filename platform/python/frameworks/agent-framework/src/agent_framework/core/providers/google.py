"""Direct Google provider, supported natively as a PydanticAI model string ('google:<model>').
PydanticAI's GoogleProvider talks to the Gemini Developer API via the google-genai SDK, already a
transitive dependency of the top-level `pydantic-ai` package (no separate install). Reads
GOOGLE_API_KEY (or the legacy GEMINI_API_KEY) from the environment - PydanticAI's own convention,
same as openrouter.py reads OPENROUTER_API_KEY."""


def resolve(model: str) -> str:
    return f"google:{model}"
