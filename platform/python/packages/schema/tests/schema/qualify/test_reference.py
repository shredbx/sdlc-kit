from process_kit.schema import qualify


def test_a_reference_to_a_type_is_the_name_the_case_expects(case):
    assert qualify(case["name"], case["namespace"]) == case["expect"]
