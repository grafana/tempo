from grafanalabs_ml_pipeline_development.core import hello


def test_hello() -> None:
    assert hello("world") == "hello world"
