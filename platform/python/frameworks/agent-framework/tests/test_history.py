"""History trimming is pure list surgery - table-tested without a model - plus one real agent run proving that what
the framework saves afterwards is the trimmed history too."""

from agent_framework.core.history import size_of, starts_turn, trim_history
from pydantic_ai import Agent
from pydantic_ai.capabilities import ProcessHistory
from pydantic_ai.messages import (
    ModelMessage,
    ModelRequest,
    ModelResponse,
    SystemPromptPart,
    TextPart,
    ToolCallPart,
    ToolReturnPart,
    UserPromptPart,
)
from pydantic_ai.models.function import AgentInfo, FunctionModel


def user(text: str) -> ModelRequest:
    return ModelRequest(parts=[UserPromptPart(content=text)])


def reply(text: str) -> ModelResponse:
    return ModelResponse(parts=[TextPart(content=text)])


def turn(n: int, size: int = 10) -> list[ModelMessage]:
    """One plain turn: a user message and a reply of `size` characters each, tagged with n."""
    return [user(f"u{n}".ljust(size, ".")), reply(f"a{n}".ljust(size, "."))]


def tool_turn(n: int) -> list[ModelMessage]:
    """A turn with a tool call: user -> call -> result -> reply."""
    call = ToolCallPart(tool_name="search", args={"q": "beach"}, tool_call_id=f"call{n}")
    result = ToolReturnPart(tool_name="search", content="3 found", tool_call_id=f"call{n}")
    return [user(f"u{n}"), ModelResponse(parts=[call]), ModelRequest(parts=[result]), reply(f"a{n}")]


def history(*turns: list[ModelMessage]) -> list[ModelMessage]:
    return [message for t in turns for message in t]


def user_texts(messages: list[ModelMessage]) -> list[str]:
    return [part.content.rstrip(".") for m in messages if starts_turn(m) for part in m.parts if isinstance(part, UserPromptPart)]  # type: ignore[union-attr]


def test_a_history_inside_the_limits_is_returned_as_it_is() -> None:
    messages = history(turn(1), turn(2), turn(3))

    assert trim_history(max_turns=8, max_chars=24_000)(messages) == messages


def test_no_limits_means_no_trimming() -> None:
    messages = history(*[turn(n) for n in range(1, 30)])

    assert trim_history(None, None)(messages) == messages


def test_only_the_newest_turns_are_kept() -> None:
    messages = history(*[turn(n) for n in range(1, 7)])

    kept = trim_history(max_turns=3, max_chars=None)(messages)

    assert user_texts(kept) == ["u4", "u5", "u6"]


def test_a_cut_is_always_at_the_start_of_a_turn_and_a_tool_call_keeps_its_result() -> None:
    messages = history(tool_turn(1), tool_turn(2), tool_turn(3), tool_turn(4))

    for max_turns in (1, 2, 3):
        kept = trim_history(max_turns, None)(messages)
        assert starts_turn(kept[0])
        calls = {p.tool_call_id for m in kept for p in m.parts if isinstance(p, ToolCallPart)}
        results = {p.tool_call_id for m in kept for p in m.parts if isinstance(p, ToolReturnPart)}
        assert calls == results and len(calls) == max_turns


def test_the_text_size_limit_drops_the_oldest_turns_first() -> None:
    messages = history(*[turn(n, size=100) for n in range(1, 6)])  # 200 characters per turn

    kept = trim_history(max_turns=None, max_chars=650)(messages)  # room for three turns, not four

    assert user_texts(kept) == ["u3", "u4", "u5"]


def test_the_newest_turn_is_kept_even_when_it_alone_is_over_the_limits() -> None:
    messages = history(turn(1), turn(2, size=5_000))

    kept = trim_history(max_turns=1, max_chars=100)(messages)

    assert user_texts(kept) == ["u2"]
    assert trim_history(max_turns=8, max_chars=100)(messages) == messages[2:]


def test_a_turn_that_does_not_fit_ends_the_search_so_no_gap_is_left_inside_what_is_kept() -> None:
    messages = history(turn(1, size=10), turn(2, size=1_000), turn(3, size=10), turn(4, size=10))

    kept = trim_history(max_turns=None, max_chars=100)(messages)

    assert user_texts(kept) == ["u3", "u4"]  # the small first turn is not brought back past the big second one


def test_whatever_comes_before_the_first_turn_is_kept() -> None:
    system = ModelRequest(parts=[SystemPromptPart(content="be brief")])
    messages = [system, *history(turn(1), turn(2), turn(3))]

    kept = trim_history(max_turns=1, max_chars=None)(messages)

    assert kept[0] is system
    assert user_texts(kept) == ["u3"]


def test_no_user_prompt_yet_or_a_single_turn_changes_nothing() -> None:
    assert trim_history(1, 1)([]) == []
    only = history(tool_turn(1))
    assert trim_history(1, 1)(only) == only


def test_size_counts_what_was_written_tool_arguments_and_tool_results() -> None:
    assert size_of(user("hello")) == 5
    assert size_of(reply("four")) == 4
    call = ModelResponse(parts=[ToolCallPart(tool_name="t", args='{"q": "ab"}', tool_call_id="c")])
    assert size_of(call) == len('{"q": "ab"}')
    assert size_of(ModelRequest(parts=[ToolReturnPart(tool_name="t", content="three", tool_call_id="c")])) == 5


# ---- a real agent run -------------------------------------------------------------------------------------------


async def test_the_model_sees_the_trimmed_history_and_the_saved_history_is_trimmed_too() -> None:
    seen: list[list[ModelMessage]] = []

    def model_fn(messages: list[ModelMessage], info: AgentInfo) -> ModelResponse:
        seen.append(list(messages))
        return reply("ok")

    agent = Agent(FunctionModel(model_fn), capabilities=[ProcessHistory(trim_history(max_turns=3, max_chars=None))])
    old = history(*[turn(n) for n in range(1, 11)])  # ten finished turns from earlier in the session

    result = await agent.run("new question", message_history=old)

    assert user_texts(seen[0]) == ["u9", "u10", "new question"]  # two old turns + the new one
    assert user_texts(result.all_messages()) == ["u9", "u10", "new question"]  # and that is what gets saved


async def test_a_tool_call_in_the_current_run_is_never_cut_from_its_result() -> None:
    requests: list[list[ModelMessage]] = []

    def model_fn(messages: list[ModelMessage], info: AgentInfo) -> ModelResponse:
        requests.append(list(messages))
        if any(isinstance(p, ToolReturnPart) for m in messages for p in m.parts):
            return reply("done")
        return ModelResponse(parts=[ToolCallPart(tool_name="search", args={"q": "x"}, tool_call_id="now")])

    agent = Agent(FunctionModel(model_fn), capabilities=[ProcessHistory(trim_history(max_turns=1, max_chars=1))])

    @agent.tool_plain
    def search(q: str) -> str:
        return "found it"

    result = await agent.run("go", message_history=history(turn(1), turn(2)))

    assert len(requests) == 2  # the call, then the answer after the result
    after_tool = requests[1]
    assert user_texts(after_tool) == ["go"]  # old turns are gone ...
    assert any(isinstance(p, ToolCallPart) for m in after_tool for p in m.parts)  # ... the run's own call and result stay
    assert any(isinstance(p, ToolReturnPart) for m in after_tool for p in m.parts)
    assert result.output == "done"
