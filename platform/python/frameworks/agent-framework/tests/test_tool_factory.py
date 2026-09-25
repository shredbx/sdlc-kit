"""Proves build_api_tool works end to end against a mocked HTTP transport — no real network or
API key needed. This is what M3's search_properties is built on."""

import httpx
from agent_framework.api.base import BaseAPIClient
from agent_framework.api.tool_factory import ApiToolConfig, build_api_tool
from pydantic import BaseModel
from pydantic_ai import Agent
from pydantic_ai.messages import ModelMessage, ModelResponse, TextPart, ToolCallPart, ToolReturnPart
from pydantic_ai.models.function import AgentInfo, FunctionModel


class SearchInput(BaseModel):
    location: str


class SearchOutput(BaseModel):
    count: int
    results: list[str]


def _mock_transport() -> httpx.MockTransport:
    def handler(request: httpx.Request) -> httpx.Response:
        assert request.url.path == "/properties/search"
        assert request.url.params["location"] == "Koh Phangan"
        return httpx.Response(200, json={"count": 1, "results": ["Villa A"]})

    return httpx.MockTransport(handler)


async def test_build_api_tool_calls_endpoint_and_validates_response() -> None:
    client = BaseAPIClient(base_url="https://example.internal", transport=_mock_transport())
    tool = build_api_tool(
        ApiToolConfig(
            name="search_properties",
            description="Search available properties by location",
            client=client,
            method="GET",
            path="/properties/search",
            input_schema=SearchInput,
            output_schema=SearchOutput,
        )
    )

    calls: list[ModelMessage] = []

    def call_tool(messages: list[ModelMessage], info: AgentInfo) -> ModelResponse:
        calls.append(messages[-1])
        if len(calls) == 1:
            return ModelResponse(parts=[ToolCallPart(tool_name="search_properties", args={"location": "Koh Phangan"})])
        return ModelResponse(parts=[TextPart(content="found it")])

    agent = Agent(FunctionModel(call_tool), name="test_agent", tools=[tool])
    result = await agent.run("find properties in Koh Phangan")

    assert result.output == "found it"
    tool_returns = [part for message in result.all_messages() for part in message.parts if isinstance(part, ToolReturnPart)]
    assert tool_returns[0].content == SearchOutput(count=1, results=["Villa A"])

    await client.aclose()
