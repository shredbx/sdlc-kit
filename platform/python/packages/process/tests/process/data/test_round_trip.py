import yaml
from process_kit.process import Run


def test_a_run_is_its_own_data_read_back(case):
    run = Run(**{key: value for key, value in case["run"].items() if key != "errors"})
    from process_kit.types import Error

    run.errors = [Error(tuple(one["path"]), one["code"], one["message"]) for one in case["run"].get("errors", [])]
    data = run.to_data()
    assert yaml.safe_load(yaml.safe_dump(data)) == data
    assert Run.from_data(data) == run
