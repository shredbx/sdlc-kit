import shutil

import pytest

pytestmark = pytest.mark.skipif(shutil.which("uv") is None, reason="the demo builds with uv")


def test_the_real_build_runs_the_same_way_from_libraries_as_from_one_folder(case, results):
    expects = [call["expect"] for call in case["runs"]]
    assert [(code, out, len(err)) for code, out, err in results] == [(one["code"], one["stdout"], len(one["starts"])) for one in expects]
    assert all(line.startswith(start) and len(line) > len(start) for (_, _, err), one in zip(results, expects) for line, start in zip(err, one["starts"]))
