import logging
from typing import Callable, Dict, List, Any
import xgboost as xgb
import numpy as np
import pandas as pd
from sklearn.base import BaseEstimator
from sklearn.feature_selection import SelectKBest
from sklearn.pipeline import Pipeline
from sklearn.utils import class_weight

from mlt.models.base_models import MLTBaseClassification
from mlt.typing import CVSplitter, SearchCV


class SampleWeightClassifier(xgb.XGBClassifier):
    """
    Wrapper for XGBClassifier that allows for sample weighting.
    """
    def __init__(self, **kwargs):
        super().__init__(**kwargs)

    def get_sample_weight(self, y: np.ndarray) -> np.ndarray:
        """
        Get sample weight for XGBClassifier.
        """
        unique_labels = np.unique(y)
        class_weights = class_weight.compute_sample_weight("balanced", classes=unique_labels, y=y)
        label_to_class_weight = {unique_labels[i]: class_weights[i] for i in range(len(unique_labels))}
        sample_weights = [label_to_class_weight[y[i]] for i in range(len(y))]

        return sample_weights
    
    def get_feature_importances(self, importance_type) -> Dict[str, float]:

        return self.get_booster().get_score(importance_type=importance_type)
    
    def fit(self, X, y, use_balanced_sample_weighting: bool = True, **kwargs):
        if use_balanced_sample_weighting:
            sample_weights = self.get_sample_weights(y)
        else:
            sample_weights = None
        
        super().fit(X, y, sample_weight=sample_weights)


class MLTXGBoostClassification(MLTBaseClassification):
    """
    MLT wrapper for XGBoost classification.
    """
    def __init__(
            self, 
            search_cv_factory: Callable[[BaseEstimator], SearchCV],
            cv_splitter: CVSplitter,
            non_feature_columns: List[str],
            model_type: str,
            model_name: str,
            model_parameter_grid: Dict[str, List[Any]],
            use_gpu: bool,
            model_objective: str,
            prediction_time_column_name: str,
            true_label_name: str,
            predicted_label_name: str,
            probability_column_names: List[str],
            class_labels: List[Any],
            sampling_method: str = "None",
            importance_type: str = "",
            has_feature_importance: bool = True,
            has_shap: bool = True,
            extra_pipeline_kwargs: Dict = None,
            use_balanced_sample_weighting: bool = False,
    ):
        self.sampling_method = sampling_method
        if self.sampling_method is not None and not use_gpu:
            logging.warning("Sampling method is not supported for CPU models. Setting sampling method to None.")

        super().__init__(
            search_cv_factory=search_cv_factory,
            cv_splitter=cv_splitter,
            non_feature_columns=non_feature_columns,
            model_type=model_type,
            model_name=model_name,
            model_parameter_grid=model_parameter_grid,
            use_gpu=use_gpu,
            model_objective=model_objective,
            prediction_time_column_name=prediction_time_column_name,
            true_label_name=true_label_name,
            predicted_label_name=predicted_label_name,
            probability_column_names=probability_column_names,
            class_labels=class_labels,
            importance_type=importance_type,
            has_feature_importance=has_feature_importance,
            has_shap=has_shap,
            extra_pipeline_kwargs=extra_pipeline_kwargs,
            use_balanced_sample_weighting=use_balanced_sample_weighting,
        )

    def _new_pipeline(self):
        tree_method = "gpu_hist" if self.use_gpu else "hist"
        sampling_method = self.sampling_method if self.use_gpu else None
        return Pipeline(
            [
                ("feature_selection", SelectKBest().set_output(transform="pandas")),
                (
                    "estimator",
                    SampleWeightClassifier(
                        tree_method=tree_method,
                        objective=self.model_objective,
                        eval_metric="auc",
                        sampling_method=sampling_method,
                        
                    ),
                ),
            ]
        )
    
    def feature_importances(self, training_data: pd.DataFrame) -> pd.DataFrame:
        raw_importances = self.pipeline["estimator"].get_feature_importances(self.importance_type)
        importances = pd.DataFrame(list(raw.importances.items()), columns=["feature", "score"])

        return importances
    
    def get_shap_values(self, feature_data: pd.DataFrame, predictions: pd.DataFrame) -> pd.DataFrame:
        return super().get_shap_values(feature_data, predictions)