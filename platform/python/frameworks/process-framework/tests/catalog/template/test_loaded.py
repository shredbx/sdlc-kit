from process_framework import Catalog
from process_kit.template import Template


def test_template_gives_the_template_with_the_id_and_the_type_of_its_data_in_full(case, tree):
    template = Catalog.open(tree / "defs").template(case["ident"])
    assert isinstance(template, Template)
    assert (template.id, template.input, list(template.pattern)) == (case["ident"], case["expect"]["input"], case["expect"]["keys"])
