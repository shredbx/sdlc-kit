def test_conform_finds_nothing_wrong_with_the_files_in_the_case(case, template, types, source):
    assert template.conform(case["data"], source, types) == []
