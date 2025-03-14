import numpy as np
import pandas as pd
from sklearn.linear_model import Lasso
from sklearn.pipeline import Pipeline
from sklearn.preprocessing import StandardScaler

from mlt.models.base_models import MLTBaseRegression


class MLTLassoRegressionModel(MLTBaseRegression):
    def _new_pipeline(self) -> Pipeline:
        return Pipeline([
            ("standard_scaler", StandardScaler()),
            ("estimator", Lasso()),
        ])

    def feature_importances(self, training_data: pd.DataFrame) -> pd.DataFrame:
        estimator = self.get_booster_if_needed()
        return pd.DataFrame({"feature": self.pipeline.feature_names_in_, 
                             "score": np.abs(estimator.coef_)})

    def get_shap_values(self, feature_data, predictions) -> pd.DataFrame:
        feature_names = [col for col in feature_data.columns if col not in 
                         self.non_feature_columns]
        coefs = self.get_booster_if_needed().coef_
        transformed = self.pipeline.named_steps.standard_scaler.transform(
            feature_data[feature_names])
        shap_data = feature_data.copy()
        shap_data[feature_names] = coefs * transformed
        shap_data[self.predicted_label_name] = predictions.iloc[:, 0]
        return shap_data
