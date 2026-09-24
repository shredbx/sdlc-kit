"""The template engine: Jinja2, made strict, and the one question the loader asks of it."""

from functools import cache

import jinja2


@cache
def environment() -> jinja2.Environment:
    """A name the data lacks is an error, and a file keeps the line feed it ends with."""
    return jinja2.Environment(undefined=jinja2.StrictUndefined, keep_trailing_newline=True)


def syntax_error(source: str) -> str | None:
    """None when `source` is a template that compiles, or one line saying what is wrong and the line it is on."""
    try:
        environment().from_string(source)
    except jinja2.TemplateSyntaxError as problem:
        return f"{problem.message} (line {problem.lineno})"
    return None
