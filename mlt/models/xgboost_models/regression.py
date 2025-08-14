import pandas as pd
import xgboost as xgb
from sklearn.feature_selection import SelectKBest, f_regression
from sklearn.pipeline import Pipeline
from sklearn.preprocessing import StandardScaler 
from sklearn.preprocessing import PowerTransformer
from sklearn.compose import TransformedTargetRegressor
import numpy as np

from mlt.models.base_models import MLTBaseRegression


class MLTXGBoostRegression(MLTBaseRegression):
    """
    XGBoost regression model implementation for MLT with target transformation support.
    
    This class extends MLTBaseRegression to provide XGBoost-specific regression 
    functionality with optional target transformation using scikit-learn's 
    TransformedTargetRegressor.
    
    The model supports:
    - Feature importance calculation using XGBoost's built-in scoring
    - GPU acceleration when available and enabled
    - Feature selection using f_regression scoring
    - Target transformation (yeo-johnson, box-cox, log1p, none)
    - Standard regression metrics like MAE, MSE, R2
    """
    
    def __init__(
        self,
        search_cv_factory,
        cv_splitter,
        non_feature_columns,
        model_type,
        model_name,
        model_parameter_grid,
        use_gpu,
        prediction_time_column_name,
        true_label_name,
        predicted_label_name,
        importance_type="",
        has_feature_importance=True,
        has_shap=True,
        extra_pipeline_kwargs=None,
        use_target_transform=True,
        target_transform="yeo-johnson",  # "yeo-johnson" | "box-cox" | "log1p" | "none"
        target_standardize=True,        # only applies to PowerTransformer
    ):
        """
        Initialize the XGBoost regression model with target transformation support.
        
        Parameters
        ----------
        use_target_transform : bool, default=True
            Whether to apply target transformation
        target_transform : str, default="yeo-johnson"
            Type of target transformation: "yeo-johnson", "box-cox", "log1p", or "none"
        target_standardize : bool, default=True
            Whether to standardize the transformed target (only applies to PowerTransformer)
        """
        # Initialize base class first
        super().__init__(
            search_cv_factory=search_cv_factory,
            cv_splitter=cv_splitter,
            non_feature_columns=non_feature_columns,
            model_type=model_type,
            model_name=model_name,
            model_parameter_grid=model_parameter_grid,
            use_gpu=use_gpu,
            prediction_time_column_name=prediction_time_column_name,
            true_label_name=true_label_name,
            predicted_label_name=predicted_label_name,
            importance_type=importance_type,
            has_feature_importance=has_feature_importance,
            has_shap=has_shap,
            extra_pipeline_kwargs=extra_pipeline_kwargs,
        )
        
        # Initialize transformer parameters
        self.use_target_transform = use_target_transform
        self.target_transform = target_transform
        self.target_standardize = target_standardize
        
        # Validate transformer parameters
        valid_transforms = ["yeo-johnson", "box-cox", "log1p", "none"]
        if (self.use_target_transform and 
            self.target_transform not in valid_transforms):
            raise ValueError(f"Unknown target_transform: {self.target_transform}")

    def _get_transformer(self):
        """Get the appropriate target transformer based on configuration."""
        if not self.use_target_transform or self.target_transform == "none":
            return None
            
        if self.target_transform in ("yeo-johnson", "box-cox"):
            return PowerTransformer(method=self.target_transform, standardize=self.target_standardize)
            
        if self.target_transform == "log1p":
            # For log1p, we'll use a custom transformer
            from sklearn.preprocessing import FunctionTransformer
            return FunctionTransformer(func=np.log1p, inverse_func=np.expm1)
            
        raise ValueError(f"Unknown target_transform: {self.target_transform}")

    def _new_pipeline(self):
        """Create the XGBoost pipeline with optional target transformation."""
        tree_method = "gpu_hist" if self.use_gpu else "auto"
        
        # Create the base estimator
        base_estimator = xgb.sklearn.XGBRegressor(tree_method=tree_method)
        
        # Create the feature pipeline
        feature_pipeline = Pipeline([
            ("feature_selection", SelectKBest(score_func=f_regression).set_output(transform="pandas")),
            ("estimator", base_estimator)
        ])
        
        # Apply target transformation if configured
        transformer = self._get_transformer()
        if transformer is not None:
            return TransformedTargetRegressor(
                regressor=feature_pipeline,
                transformer=transformer,
                check_inverse=False  # Disable inverse check for log1p
            )
        else:
            return feature_pipeline

    def feature_importances(self, training_data: pd.DataFrame) -> pd.DataFrame:
        """
        Calculate feature importance for the model.
        """
        # Get the actual estimator from the pipeline
        if hasattr(self.pipeline, 'regressor_'):
            # TransformedTargetRegressor case
            estimator = self.pipeline.regressor_.named_steps['estimator']
        else:
            # Direct pipeline case
            estimator = self.pipeline.named_steps['estimator']
            
        raw_importance = estimator.get_booster().get_score(importance_type=self.importance_type)
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