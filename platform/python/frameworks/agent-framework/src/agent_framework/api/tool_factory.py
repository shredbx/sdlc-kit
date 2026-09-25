"""Config-driven tool factory: turn "call one endpoint, get typed JSON back" into a working Tool
from a config object — no hand-written client class + parsing code needed for that common case.
A tool with real business logic on top of the response (not just a passthrough) still gets
written by hand; this is for the simple case, not a replacement for every tool."""

from dataclasses import dataclass
from typing import Any

from pydantic import BaseModel
from pydantic_ai import Tool

from agent_framework.api.base import BaseAPIClient


@dataclass
class ApiToolConfig:
    name: str
    description: str
    client: BaseAPIClient
    method: str
    path: str
    input_schema: type[BaseModel]
    output_schema: type[BaseModel]


def build_api_tool(config: ApiToolConfig) -> Tool:
    """Builds a Tool whose function validates its input against `input_schema`, calls
    `config.client`, and validates the JSON response against `output_schema`."""

    async def call(params: Any) -> Any:
        response = await config.client.request(config.method, config.path, params=params.model_dump(exclude_none=True))
        return config.output_schema.model_validate(response.json())

    # The real per-config types can't be written statically (they're only known at runtime), so
    # `call` is declared `Any` above for mypy and given its real signature here — PydanticAI's own
    # tool introspection reads __annotations__, so this is what actually builds the tool's schema.
    call.__annotations__["params"] = config.input_schema
    call.__annotations__["return"] = config.output_schema
    call.__name__ = config.name
    call.__doc__ = config.description
    return Tool(call, name=config.name, description=config.description)
