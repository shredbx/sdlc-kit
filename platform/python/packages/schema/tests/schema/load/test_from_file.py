from process_kit.schema import Schema


def test_load_reads_the_schema_from_a_file(tmp_path, case):
    source = tmp_path / "definition.yaml"
    source.write_text(case["text"], encoding="utf-8")
    schema = Schema.load(source, case["namespace"])
    assert schema.name == case["expect"]
