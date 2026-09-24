from process_kit.config import Config


def test_load_gives_the_resolved_paths_in_the_case(case, tmp_path):
    for folder in case.get("folders", []):
        (tmp_path / folder).mkdir(parents=True, exist_ok=True)
    for name, text in case["files"].items():
        (tmp_path / name).parent.mkdir(parents=True, exist_ok=True)
        (tmp_path / name).write_text(text, encoding="utf-8")
    file = tmp_path / case.get("file", "processos.yaml")
    config = Config.load(file)
    assert isinstance(config, Config)
    assert config.file == file
    found = {
        "definitions": config.definitions,
        "root": config.runtime.root,
        "records": config.runtime.records,
        "output": config.runtime.output,
        "runs": config.runtime.runs,
    }
    expect = dict(case["expect"])
    libraries = expect.pop("libraries", [])
    version = expect.pop("version", None)
    history = expect.pop("history", 10)
    assert {name: path.relative_to(tmp_path).as_posix() for name, path in found.items()} == expect
    assert config.version == version
    assert config.runtime.history == history
    shown = [[library.name, library.path.relative_to(tmp_path).as_posix(), library.access, library.version] for library in config.libraries]
    assert shown == libraries
