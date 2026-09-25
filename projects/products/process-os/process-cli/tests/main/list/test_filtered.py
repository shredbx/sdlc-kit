def test_list_filters_by_scope_namespace_and_limit_before_anything_else_is_loaded(case, result):
    assert result == (0, case["expect"]["stdout"], [])
