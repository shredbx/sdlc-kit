from pathlib import Path

from process_kit.config import Config


def test_load_gives_exactly_the_errors_in_the_case_and_nothing_else(case, tmp_path):
    for folder in case.get("folders", []):
        (tmp_path / folder).mkdir(parents=True, exist_ok=True)
    for name, text in case["files"].items():
        (tmp_path / name).parent.mkdir(parents=True, exist_ok=True)
        (tmp_path / name).write_text(text, encoding="utf-8")
    errors = Config.load(tmp_path / "processos.yaml")
    assert isinstance(errors, list)
    shown = [((Path(error.path[0]).relative_to(tmp_path).as_posix(), *error.path[1:]), error.code) for error in errors]
    assert shown == [(tuple(one["path"]), one["code"]) for one in case["expect"]]
    assert all(error.message for error in errors)
