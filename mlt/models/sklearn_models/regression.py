import numpy as np
import pandas as pd
import shap

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
        return pd.DataFrame({"feature": self.pipeline.feature_names_in_, "score": np.abs(estimator.coef_)})
    
    def get_shap_values(self, feature_data, predictions) -> pd.DataFrame:
        feature_names = [col for col in feature_data.columns if col not in self.non_feature_columns]

    
    
