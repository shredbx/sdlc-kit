"""Serves this workspace's tools and processes over MCP, on standard input and output. Owns them for
as long as it runs; nothing else may write there while it does."""

import importlib.resources
from typing import Any

import anyio
from mcp import MCPError, types
from mcp.server.lowlevel import Server
from mcp.server.stdio import stdio_server
from process_framework import KINDS, ProcessFramework, Run
from process_kit.types import Error

#: process-cli's own commands, not one workspace's actions or processes — task 08. Grouped the same
#: way process-os.command's own records are; each group's own detail ships as package data,
#: rendered by process-os.skill-reference into tools/<group>.md (products/process-cli's own build).
HELP_GROUPS = ("schema", "template", "process", "runtime")

#: What process-os.show/.remove take as a kind — the four catalog kinds write also takes, plus
#: record (a path under runtime.records, not a catalog id — task 09, M6).
SHOW_KINDS = ("type", "schema", "process", "action", "record")

#: What process-os.write takes — record is not here: create/edit own that address space, not this
#: milestone's tools (task 09, M6).
WRITE_KINDS = ("type", "schema", "process", "action")

#: What process-os.list's own kind takes: every Catalog kind, plus records (the other address space).
LIST_KINDS = (*KINDS, "records")

#: process-os.list's own limit when the caller gives none — never unbounded by default for a tool
#: this new (decision 105, task 09): unlike the CLI's own list, which keeps its existing, already-
#: shipped unbounded default rather than changing behaviour nobody asked to change.
LIST_DEFAULT_LIMIT = 100

#: Reaches every MCP client, not only Claude Code's own skill (which gets the same governance from
#: SKILL.md's own inline `process-cli check`, model-invoked, only when it loads) — research.md §7's
#: own finding: InitializeResult.instructions was unset, so no client got this at the protocol level.
INSTRUCTIONS = (
    "process-cli check proves every definition in this workspace is still sound: run it, or read "
    "process-os.help(group: \"runtime\"), before the first tools/call of a session and again after "
    "editing a definition. process-os.help(group) also documents process-cli's own schema, template "
    "and process commands (validate, create, edit, render, conform, run, resume, runs, show, write, "
    "remove) — not the actions and processes tools/list already describes. process-os.show/write/"
    "remove read, create-or-replace and delete a definition or a record by id or path, and "
    "process-os.list finds one, capped at 100 unless you say otherwise — use these instead of "
    "shelling out to process-cli itself for any of that."
)


def serve(framework: ProcessFramework) -> None:
    """Run the MCP server for `framework` until the client disconnects. Blocks."""
    anyio.run(_serve, framework)


async def _serve(framework: ProcessFramework) -> None:
    async def on_list_tools(ctx: Any, params: Any) -> types.ListToolsResult:
        own = [_help_tool(), _show_tool(), _write_tool(), _remove_tool(), _list_tool()]
        return types.ListToolsResult(tools=[types.Tool(**tool) for tool in [*own, *framework.tools()]])

    async def on_call_tool(ctx: Any, params: types.CallToolRequestParams) -> types.CallToolResult:
        arguments = params.arguments or {}
        if params.name == "process-os.help":
            return _help(arguments.get("group"))
        if params.name == "process-os.show":
            return _show(framework, arguments)
        if params.name == "process-os.write":
            return _write(framework, arguments)
        if params.name == "process-os.remove":
            return _remove(framework, arguments)
        if params.name == "process-os.list":
            return _list(framework, arguments)
        return _result(framework.call(params.name, arguments))

    server = Server("process-os", instructions=INSTRUCTIONS, on_list_tools=on_list_tools, on_call_tool=on_call_tool)
    async with stdio_server() as (read_stream, write_stream):
        await server.run(read_stream, write_stream, server.create_initialization_options())


def _help_tool() -> dict[str, Any]:
    """`process-os.help`'s own MCP tool descriptor — always listed, alongside whatever the workspace
    itself defines: process-cli's own command surface is the same regardless of which workspace this
    server is pointed at, so it is not one of `framework.tools()`'s own, workspace-defined tools."""
    return {
        "name": "process-os.help",
        "description": (
            "process-cli's own command surface, by capability group — schema (validate, create, edit), "
            "template (render, conform), process (run, resume, runs) or runtime (init, check, list, "
            "show, write, remove, mcp). The same detail a Claude Code session reads from the plugin's "
            "own bundled file; here for any MCP client, live."
        ),
        "inputSchema": {
            "type": "object",
            "properties": {"group": {"type": "string", "enum": list(HELP_GROUPS)}},
            "required": ["group"],
            "additionalProperties": False,
        },
    }


def _help(group: str | None) -> types.CallToolResult:
    """`process-os.help`'s own `tools/call`: the text of `tools/<group>.md`, package data shipped
    with `process-cli` itself (`process-os.skill-reference`'s own second render target, task 08 M4)
    — never read from a workspace, since this is the program's own documentation, not the
    workspace's. A `group` that is missing or not one of `HELP_GROUPS` is a wrong call, the same
    `INVALID_PARAMS` a call that never starts already gives."""
    if group not in HELP_GROUPS:
        raise MCPError(types.INVALID_PARAMS, f"{group!r} is not a group: {', '.join(HELP_GROUPS)}")
    text = importlib.resources.files("process_cli").joinpath("tools", f"{group}.md").read_text(encoding="utf-8")
    return types.CallToolResult(content=[types.TextContent(type="text", text=text)], isError=False)


def _show_tool() -> dict[str, Any]:
    """`process-os.show`'s own descriptor."""
    return {
        "name": "process-os.show",
        "description": (
            "The raw text of a definition (type, schema, process or action, by id), or a record (by its own "
            "path under runtime.records, not a catalog id). An action comes back as {relative path: text}."
        ),
        "inputSchema": {
            "type": "object",
            "properties": {"kind": {"type": "string", "enum": list(SHOW_KINDS)}, "id": {"type": "string"}},
            "required": ["kind", "id"],
            "additionalProperties": False,
        },
    }


def _write_tool() -> dict[str, Any]:
    """`process-os.write`'s own descriptor."""
    return {
        "name": "process-os.write",
        "description": (
            "Create or replace a definition (type, schema, process or action, by id), checked before it is "
            "kept. content is text for type/schema/process; for action, {relative path: text} — action.yaml, "
            "its named scripts (check.sh, pre.sh, action.sh, post.sh), and anything under assets/."
        ),
        "inputSchema": {
            "type": "object",
            "properties": {
                "kind": {"type": "string", "enum": list(WRITE_KINDS)},
                "id": {"type": "string"},
                "content": {"oneOf": [{"type": "string"}, {"type": "object", "additionalProperties": {"type": "string"}}]},
            },
            "required": ["kind", "id", "content"],
            "additionalProperties": False,
        },
    }


def _remove_tool() -> dict[str, Any]:
    """`process-os.remove`'s own descriptor."""
    return {
        "name": "process-os.remove",
        "description": "Delete a definition (type, schema, process or action, by id) or a record (by its own path under runtime.records).",
        "inputSchema": {
            "type": "object",
            "properties": {"kind": {"type": "string", "enum": list(SHOW_KINDS)}, "id": {"type": "string"}},
            "required": ["kind", "id"],
            "additionalProperties": False,
        },
    }


def _list_tool() -> dict[str, Any]:
    """`process-os.list`'s own descriptor."""
    return {
        "name": "process-os.list",
        "description": (
            "The ids of one definition kind, or of every kind, or, for kind records, the paths of the "
            "records. long also shows each definition's own one-line description. scope/namespace/prefix "
            f"filter before anything past the bare id or path is loaded. limit defaults to {LIST_DEFAULT_LIMIT} "
            "when not given — never unbounded by default."
        ),
        "inputSchema": {
            "type": "object",
            "properties": {
                "kind": {"type": "string", "enum": list(LIST_KINDS)},
                "prefix": {"type": "string"},
                "scope": {"type": "string"},
                "namespace": {"type": "string"},
                "long": {"type": "boolean"},
                "limit": {"type": "integer", "minimum": 1},
            },
            "additionalProperties": False,
        },
    }


def _show(framework: ProcessFramework, arguments: dict[str, Any]) -> types.CallToolResult:
    """`process-os.show`'s own `tools/call`: `framework.show`, wrapped — its own errors, or an unsupported
    `kind`, are `INVALID_PARAMS`, the same as every other tool that didn't start."""
    kind, id = arguments.get("kind"), arguments.get("id")
    if kind not in SHOW_KINDS or not isinstance(id, str):
        raise MCPError(types.INVALID_PARAMS, f"kind must be one of {', '.join(SHOW_KINDS)}, and id text")
    found = framework.show(kind, id)
    if isinstance(found, list):
        raise MCPError(types.INVALID_PARAMS, "; ".join(error.message for error in found))
    if isinstance(found, dict):
        text = "\n\n".join(f"# {path}\n{content}" for path, content in found.items())
        return types.CallToolResult(content=[types.TextContent(type="text", text=text)], structuredContent={"files": found}, isError=False)
    return types.CallToolResult(content=[types.TextContent(type="text", text=found)], structuredContent={"text": found}, isError=False)


def _write(framework: ProcessFramework, arguments: dict[str, Any]) -> types.CallToolResult:
    """`process-os.write`'s own `tools/call`: `framework.write`, wrapped."""
    kind, id, content = arguments.get("kind"), arguments.get("id"), arguments.get("content")
    if kind not in WRITE_KINDS or not isinstance(id, str) or not isinstance(content, (str, dict)):
        raise MCPError(types.INVALID_PARAMS, f"kind must be one of {', '.join(WRITE_KINDS)}, id text, and content text or an object")
    errors = framework.write(kind, id, content)
    if errors:
        raise MCPError(types.INVALID_PARAMS, "; ".join(error.message for error in errors))
    return types.CallToolResult(content=[types.TextContent(type="text", text=f"{kind} {id}: written")], isError=False)


def _remove(framework: ProcessFramework, arguments: dict[str, Any]) -> types.CallToolResult:
    """`process-os.remove`'s own `tools/call`: `framework.remove`, wrapped."""
    kind, id = arguments.get("kind"), arguments.get("id")
    if kind not in SHOW_KINDS or not isinstance(id, str):
        raise MCPError(types.INVALID_PARAMS, f"kind must be one of {', '.join(SHOW_KINDS)}, and id text")
    errors = framework.remove(kind, id)
    if errors:
        raise MCPError(types.INVALID_PARAMS, "; ".join(error.message for error in errors))
    return types.CallToolResult(content=[types.TextContent(type="text", text=f"{kind} {id}: removed")], isError=False)


def _list(framework: ProcessFramework, arguments: dict[str, Any]) -> types.CallToolResult:
    """`process-os.list`'s own `tools/call`: `framework.list`/`.list_records`/`.description`, wrapped —
    `limit` defaults to `LIST_DEFAULT_LIMIT` here, the one behaviour genuinely new to this tool and not
    merely mirrored from the CLI (decision 105, task 09)."""
    kind = arguments.get("kind")
    if kind is not None and kind not in LIST_KINDS:
        raise MCPError(types.INVALID_PARAMS, f"kind must be one of {', '.join(LIST_KINDS)}")
    limit = arguments.get("limit", LIST_DEFAULT_LIMIT)
    if kind == "records":
        paths = framework.list_records(arguments.get("prefix"), limit)
        return types.CallToolResult(content=[types.TextContent(type="text", text="\n".join(paths))], structuredContent={"records": paths}, isError=False)
    kinds = [kind] if kind else list(KINDS)
    scope, namespace, long_ = arguments.get("scope"), arguments.get("namespace"), bool(arguments.get("long"))
    entries = [(one, entry) for one in kinds for entry in framework.list(one, scope, namespace, limit)]
    if long_:
        described = [(one, entry, framework.description(one, entry.id)) for one, entry in entries]
        lines = [f"{entry.id if kind else f'{one} {entry.id}'}  {description or ''}".rstrip() for one, entry, description in described]
        data = {"entries": [{"kind": one, "id": entry.id, "description": description} for one, entry, description in described]}
    else:
        lines = [entry.id if kind else f"{one} {entry.id}" for one, entry in entries]
        data = {"entries": [{"kind": one, "id": entry.id} for one, entry in entries]}
    return types.CallToolResult(content=[types.TextContent(type="text", text="\n".join(lines))], structuredContent=data, isError=False)


def _result(made: Run | list[Error]) -> types.CallToolResult:
    """One `tools/call`'s answer. `made` is `ProcessFramework.call`'s own: a didn't-start call — a list of
    `Error` — raises `MCPError` (`INVALID_PARAMS`), a real protocol error, decision 64's one case that is.
    Anything else — a `Run`, whatever its status — is a normal result, never `isError`: its `to_data()` is
    the `structuredContent`, the same shape `runs RUN` already prints, and the text is one line, the same
    shape `process-cli run` already prints on the command line."""
    if isinstance(made, list):
        raise MCPError(types.INVALID_PARAMS, "; ".join(error.message for error in made))
    reason = f" — {made.reason}" if made.reason else ""
    text = f"{made.target}: {made.status}{reason}"
    return types.CallToolResult(content=[types.TextContent(type="text", text=text)], structuredContent=made.to_data(), isError=False)
