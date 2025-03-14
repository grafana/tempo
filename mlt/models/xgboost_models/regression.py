import pandas as pd
import xgboost as xgb
from sklearn.feature_selection import SelectKBest, f_regression
from sklearn.pipeline import Pipeline

from mlt.models.base_models import MLTBaseRegression


class MLTXGBoostRegression(MLTBaseRegression):
    """
    XGBoost regression model implementation for MLT.

    This class extends MLTBaseRegression to provide XGBoost-specific regression functionality.
    It implements a pipeline with feature selection using SelectKBest and XGBoost regression,
    with optional GPU acceleration.

    The model supports:
    - Feature importance calculation using XGBoost's built-in scoring
    - GPU acceleration when available and enabled
    - Feature selection using f_regression scoring
    - Standard regression metrics like MAE, MSE, R2

    Parameters are inherited from MLTBaseRegression, including:
        search_cv_factory: Factory function for cross-validation search
        cv_splitter: Cross validation splitting strategy
        non_feature_columns: Columns to exclude from feature set
        model_parameter_grid: Grid of parameters for hyperparameter tuning
        use_gpu: Whether to use GPU acceleration
        prediction_time_column_name: Column name for prediction timestamps
        true_label_name: Column name for true labels
        predicted_label_name: Column name for predicted labels
    """

    def _new_pipeline(self) -> Pipeline:
        tree_method = "gpu_hist" if self.use_gpu else "auto"
        return Pipeline([
            ("feature_selection", SelectKBest(score_func=f_regression).set_output(transform="pandas")),
            ("estimator", xgb.sklearn.XGBRegressor(
                tree_method=tree_method
            ))
        ])

    def feature_importances(self, training_data: pd.DataFrame) -> pd.DataFrame:
        """
        Calculate feature importance for the model.
        """
        raw_importance = self.get_booster_if_needed().get_score(importance_type=self.importance_type)
        importance = pd.DataFrame(list(raw_importance.items()), columns=["feature", "score"])

        return importance

    def get_shap_values(self, feature_data: pd.DataFrame, predictions: pd.Series) -> pd.DataFrame:
        """
        Get a dataframe of shap values for features and predictions.

        Parameters
        ----------
        feature_data : pd.DataFrame
            Raw feature data before any selection or transforms, and with metadata
        predictions : pd.Series
            Outputs from self.get_prediction_data

        Returns
        -------
        pd.DataFrame
            A dataframe of shap values for each prediction and feature
        """
        # Call the parent class implementation from MLTBaseRegression
        return super().get_shap_values(feature_data, predictions)