import numpy as np
import pandas as pd
import shap
from sklearn.feature_selection import SelectKBest
from sklearn.inspection import permutation_importance
from sklearn.linear_model import LogisticRegression
from sklearn.pipeline import Pipeline
from sklearn.preprocessing import StandardScaler
from sklearn.svm import SVC
from sklearn.tree import DecisionTreeClassifier

from mlt.models.base_models import MLTBaseClassification


class MLTSKLearnClassification(MLTBaseClassification):
    """
    MLT wrapper for SKLearn classification.
    """
    def _new_pipeline(self) -> Pipeline:
        return Pipeline(
            [
                ("feature_selection", SelectKBest().set_output(transform="pandas")),
                ("feature_scaler", StandardScaler().set_output(transform="pandas")),
                (
                    "estimator",
                    LogisticRegression(multi_class="multinomial"),
                ),
            ]
        )
    
    def feature_importances(self, training_data: pd.DataFrame) -> pd.DataFrame:
        estimator = self.get_booster_if_needed()
        names = self.pipeline.name_steps.feature_selection.get_feature_names_out(self.pipeline.feature_names_in_)
        return pd.DataFrame(
            {
                "feature": names,
                "score": np.abs(estimator.coef_).mean(axis=0),
            }
        )
    
    def get_shap_values(self, feature_data: pd.DataFrame, predictions: pd.DataFrame) -> pd.DataFrame:
        feature_names = [col for col in feature_data.columns if col not in self.non_feature_columns]
        transformed = self.pipeline.named_steps.standard_scaler.transform(feature_data[feature_names])
        transformed = self.pipeline.named_steps.feature_selection.transform(transformed)
        metadata_columns = [col for col in feature_data.columns if col in self.non_feature_columns]

        logistic_regressor = self.get_booster_if_needed()
        shap_values = [
            (logistic_regressor.coef_[int(i), :] * transformed).to_numpy() for i in logistic_regressor.classes_
        ]
        expected_values = logistic_regressor.predict_proba(transformed)
        return self._make_scaled_shap_df(
            transformed.columns, feature_data[metadata_columns], predictions, shap_values, expected_values
        )
   