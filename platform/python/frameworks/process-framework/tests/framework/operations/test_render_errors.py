def test_render_gives_exactly_the_errors_in_the_case_and_writes_nothing(case, tree, framework, located):
    paths, errors = framework.render(case.get("template", "sdlc.pkg"), tree / "data.yaml", case.get("into"))
    assert paths == []
    assert located(errors) == [(tuple(one["path"]), one["code"]) for one in case["expect"]]
    assert all(error.message for error in errors)
    assert not (tree / "output").exists()
