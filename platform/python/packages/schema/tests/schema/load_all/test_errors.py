from process_kit.schema import Schema


def test_load_all_locates_every_error_by_the_file_it_came_from(tmp_path, case):
    for filename, text in case["files"].items():
        (tmp_path / filename).write_text(text, encoding="utf-8")
    errors = Schema.load_all(tmp_path, case["namespace"])
    assert [(error.path, error.code) for error in errors] == [(tuple(one["path"]), one["code"]) for one in case["expect"]]
    assert all(error.message for error in errors)
