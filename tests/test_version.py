import re

from grafanalabs_ml_pipeline_development import __version__


def test_version_is_semver() -> None:
    assert re.match(r"^\d+\.\d+\.\d+$", __version__)
