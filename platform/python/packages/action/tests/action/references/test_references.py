def test_references_lists_each_type_name_with_where_it_is_written(case, built):
    assert built.references() == [(tuple(one["path"]), one["name"]) for one in case["expect"]]
