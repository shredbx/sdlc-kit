def test_each_call_stops_where_the_case_says_with_the_lines_and_the_code_it_expects(case, results):
    expects = [call["expect"] for call in case["runs"]]
    assert [(code, out, len(err)) for code, out, err in results] == [(one["code"], one["stdout"], len(one["starts"])) for one in expects]
    assert all(line.startswith(start) and len(line) > len(start) for (_, _, err), one in zip(results, expects) for line, start in zip(err, one["starts"]))
