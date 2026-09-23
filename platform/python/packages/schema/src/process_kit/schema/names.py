"""The name rule: how a name written in a definition becomes the name the whole kit sees."""

from collections.abc import Container

from process_kit.types import YAML_TYPES


def check_namespace(namespace: str) -> None:
    """A wrong call raises: a namespace is text, empty or words joined by single dots."""
    if not isinstance(namespace, str):
        raise TypeError(f"namespace must be text, got {namespace!r}")
    if namespace and "" in namespace.split("."):
        raise ValueError(f"namespace {namespace!r} must be words joined by single dots")


def qualify(name: str, namespace: str) -> str:
    """A reference to a type, as the whole kit sees it. A YAML type name, or a name with a dot, stays as written;
    any other bare name is the namespace's own. An empty namespace changes nothing."""
    return _qualified(name, namespace, YAML_TYPES)


def qualify_id(name: str, namespace: str) -> str:
    """An id, as the whole kit sees it. A name with a dot stays as written; every bare name is the namespace's own,
    a YAML type name too, since an id is never a YAML type."""
    return _qualified(name, namespace, ())


def _qualified(name: str, namespace: str, kept: Container[str]) -> str:
    if not isinstance(name, str):
        raise TypeError(f"a name must be text, got {name!r}")
    check_namespace(namespace)
    return name if not namespace or "." in name or name in kept else f"{namespace}.{name}"
