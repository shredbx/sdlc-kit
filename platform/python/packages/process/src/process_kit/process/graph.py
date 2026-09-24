"""The graph check: before anything runs, every input is there, every choice is real, and the output is produced."""

from collections.abc import Mapping

from process_kit.types import Error, Field, Location, MappingType, StringType, Types

from .process import Process
from .resolver import Resolver
from .steps import Call, Step, Stop, Switch


def check(process: Process, resolver: Resolver, types: Types, checked: set[str] | None = None) -> list[Error]:
    """Every error in the graph of `process`, located in the file of the process it is in, at the step, or `[]`. Nothing is read
    and nothing runs: what a node is comes from `resolver`, and what a type is from `types`. `checked`, shared between calls, holds
    the ids of the processes already checked, so that checking many processes reports each one's errors once."""
    graph = _Graph(resolver, types, set() if checked is None else checked)
    return [] if process.id in graph.checked else graph.process(process, (process.id,))


class _Graph:
    def __init__(self, resolver: Resolver, types: Types, checked: set[str]) -> None:
        self.resolver, self.types, self.checked = resolver, types, checked

    def process(self, process: Process, stack: tuple[str, ...]) -> list[Error]:
        self.checked.add(process.id)
        file = (str(process.home),)
        errors = [Error(file + ("requires", name), "duplicate", f"{name!r} is an input already") for name in process.requires if name in process.input]
        context, stopped = self._steps(process.steps, file + ("steps",), {**process.input, **process.requires}, set(), stack, errors)
        if not stopped:
            for name, kind in process.output.items():
                if name not in context:
                    errors.append(Error(file + ("output", name), "not_produced", f"no step gives {name!r}"))
                elif context[name] != kind:
                    errors.append(Error(file + ("output", name), "type_mismatch", f"{name!r} is given as {context[name]!r}, and declared as {kind!r}"))
        return errors

    def _steps(
        self, steps: tuple[Step, ...], path: Location, context: dict[str, str], used: set[str], stack: tuple[str, ...], errors: list[Error]
    ) -> tuple[dict[str, str], bool]:
        """Walk the steps in order. The context after them, and whether the flow ended in a stop."""
        stopped = False
        for index, step in enumerate(steps):
            here = (*path, index)
            if stopped:
                errors.append(Error(here, "unreachable", "no step can run after a stop"))
                break
            if isinstance(step, Call):
                self._call(step, here, context, used, stack, errors)
            elif isinstance(step, Switch):
                context, stopped = self._switch(step, here, context, used, stack, errors)
            else:
                stopped = isinstance(step, Stop)
        return context, stopped

    def _call(self, step: Call, here: Location, context: dict[str, str], used: set[str], stack: tuple[str, ...], errors: list[Error]) -> None:
        node = self.resolver.node(step.id)
        if isinstance(node, list):
            first = node[0]
            errors.append(Error(here, first.code, f"the node {step.id!r} cannot be used: {first.message}"))
            return
        if isinstance(node, Process) and node.id in stack:
            errors.append(Error(here, "cycle", f"{node.id!r} runs itself: {' -> '.join((*stack, node.id))}"))
            return
        if step.id in used:
            errors.append(Error(here, "duplicate", f"{step.id!r} runs twice on one path: a node is named by its id"))
        used.add(step.id)
        takes = {**node.input, **node.requires} if isinstance(node, Process) else node.input
        for name, kind in takes.items():
            if name not in context:
                errors.append(Error(here, "missing", f"{step.id!r} takes {name!r}, which no input or earlier step gives"))
            elif context[name] != kind:
                errors.append(Error(here, "type_mismatch", f"{step.id!r} takes {name!r} as {kind!r}, and it is {context[name]!r} here"))
        if isinstance(node, Process) and node.id not in self.checked:
            errors.extend(self.process(node, (*stack, node.id)))
        for name, kind in node.output.items():
            if name in context and context[name] != kind:
                errors.append(Error(here, "type_mismatch", f"{step.id!r} gives {name!r} as {kind!r}, and it is {context[name]!r} here"))
            context[name] = kind

    def _switch(
        self, step: Switch, here: Location, context: dict[str, str], used: set[str], stack: tuple[str, ...], errors: list[Error]
    ) -> tuple[dict[str, str], bool]:
        enum = self._enum(self._value_type(step.on, context, (*here, "switch"), errors))
        if enum is not None:
            errors += [Error((*here, "cases", key), "invalid", f"{key!r} is not one of {enum}") for key in step.cases if key not in enum]
            missing = [value for value in enum if value not in step.cases]
            if step.default is None and missing:
                errors.append(Error((*here, "cases"), "unhandled", f"no case for {missing}, and no default"))
        exhaustive = step.default is not None or (enum is not None and all(value in step.cases for value in enum))
        branches = [((*here, "cases", key), steps) for key, steps in step.cases.items()]
        if step.default is not None:
            branches.append(((*here, "default"), step.default))
        live, before = [], set(used)
        for path, steps in branches:
            branch_used = set(before)
            after, stopped = self._steps(steps, path, dict(context), branch_used, stack, errors)
            used |= branch_used
            if not stopped:
                live.append(after)
        if not exhaustive:
            live.append(dict(context))
        if not live:
            return context, True
        merged = dict(context)
        for name in live[0]:
            if name in context or not all(name in after for after in live):
                continue
            kinds = {after[name] for after in live}
            if len(kinds) > 1:
                errors.append(Error(here, "type_mismatch", f"the branches give {name!r} as {sorted(kinds)}"))
            else:
                merged[name] = kinds.pop()
        return merged, False

    def _value_type(self, on: str, context: Mapping[str, str], place: Location, errors: list[Error]) -> str | None:
        """The type of the value `on` names: a name in the context, then fields. None, and an error, when it cannot be found."""
        name, *fields = on.split(".")
        if name not in context:
            errors.append(Error(place, "missing", f"{on!r} starts with {name!r}, which is not in the context here"))
            return None
        kind = context[name]
        for part in fields:
            found = self._field(kind, part)
            if found is None:
                errors.append(Error(place, "unknown_field", f"{kind!r} has no field {part!r}"))
                return None
            kind = found.type
        return kind

    def _field(self, kind: str, name: str) -> Field | None:
        found = self.types.get(kind)
        base = getattr(found, "base", found)
        if isinstance(base, MappingType) and base.fields:
            return next((field for field in base.fields if field.name == name), None)
        return None

    def _enum(self, kind: str | None) -> list[str] | None:
        found = None if kind is None else self.types.get(kind)
        base = getattr(found, "base", found)
        return base.enum if isinstance(base, StringType) else None
