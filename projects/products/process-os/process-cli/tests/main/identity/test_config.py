def test_config_beside_the_program_is_found_exactly_when_the_case_says(case, result):
    code, out, err = result
    expect = case["expect"]
    assert code == expect["code"]
    if "ok_suffix" in expect:
        assert out and out[0].endswith(expect["ok_suffix"])
    if "stderr" in expect:
        assert err == expect["stderr"]
    if "stderr_has" in expect:
        assert any(expect["stderr_has"] in line for line in err)
