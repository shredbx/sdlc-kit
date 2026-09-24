from process_kit.process import Process


def test_load_gives_the_process_the_case_describes(case, tree, shape):
    process = Process.load(tree / case.get("file", "x.yaml"), case.get("namespace", ""))
    assert isinstance(process, Process)
    expect = case["expect"]
    assert process.id == expect["id"]
    assert process.description == expect.get("description")
    assert [list(ports.items()) for ports in (process.input, process.requires, process.output)] == [
        list(expect.get(key, {}).items()) for key in ("input", "requires", "output")
    ]
    assert [shape(step) for step in process.steps] == expect["steps"]
    assert process.home == tree / case.get("file", "x.yaml")
