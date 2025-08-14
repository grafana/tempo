import logging
from ray.tune import randint, uniform, loguniform
from ray.tune.search.sample import Domain

import functools


def _rvs(space: Domain, size=1, random_state=None):
    return space.sample(size=size, random_state=random_state)


def add_rvs_method(space: Domain):

    if not hasattr(space, "rvs"):
        setattr(space, "rvs", functools.partial(_rvs, space=space))
    else:

        logging.warning("Ray tune's space already has an rvs method. Skipping.")
    return space


def mlt_randint_inclusive(lower: int, upper:int):

    return add_rvs_method(randint(lower, upper + 1))


def mlt_uniform(lower: float, upper: float):

    return add_rvs_method(uniform(lower, upper))


def mlt_loguniform(lower: float, upper: float):

    return add_rvs_method(loguniform(lower, upper))
