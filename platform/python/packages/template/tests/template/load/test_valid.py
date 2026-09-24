from process_kit.template import Template


def test_load_gives_the_template_the_case_describes(case, tree):
    template = Template.load(tree / case["folder"], case.get("namespace", ""))
    assert isinstance(template, Template)
    expect = case["expect"]
    assert (template.id, template.description, template.input) == (expect["id"], expect.get("description"), expect["input"])
    assert [(path, list(lines)) for path, lines in template.pattern.items()] == [(one["path"], one["lines"]) for one in expect.get("pattern", [])]
    assert template.home == tree / case["folder"]
