# ml-pipeline-development

## Description

This repo contains code for training machine learning models and tracking the results.

## Development Prerequisites

* [pyenv](https://github.com/pyenv/pyenv)
* [Poetry](https://python-poetry.org)

## Contributing

This package utilizes [Poetry](https://python-poetry.org) for dependency management
and [pre-commit](https://pre-commit.com/) for ensuring code formatting is automatically done and code style checks are
performed.

You'll also want to set up and use `pyenv` to manage Python versions. 

## Installation

After cloning this repo, in the project home directory, run:

```bash
pyenv install 3.9.10
pyenv local 3.9.10
python -m pip install --upgrade pip
python -m pip install poetry
python -m poetry config virtualenvs.in-project true
python -m poetry install
python -m poetry shell
pre-commit install
```

The required dependencies for this repo only include scikit-learn models. To install with xgboost, lightgbm, or pytorch
models, run:

```bash
poetry install xgboost 
```
