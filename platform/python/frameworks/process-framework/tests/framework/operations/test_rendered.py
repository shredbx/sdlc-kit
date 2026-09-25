def test_render_gives_the_paths_and_writes_the_files_in_the_case(case, tree, framework):
    paths, errors = framework.render("sdlc.pkg", tree / "data.yaml", case.get("into"))
    assert (paths, errors) == (case["expect"]["paths"], [])
    written = {path: (tree / path).read_text() if (tree / path).is_file() else None for path in case["expect"]["files"]}
    assert written == case["expect"]["files"]
