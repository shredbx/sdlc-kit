def test_list_long_shows_each_definitions_own_description_or_nothing_extra(case, result):
    assert result == (0, case["expect"]["stdout"], [])
