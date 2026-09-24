def test_render_gives_the_files_in_the_case_in_order(case, template, types):
    files = template.render(case["data"], types)
    assert isinstance(files, dict)
    assert list(files.items()) == [tuple(one) for one in case["expect"]]
