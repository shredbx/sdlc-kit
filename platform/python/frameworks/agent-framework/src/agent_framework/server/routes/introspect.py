"""GET /agents and GET /agents/{name} - introspection for a dev console: which agents exist, and
for one agent, its fully-resolved system prompt (info.instructions - the same string a real model
receives), the individual fields it was composed from when the consumer provided them
(personality, prompt_fields - see core/types.py's RegisteredAgent), and its real tool schemas
(the same info.function_tools tools:list already reads, plus each tool's `kind` and whether it has
a debug card mapping, from tool_registry - see core/tool_registry.py). The prompt/tools split
comes from ONE FunctionModel-capture run per agent - the same proven mechanism tools:list already
runs in production, just also grabbing `instructions` this time. Mounted only when the server's
own debug_mode is on (see server/app.py) - absent in prod, not merely unauthenticated (a GET with
no user input has nothing to misinterpret, so no per-request double-gate is needed here)."""

from fastapi import APIRouter, HTTPException
from pydantic import BaseModel
from pydantic_ai import ModelRequest, ModelResponse, TextPart
from pydantic_ai.models.function import AgentInfo, FunctionModel
from pydantic_ai.tools import ToolDefinition

from agent_framework.core.store.base import Session
from agent_framework.core.tool_registry import ToolEntry
from agent_framework.core.types import RegisteredAgent


class ToolSchema(BaseModel):
    name: str
    description: str | None
    parameters_json_schema: dict
    kind: str
    has_debug_card_mapping: bool
    output_json_schema: dict | None
    usage_guidance: str | None


class PersonalityTraitOut(BaseModel):
    name: str
    description: str


class ConversionTechniqueOut(BaseModel):
    name: str
    when_to_use: str
    examples: list[str]


class PhrasesOut(BaseModel):
    use: list[str]
    avoid: list[str]


class RulesOut(BaseModel):
    do: list[str]
    dont: list[str]


class HesitationReplyOut(BaseModel):
    customer_says: str
    reply: str


class PersonalityOut(BaseModel):
    role: str
    scope: str
    traits: list[PersonalityTraitOut]
    techniques: list[ConversionTechniqueOut]
    phrases: PhrasesOut
    rules: RulesOut
    hesitation_replies: list[HesitationReplyOut]
    max_techniques_per_message: int


class AgentDetail(BaseModel):
    name: str
    channel: str
    output_protocol: str
    instructions: str | None
    personality: PersonalityOut | None
    prompt_fields: dict[str, str] | None
    tools: list[ToolSchema]


async def _capture_tools_and_instructions(registered: RegisteredAgent) -> tuple[list[ToolDefinition], str | None]:
    captured_tools: list[ToolDefinition] = []
    captured_instructions: str | None = None

    def _capture(messages: list[ModelRequest | ModelResponse], info: AgentInfo) -> ModelResponse:
        nonlocal captured_instructions
        captured_tools.extend(info.function_tools)
        captured_instructions = info.instructions
        return ModelResponse(parts=[TextPart(content="(schema introspection - not a real reply)")])

    deps = registered.build_deps(Session(id="__introspect__"))
    with registered.agent.override(model=FunctionModel(_capture)):
        await registered.agent.run("", deps=deps)

    return captured_tools, captured_instructions


def _to_tool_schema(tool: ToolDefinition, registry: dict[str, ToolEntry]) -> ToolSchema:
    entry = registry.get(tool.name)
    return ToolSchema(
        name=tool.name,
        description=tool.description,
        parameters_json_schema=tool.parameters_json_schema,
        kind=entry.kind if entry else "function",
        has_debug_card_mapping=bool(entry and entry.to_reply is not None),
        output_json_schema=entry.output_schema.model_json_schema() if entry and entry.output_schema else None,
        usage_guidance=entry.usage_guidance if entry else None,
    )


def _to_personality_out(personality: RegisteredAgent) -> PersonalityOut | None:
    p = personality.personality
    if p is None:
        return None
    return PersonalityOut(
        role=p.role,
        scope=p.scope,
        traits=[PersonalityTraitOut(name=t.name, description=t.description) for t in p.traits],
        techniques=[ConversionTechniqueOut(name=t.name, when_to_use=t.when_to_use, examples=list(t.examples)) for t in p.techniques],
        phrases=PhrasesOut(use=list(p.phrases.use), avoid=list(p.phrases.avoid)),
        rules=RulesOut(do=list(p.rules.do), dont=list(p.rules.dont)),
        hesitation_replies=[HesitationReplyOut(customer_says=h.customer_says, reply=h.reply) for h in p.hesitation_replies],
        max_techniques_per_message=p.max_techniques_per_message,
    )


def build_introspection_router(agents: dict[str, RegisteredAgent], tool_registry: dict[str, ToolEntry] | None = None) -> APIRouter:
    router = APIRouter()
    registry = tool_registry or {}

    @router.get("/agents", response_model=list[str])
    async def list_agents() -> list[str]:
        return sorted(agents.keys())

    @router.get("/agents/{name}", response_model=AgentDetail)
    async def get_agent(name: str) -> AgentDetail:
        registered = agents.get(name)
        if registered is None:
            raise HTTPException(status_code=404, detail=f"no agent named {name!r}")

        tools, instructions = await _capture_tools_and_instructions(registered)

        return AgentDetail(
            name=registered.name,
            channel=registered.channel,
            output_protocol=registered.output_protocol,
            instructions=instructions,
            personality=_to_personality_out(registered),
            prompt_fields=registered.prompt_fields,
            tools=[_to_tool_schema(tool, registry) for tool in sorted(tools, key=lambda t: t.name)],
        )

    return router
