from datetime import datetime

from process_framework import Runs
from process_kit.filesystem import Folder


def test_new_id_is_the_time_and_the_name_and_a_number_when_it_is_taken(case, tree):
    runs = Runs(Folder(tree / "runs"), now=lambda: datetime(2026, 9, 21, 14, 30, 0))
    assert runs.new_id(case["target"]) == case["expect"]
