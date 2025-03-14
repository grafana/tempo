from abc import abstractmethod
from typing import Protocol, Dict, Any, Callable
from sklearn.base import BaseEstimator


class SearchCV(Protocol):
    """
    A protocol for search cross-validation.
    """
    cv_results_: Dict[str, Any]
    best_estimator_: BaseEstimator
    best_score_: float
    best_params: Dict[str, Any]
    best_index_: int
    scorer: Callable[[Any], float]
    multi_metric: bool
    n_splits_: int
    refit_time_: float
    n_features_in_: int

    @abstractmethod
    def score(self, X, y):
        pass

    @abstractmethod
    def score_Samples(self, X):
        pass

    @abstractmethod
    def predict(self, X):
        pass

    @abstractmethod
    def predict_proba(self, X):
        pass

    @abstractmethod
    def predict_log_proba(self, X):
        pass

    @abstractmethod
    def decision_function(self, X):
        pass

    @abstractmethod
    def transform(self, X):
        pass

    @abstractmethod
    def inverse_transform(self, X):
        pass

    @abstractmethod
    def fit(self, X, y):
        pass

    @abstractmethod
    def set_params(self, **kwargs):
        pass

    @abstractmethod
    def get_params(self) -> Dict[str, Any]:
        pass
