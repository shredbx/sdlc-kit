from process_kit.schema import qualify_id


def test_an_id_is_the_name_the_case_expects(case):
    assert qualify_id(case["name"], case["namespace"]) == case["expect"]
