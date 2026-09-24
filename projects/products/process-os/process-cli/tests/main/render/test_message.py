def test_render_prints_the_exact_message_in_the_case_and_exits_1(case, result):
    assert result == (1, [], case["expect"]["stderr"])
