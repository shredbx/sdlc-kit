from process_kit.template import Template


def test_references_names_the_type_of_the_data_in_full(case, tree):
    template = Template.load(tree / "x", case.get("namespace", ""))
    assert template.references() == [(tuple(one["path"]), one["name"]) for one in case["expect"]]
