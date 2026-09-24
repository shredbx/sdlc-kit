"""The runner: walk a checked process one node at a time, save the run after each, and go on from where it stopped."""

from collections.abc import Callable, Mapping
from typing import Any

from process_kit.action import Action, Context, Executor, Outcome
from process_kit.types import Error, Types

from .process import Process
from .resolver import Resolver
from .run import Event, Run
from .steps import Call, Step, Stop, Switch
from .store import RunStore


class Runner:
    """Runs processes. It is handed everything it uses, and reads no file: what an id names (`resolver`), what a type is (`types`),
    how scripts run (`executor`), where a run is kept (`store`), what an action is given (`context`), and who is told (`on_event`)."""

    def __init__(
        self,
        resolver: Resolver,
        types: Types,
        executor: Executor,
        store: RunStore,
        context: Context,
        on_event: Callable[[Event], None] | None = None,
    ) -> None:
        self.resolver, self.types, self.executor, self.store, self.context = resolver, types, executor, store, context
        self.on_event = on_event or (lambda event: None)

    def start(self, process: Process, inputs: Mapping[str, Any], scopes: Mapping[str, str], digest: str) -> Run:
        """A new run of `process` on `inputs`, its input and requires by name, walked until it is done, stops or fails. The process is
        checked, and the inputs are the caller's to have checked. Nothing raises for wrong data."""
        run = Run(self.store.new_id(process.id), process.id, dict(scopes), digest, context=dict(inputs))
        return self._walk(process, run)

    def resume(self, run: Run, process: Process | None = None) -> Run:
        """Go on with `run` from its first node that is not done. A run that is done is returned as it is. The process is the given one,
        whose id must be the run's target, or else the resolver's. A target that is not a process, or a `process` that is not the
        target, is a wrong call and raises."""
        if process is None:
            node = self.resolver.node(run.target)
            if not isinstance(node, Process):
                raise ValueError(f"{run.target!r} is not a process: {node!r}")
            process = node
        elif not isinstance(process, Process):
            raise TypeError(f"process must be a Process or None, got {process!r}")
        elif process.id != run.target:
            raise ValueError(f"the run is of {run.target!r}, not of {process.id!r}")
        if run.status == "done":
            return run
        run.status, run.reason, run.errors = "running", None, []
        return self._walk(process, run)

    def _walk(self, process: Process, run: Run) -> Run:
        self.store.save(run)
        if self._steps(process.steps, "", run.context, run):
            errors = _outputs(process, run.context, self.types)
            if errors:
                self._end(run, "failed", "the outputs are wrong", errors)
            else:
                run.output = {name: run.context[name] for name in process.output}
                self._end(run, "done", None, [])
        return run

    def _steps(self, steps: tuple[Step, ...], prefix: str, context: dict[str, Any], run: Run) -> bool:
        """Walk the steps. False when the run has ended, so the walk ends too."""
        for step in steps:
            if isinstance(step, Call):
                if not self._call(step, prefix, context, run):
                    return False
            elif isinstance(step, Switch):
                if not self._switch(step, prefix, context, run):
                    return False
            else:
                assert isinstance(step, Stop)
                self._end(run, "stopped", step.reason, [])
                return False
        return True

    def _call(self, step: Call, prefix: str, context: dict[str, Any], run: Run) -> bool:
        key = f"{prefix}{step.id}"
        node = self.resolver.node(step.id)
        if isinstance(node, Process):
            return self._nested(node, key, context, run)
        assert isinstance(node, Action), node
        if key in run.done:
            return True
        self._tell(run, key, "started")
        result = node.run({name: context[name] for name in node.input if name in context}, self.context, self.types, self.executor)
        if result.stdout is not None:
            self.store.log(run.id, key, result.stdout, result.stderr or "")
        if result.outcome in (Outcome.done, Outcome.skipped):
            context.update(result.outputs)
            run.done.append(key)
            if result.outcome is Outcome.skipped:
                run.skipped.append(key)
            self.store.save(run)
            self._tell(run, key, result.outcome.value)
            return True
        self._end(run, result.outcome.value, result.reason, list(result.errors))
        self._tell(run, key, result.outcome.value, result.reason)
        return False

    def _nested(self, process: Process, key: str, context: dict[str, Any], run: Run) -> bool:
        if key in run.done:
            return True
        inner = run.nested.get(key)
        if inner is None:
            inner = {name: context[name] for name in {**process.input, **process.requires} if name in context}
            run.nested[key] = inner
        self._tell(run, key, "started")
        if not self._steps(process.steps, f"{key}/", inner, run):
            return False
        errors = _outputs(process, inner, self.types)
        if errors:
            self._end(run, "failed", "the outputs are wrong", errors)
            self._tell(run, key, "failed", "the outputs are wrong")
            return False
        context.update({name: inner[name] for name in process.output})
        del run.nested[key]
        inside = [node for node in run.done if node.startswith(f"{key}/")]
        idle = bool(inside) and all(node in run.skipped for node in inside)
        run.done.append(key)
        if idle:
            run.skipped.append(key)
        self.store.save(run)
        self._tell(run, key, "skipped" if idle else "done")
        return True

    def _switch(self, step: Switch, prefix: str, context: dict[str, Any], run: Run) -> bool:
        try:
            value = _value(context, step.on)
        except KeyError:
            self._end(run, "failed", f"{step.on!r} is not in the context", [])
            return False
        text = "true" if value is True else "false" if value is False else str(value)
        branch = step.cases.get(text, step.default)
        if branch is None:
            self._end(run, "failed", f"no case for {text}, and no default", [])
            return False
        return self._steps(branch, prefix, context, run)

    def _end(self, run: Run, status: str, reason: str | None, errors: list[Error]) -> None:
        run.status, run.reason, run.errors = status, reason, errors
        self.store.save(run)

    def _tell(self, run: Run, node: str, state: str, reason: str | None = None) -> None:
        self.on_event(Event(run.id, node, state, reason))


def _value(context: Mapping[str, Any], on: str) -> Any:
    """The value a switch reads: a name in the context, then keys."""
    name, *keys = on.split(".")
    value = context[name]
    for key in keys:
        value = value[key]
    return value


def _outputs(process: Process, context: Mapping[str, Any], types: Types) -> list[Error]:
    """What is wrong with the declared output in `context`: a name missing, or a value that is not of its type."""
    errors: list[Error] = []
    for name, kind in process.output.items():
        if name not in context:
            errors.append(Error((name,), "missing", f'"{name}" is required'))
        else:
            errors.extend(types.validate(context[name], kind, (name,)))
    return errors
