def test_list_records_shows_every_path_or_only_those_under_a_prefix(case, result):
    assert result == (0, case["expect"]["stdout"], [])
