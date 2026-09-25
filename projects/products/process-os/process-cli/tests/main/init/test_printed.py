def test_init_prints_the_paths_it_wrote_and_exits_0(case, tree, result):
    code, out, err = result
    assert (code, out, err) == (0, case["expect"]["stdout"], [])
    assert all((tree / line).is_file() for line in out)
