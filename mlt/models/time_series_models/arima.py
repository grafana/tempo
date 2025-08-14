import itertools
import json
import joblib
import numpy as np
import pandas as pd
from pathlib import Path
from statsmodels.tsa.arima.model import ARIMA
from sklearn.pipeline import Pipeline
from sklearn.base import BaseEstimator, TransformerMixin
from sklearn.model_selection import TimeSeriesSplit
from mlt.models.base_models import MLTBaseRegression

class DummyEstimator(BaseEstimator):
    """Placeholder estimator for ARIMA that satisfies pipeline requirements."""
    def fit(self, X, y=None):
        return self
    def predict(self, X):
        return np.zeros(len(X))

class ArimaSearchCV:
    """
    Custom ARIMA Hyperparameter Search using Time-Series Cross Validation.
    """
    def __init__(self, param_grid, scoring, cv=3):
        self.param_grid = param_grid  # Grid of (p, d, q) values
        self.scoring = scoring  # Function to evaluate model
        self.cv = cv  # Number of CV folds
        self.best_params_ = None
        self.best_score_ = float("inf")  # Minimize AIC/BIC
        self.best_model_ = None
        self.cv_results_ = []  # Ensure cv_results_ is a list of dicts

    def fit(self, X: pd.Series):
        """
        Perform a grid search over (p, d, q) parameters using time-series cross-validation.
        """
        tscv = TimeSeriesSplit(n_splits=self.cv)
        param_combinations = list(itertools.product(*self.param_grid.values()))
        
        for params in param_combinations:
            p, d, q = params
            scores = []
            
            for train_idx, test_idx in tscv.split(X):
                train, test = X.iloc[train_idx], X.iloc[test_idx]
                
                try:
                    model = ARIMA(train, order=(p, d, q)).fit()
                    pred = model.forecast(steps=len(test))
                    score = self.scoring(test, pred)  # Custom scoring function
                    scores.append(score)
                except:
                    continue  # Skip invalid configurations
            
            mean_score = np.mean(scores)
            self.cv_results_.append({"params": {"p": p, "d": d, "q": q}, "score": mean_score})
            
            if mean_score < self.best_score_:
                self.best_score_ = mean_score
                self.best_params_ = {"p": p, "d": d, "q": q}
                self.best_model_ = ARIMA(X, order=(p, d, q)).fit()

    def predict(self, steps=10):
        """
        Forecast future values using the best ARIMA model.
        """
        return self.best_model_.forecast(steps=steps)

    def get_params(self):
        return self.best_params_

class MLTArimaModel(MLTBaseRegression):
    def __init__(
        self,
        non_feature_columns,
        model_type,
        model_name,
        model_parameter_grid,
        prediction_time_column_name,
        true_label_name,
        predicted_label_name,
        importance_type="",
        has_feature_importance=False,
        has_shap=False,
        extra_pipeline_kwargs=None,
        experiment_path="outputs/experiments",  # Path to save experiment results
    ):
        super().__init__(
            search_cv_factory=None,  # ARIMA does not use search CV
            cv_splitter=None,  # No CV needed for ARIMA
            non_feature_columns=non_feature_columns,
            model_type=model_type,
            model_name=model_name,
            model_parameter_grid=model_parameter_grid,
            use_gpu=False,  # ARIMA does not support GPU
            prediction_time_column_name=prediction_time_column_name,
            true_label_name=true_label_name,
            predicted_label_name=predicted_label_name,
            importance_type=importance_type,
            has_feature_importance=has_feature_importance,
            has_shap=has_shap,
            extra_pipeline_kwargs=extra_pipeline_kwargs,
        )
        self.best_model_ = None
        self.best_params_ = None
        self.cv_results_ = []  # Ensure cv_results_ is a list
        self.pipeline = self._new_pipeline()
        self.experiment_path = Path(experiment_path) / model_name
        self.experiment_path.mkdir(parents=True, exist_ok=True)
    
    def _new_pipeline(self):
        """
        Returns a dummy pipeline to satisfy parent class expectations.
        """
        return Pipeline([
            ("estimator", DummyEstimator())
        ])

    def train_model(self, data: pd.DataFrame, forecast_steps=10):
        """
        Train the ARIMA model and store best parameters.
        """
        self.logger.info(f"Training ARIMA model for {self.model}")
        
        # Ensure time series is indexed correctly
        target_series = data.set_index(self.prediction_time_column_name)[self.true_label_name]
        param_grid = {"p": [1, 2, 3], "d": [1], "q": [1, 2, 3]}
        
        # Hyperparameter search
        search_cv = ArimaSearchCV(param_grid, scoring=self._evaluate_arima_model)
        search_cv.fit(target_series)

        # Store best model and parameters
        self.best_model_ = search_cv.best_model_
        self.best_params_ = search_cv.best_params_
        self.cv_results_ = search_cv.cv_results_
        
        # Store default forecast length
        self.forecast_steps = forecast_steps

        self.logger.info(f"Best ARIMA parameters: {self.best_params_}")


    def tune_hyperparameters(self, data: pd.DataFrame):
        """
        Override method to disable hyperparameter tuning since ARIMA does not use it.
        """
        self.logger.info("ARIMA does not use hyperparameter tuning via search CV.")

    def _get_prediction_data(self, features: pd.DataFrame, index: pd.Index) -> pd.DataFrame:
        """
        Forecasts using the best trained ARIMA model.
        """
        if self.best_model_ is None:
            raise ValueError("Model not trained. Call train_model() first.")

        # Get predictions using the fitted model
        predictions = self.best_model_.fittedvalues
        
        # Extend predictions if needed to match index length
        if len(predictions) < len(index):
            forecast = self.best_model_.forecast(steps=len(index) - len(predictions))
            predictions = pd.concat([predictions, forecast])
        
        # Ensure predictions match index length
        predictions = predictions[:len(index)]
        
        return pd.DataFrame(
            {self.predicted_label_name: predictions},
            index=index
        )

    
    def feature_importances(self, training_data: pd.DataFrame) -> pd.DataFrame:
        """
        ARIMA does not provide feature importances, returns an empty DataFrame.
        """
        return pd.DataFrame(columns=["feature", "score"])

    def get_shap_values(self, feature_data: pd.DataFrame, predictions: pd.Series) -> pd.DataFrame:
        """
        Computes residuals as a proxy for model explanations.
        """
        feature_data = feature_data.copy()
        feature_data["residuals"] = feature_data[self.true_label_name] - predictions
        feature_data[self.predicted_label_name] = predictions.values
        return feature_data

    def _evaluate_arima_model(self, actual, predicted):
        """
        Computes mean absolute error as a scoring function for ARIMA tuning.
        """
        return np.mean(np.abs(actual - predicted))

    def _save_experiment_results(self):
        """
        Saves trained ARIMA model, best parameters, and CV results.
        """
        # Save trained model
        model_path = self.experiment_path / "best_arima_model.pkl"
        joblib.dump(self.best_model_, model_path)
        self.logger.info(f"Saved ARIMA model to {model_path}")

        # Save best parameters
        params_path = self.experiment_path / "best_params.json"
        with open(params_path, "w") as f:
            json.dump(self.best_params_, f, indent=4)
        self.logger.info(f"Saved ARIMA parameters to {params_path}")

        # Save cross-validation results
        cv_results_path = self.experiment_path / "cv_results.csv"
        pd.DataFrame(self.cv_results_).to_csv(cv_results_path, index=False)
        self.logger.info(f"Saved ARIMA CV results to {cv_results_path}")
    
    def forecast(self, steps=10):
        """
        Forecast future values using the trained ARIMA model.
        """
        if self.best_model_ is None:
            raise ValueError("ARIMA model is not trained yet. Call train_model() before forecasting.")
        
        return self.best_model_.forecast(steps=steps)

