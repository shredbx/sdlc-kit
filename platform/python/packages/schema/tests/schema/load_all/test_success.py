from process_kit.schema import Schema


def test_load_all_reads_every_real_yaml_file_in_the_folder(tmp_path, case):
    for filename, text in case["files"].items():
        (tmp_path / filename).write_text(text, encoding="utf-8")
    result = Schema.load_all(tmp_path, case["namespace"])
    assert sorted(result.keys()) == sorted(case["expect"])
