def test_version_prints_one_line_and_exits_0_needing_no_config(case, result):
    assert result == (0, [case["expect"]], [])
