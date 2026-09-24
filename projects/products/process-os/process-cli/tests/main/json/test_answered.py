import json


def test_each_call_answers_with_one_json_document_that_holds_what_the_case_says(case, answers, holds):
    expects = [call["expect"] for call in case["runs"]]
    assert [code for code, _, _ in answers] == [one["code"] for one in expects]
    assert [len(out.splitlines()) for _, out, _ in answers] == [1] * len(expects)
    docs = [json.loads(out) for _, out, _ in answers]
    assert all(holds(doc, one["doc"]) for doc, one in zip(docs, expects))
    assert all(error["message"] for doc in docs for error in doc["errors"])
    assert all(text in err for (_, _, err), one in zip(answers, expects) for text in one.get("stderr_has", []))
