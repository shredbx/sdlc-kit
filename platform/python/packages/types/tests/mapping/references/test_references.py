from process_kit.types import MappingType


def test_references_lists_each_type_name_with_where_it_is_written(case):
    found = MappingType(**case["properties"]).references()
    assert found == [(tuple(one["path"]), one["name"]) for one in case["expect"]]
