import logging
from abc import ABC, abstractmethod
from pathlib import Path
from time import time
from typing import Callable, List, Dict, Any, Tuple, Union

import pickle
import joblib
import numpy as np
import pandas as pd
import shap
from sklearn.base import BaseEstimator
from sklearn.pipeline import Pipeline

from mlt.typing import SearchCV, CVSplitter


class MLTBaseModel(ABC):
    def __init__(
        self,
        search_cv_factory: Callable[[BaseEstimator], SearchCV],
        cv_splitter: CVSplitter,
        non_feature_columns: List[str],
        model_type: str,
        model_name: str,
        model_parameter_grid: Dict[str, List[Any]],
        use_gpu: bool,
        prediction_time_column_name: str,
        true_label_name: str,
        predicted_label_name: str,
        importance_type="",
        has_feature_importance: bool = True,
        has_shap: bool = True,
        extra_pipeline_kwargs: Dict = None,
    ):
        """
        Base class for MLT models that provides common functionality for
        training, prediction, and evaluation.

        Parameters
        ----------
        search_cv_factory : Callable[[BaseEstimator], SearchCV]
            Factory function that creates a SearchCV object for hyperparameter
            tuning
        cv_splitter : CVSplitter
            Cross validation splitter object that defines train/test splits
        non_feature_columns : List[str]
            Column names that should be excluded from feature data
        model_type : str
            Type of model (e.g. "regression", "classification")
        model_name : str
            Name identifier for this model
        model_parameter_grid : Dict[str, List[Any]]
            Grid of hyperparameters to search during tuning
        use_gpu : bool
            Whether to use GPU acceleration if available
        prediction_time_column_name : str
            Name of column to store prediction timestamps
        true_label_name : str
            Name of column containing true labels
        predicted_label_name : str
            Name of column to store predicted labels
        importance_type : str, optional
            Type of feature importance to calculate, by default ""
        has_feature_importance : bool, optional
            Whether model supports feature importance calculation, by default True
        has_shap : bool, optional
            Whether model supports SHAP value calculation, by default True
        extra_pipeline_kwargs : Dict, optional
            Additional keyword arguments for pipeline steps, by default None
        """
        if extra_pipeline_kwargs is None:
            extra_pipeline_kwargs = {}
        self.logger = logging.getLogger(__name__)
        self.non_feature_columns = non_feature_columns

        self.model_type = model_type
        self.model = model_name
        self.model_parameter_grid = model_parameter_grid
        self.use_gpu = use_gpu
        self.cv_set: List[Tuple[pd.Series, pd.Series]] = []
        self.has_feature_importance = has_feature_importance
        self.has_shap = has_shap
        self.extra_pipeline_kwargs = extra_pipeline_kwargs
        self.importance_type = importance_type
        self.pipeline: Pipeline = self._new_pipeline()
        self.search_cv_factory = search_cv_factory
        self.cv_splitter = cv_splitter
        self.prediction_time_column_name = prediction_time_column_name
        self.true_label_name = true_label_name
        self.predicted_label_name = predicted_label_name
        self.cv_results_ = None
        self.cv_params_ = None
        if "estimator" not in self.pipeline.named_steps.keys():
            raise ValueError("Model pipeline must contain a step named 'estimator'")

    def load_pipeline_from_file(self, model_path: Union[Path, str]):
        """
        Load an existing pipeline from a file and set it as `self.pipeline`
        :param model_path: Location of pipeline file
        """
        new_pipeline = joblib.load(model_path)
        # Things could get weird if a different pipeline is loaded so ensure that the steps match
        fail = new_pipeline.named_steps.keys() != self._new_pipeline().named_steps.keys()
        for name, step in self._new_pipeline().named_steps.items():
            new_step = new_pipeline.named_steps[name]
            fail |= type(step) != type(new_step)

        if fail:
            raise ValueError("The steps in pipeline from the file do not match the steps for this model")
        self.pipeline = new_pipeline

    def to_pickle(self, model_path: Union[Path, str]):
        """
        Save model to the given path using pickle
        """
        with open(model_path, "wb") as pkl_out:
            pickle.dump(self,pkl_out)

    @staticmethod
    def from_pickle(model_path: Union[Path, str]):
        """
        Load model from the given path using pickle
        """
        with open(model_path, "rb") as pkl_in:
            return pickle.load(pkl_in)

    def save(self, model_path: Union[Path, str]):
        """
        Save model to the given path using joblib
        """
        with open(model_path, "wb") as pkl_out:
            joblib.dump(self.pipeline, pkl_out)

    @abstractmethod
    def _new_pipeline(self) -> Pipeline:
        """
        Get a new instance of the desired pipeline to be used in this model
        :return: A pipeline that has at least on step named "estimator"
        """
        ...

    @abstractmethod
    def get_fold_from_index(self, data: pd.DataFrame, indices: np.ndarray) -> pd.DataFrame:
        """
        Get the data at the passed indices plus any additional values that may be needed. Sequence models need a full
        sequence length before making predictions so their implementation of this method will prepend the necessary
        samples.
        :param data: Complete dataset to be indexed
        :param indices: Indices to get the data at
        :return: The data selected at the necessary spots
        """
        return data.loc[indices]

    def transform_data_to_features(self, data: pd.DataFrame) -> pd.DataFrame:
        """
        Convert data to the input format needed for pipeline.fit()

        :param data: data
        :return: Data for training. Could be DataFrame or numpy array or Tensor
        """
        features = data.loc[:, [col for col in data.columns if col not in self.non_feature_columns]]
        # just in case
        if self.true_label_name in features.columns:
            logging.warning(
                f"{self.true_label_name} is not in the passed `non_feature_columns` it will be removed but you should "
                f"add it to `non_feature_columns` as it could be used as a feature somewhere else"
            )
            features = features.drop(columns=[self.true_label_name])
        return features

    def _match_data_to_features(self, data, features):
        """
        In sequential models `transform_data_to_features` may alter the shape of the data which causes issues because we
        get the inner cv splits from data and not features. Override this if needed
        :param data: data that will be passed to splitter
        :param features: features that may have been cut down to make sequences or something
        :return: data with the right indices for cv
        """
        return data

    def train_model(self, data: pd.DataFrame):
        """
        Train the model using the provided data
        :param data: DataFrame containing training features and labels
        """
        self.logger.info(
            f"Model Training Start. Start date: {data[self.prediction_time_column_name].min().strftime('%Y-%m-%d')}. "
            f"End date: {data[self.prediction_time_column_name].max().strftime('%Y-%m-%d')}"
        )

        tik = time()
        features = self.transform_data_to_features(data)
        data = self._match_data_to_features(data, features)
        labels = self._prepare_labels(data[self.true_label_name])

        self.pipeline.fit(X=features, y=labels, **self.extra_pipeline_kwargs)

        self.logger.info(f"Model Training finished in {round(time() - tik, 2)} seconds")

    @abstractmethod
    def _prepare_labels(self, label_series) -> np.ndarray:
        """
        Prepare labels for training and parameter tuning by turning them into a numpy array and encoding them if needed
        :param label_series: labels for the data
        :return: array of transformed labels
        """
        ...

    def tune_hyperparameters(self, data: pd.DataFrame):
        """
        Tune the hyperparameters using the provided data
        :param data: DataFrame containing training features and labels
        """
        self.logger.info(
            f"Model Tuning Start. Start date: {data[self.prediction_time_column_name].min().strftime('%Y-%m-%d')}. "
            f"End date: {data[self.prediction_time_column_name].max().strftime('%Y-%m-%d')}"
        )
        tik = time()

        features = self.transform_data_to_features(data)
        data = self._match_data_to_features(data, features)
        labels = self._prepare_labels(data[self.true_label_name])
        dates = data[self.prediction_time_column_name]

        # generator must be consumed because generators can't be pickled
        indices = list(self.cv_splitter.split(data))
        self.cv_set = [
            (dates.iloc[train_indices], dates.iloc[test_indices]) for (train_indices, test_indices) in indices
        ]
        clf = self.search_cv_factory(self.pipeline)
        clf = clf.set_params(cv=indices)
        clf.fit(features, labels, **self.extra_pipeline_kwargs)

        self.pipeline = clf.best_estimator_
        self.cv_params_ = self._trim_cv_params(clf.get_params(deep=False))
        self.cv_results_ = clf.cv_results_
        self.logger.info(f"Model Tuning finished in {round(time() - tik, 2)} seconds")

    @staticmethod
    def _trim_cv_params(params: Dict[str, Any]) -> Dict[str, Any]:
        """
        Remove unneeded keys and values from search_cv params. These have the estimator as an object which we don't need
        another copy of.

        :param params: Params from self.search_cv_factory right after it was fit
        :return: Params without the estimator
        """
        params.pop("estimator")
        return params

    def get_predictions(self, data: pd.DataFrame) -> pd.DataFrame:
        """
        Retrieve predictions for the provided dataframe
        :param data: DataFrame containing features
        :return DataFrame with predictions and prediction probabilities as well as the original data
        """
        data = data.copy()
        features = self.transform_data_to_features(data)
        data = self._match_data_to_features(data, features)
        predictions = self._get_prediction_data(features, data.index)
        data[predictions.columns] = predictions
        return data

    @abstractmethod
    def _get_prediction_data(self, features: pd.DataFrame, index: pd.Index) -> pd.DataFrame:
        """
        Get a dataframe of just predictions, could have the columns
        `self.probability_column_names + [self.predicted_label_name]` for classification
        or just `[self.predicted_label_name]` for regression
        :param features: Only the features to use for predictions
        :param index: The index for the resultant dataframe
        :return: Dataframe of relevant prediction values
        """
        ...

    @abstractmethod
    def feature_importances(self, training_data: pd.DataFrame) -> pd.DataFrame:
        """
        Get feature importances from model
        :param training_data: training data. Not used for models other than SVM models
        :return: Dataframe containing feature_importances and feature names
        """
        ...

    @abstractmethod
    def get_shap_values(self, feature_data, predictions) -> pd.DataFrame:
        """
        Calculate the shap values for the generated predictions.
        :param feature_data: dataframe that contains the feature data
        :param predictions: dataframe of generated model predictions
        :return: dataframe of shap values
        """
        ...

    def get_booster_if_needed(self) -> Any:
        """
        Convenience method for accessing xgboost models' boosters
        :return: The booster for the model if pipeline["estimator"] has one otherwise just the estimator itself
        """
        try:
            return self.pipeline["estimator"].get_booster()
        except AttributeError:
            return self.pipeline["estimator"]

    @property
    def model_parameters(self):
        """
        Get the current parameters for the estimator
        """
        return {key: value for key, value in self.pipeline.get_params().items() if key in self.model_parameter_grid}


class MLTBaseRegression(MLTBaseModel, ABC):
    def get_fold_from_index(self, data: pd.DataFrame, indices: np.ndarray) -> pd.DataFrame:
        return super().get_fold_from_index(data, indices)

    @abstractmethod
    def _new_pipeline(self):
        pass

    def _get_prediction_data(self, features: pd.DataFrame, index: pd.Index) -> pd.DataFrame:
        return pd.DataFrame({self.predicted_label_name: self.pipeline.predict(features)}, index=index)

    def _prepare_labels(self, label_series: pd.Series) -> np.ndarray:
        return label_series.ravel()

    @abstractmethod
    def feature_importances(self, training_data: pd.DataFrame) -> pd.DataFrame:
        """
        Get feature importances from model
        :param training_data: training data. Not used for models other than SVM models
        :return: Dataframe containing feature_importances and feature names
        """
        pass

    @abstractmethod
    def get_shap_values(self, feature_data: pd.DataFrame, predictions: pd.Series) -> pd.DataFrame:
        """
        Get a dataframe of SHAP values for features and predictions. This currently works for LightGBM and XGBoost,
        but may require adjustments for other models.

        :param feature_data: Raw feature data before any selection or transformations, including metadata.
        :param predictions: Outputs from self.get_prediction_data.
        :return: A dataframe of SHAP values for each prediction and feature.
        """
        
        feature_names = [col for col in feature_data.columns if col not in self.non_feature_columns]
        booster = self.get_booster_if_needed()

        # More explicit approach for selecting the SHAP explainer
        if hasattr(booster, 'get_booster'):  # XGBoost
            shap_explainer = shap.TreeExplainer(booster)
        elif hasattr(booster, 'booster_'):  # LightGBM
            shap_explainer = shap.TreeExplainer(booster)
        else:
            shap_explainer = shap.Explainer(booster)  # Fallback for other models

        # Compute SHAP values
        transformed_features = self.pipeline.named_steps.feature_selection.transform(feature_data[feature_names])
        shap_values = shap_explainer.shap_values(transformed_features, check_additivity=False)

        # Retrieve feature names after selection
        used_names = self.pipeline.named_steps.feature_selection.get_feature_names_out(feature_names)

        # Assign SHAP values to the original dataframe
        feature_data[used_names] = shap_values
        feature_data[[x for x in feature_names if x not in used_names]] = 0  # Set SHAP values to 0 for unused features
        feature_data[self.predicted_label_name] = predictions.iloc[:, 0]

        return feature_data
