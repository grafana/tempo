import logging 
from abc import ABC, abstractmethod
from pathlib import Path
from typing import Dict, Callable, Union, List, Tuple, Optional
from omegaconf import DictConfig, OmegaConf
import matplotlib.pyplot as plt
from matplotlib.patches import Patch
import numpy as np
import pandas as pd
import seaborn as sns


from sklearn.metrics import (
    mean_absolute_error,
    root_mean_squared_error,
)

from mlt.models.base_models import MLTBaseModel
from mlt.experiment_pipeline.training_pipeline import PredictionResults

ScoringFunction = Callable[[pd.Series, pd.Series], float]

class ExperimentLogger(ABC):
    def __init__(
        self,
        experiment_path: Path,
        inner_cv_name: str,
        prediction_columns_to_save: List[str],
        prediction_time_column_name: str,
        true_label_name: str,
        predicted_label_name: str,
    ):
        self.prediction_time_column_name = prediction_time_column_name
        self.inner_cv_name = inner_cv_name
        self.metrics_save_path = experiment_path / "ml_metrics"
        self.metrics_save_path.mkdir(exist_ok=True)
        self.experiment_path = experiment_path
        self.resampling_to_frequency_code = {"Monthly": "M", "Annual": "A"}
        self.prediction_columns_to_save = prediction_columns_to_save
        self.true_label_name = true_label_name
        self.predicted_label_name = predicted_label_name

    def default_save_experiment_artifacts_to_disk(
        self,
        prediction_model: MLTBaseModel,
        all_results: PredictionResults,
        last_results: PredictionResults,
        model_name: str,
        run_config: Optional[DictConfig] = None,
    ):
        """
        Save experiment artifacts to disk including predictions, feature importance, model parameters, and SHAP values.

        Parameters
        ----------
        prediction_model : MLTBaseModel
            The trained model used to generate predictions
        all_results : PredictionResults
            Results from all predictions including out-of-sample predictions, in-sample predictions,
            feature importance, model parameters and SHAP values
        last_results : PredictionResults
            Results from the last prediction period
        model_name : str
            Name of the model to use when saving model file

        Notes
        -----
        Saves the following files:
        - historicalfeature_importance.csv: Feature importance scores
        - historicalpredictions.pkl: Out-of-sample predictions
        - historical_in_sample_predictions.pkl: In-sample predictions  
        - historical_model_params.csv: Model parameters
        - {model_name}.pkl: Saved model file
        - all_shaps.parquet: SHAP values if available
        """
        predictions = all_results.out_of_sample_predictions
        print(predictions.prediction_date.min())
        print(predictions.prediction_date.max())
        if all_results.feature_importances is not None:
            self.save_feature_importances(last_results.feature_importances, prediction_model.importance_type)
            self.save_feature_importances(
                all_results.feature_importances.groupby("feature")["score"].sum().to_frame().reset_index(), 
                prediction_model.importance_type,
                "_aggregate",
            )
            
            all_results.feature_importances.to_csv(self.experiment_path / "historical_feature_importance.csv", index=False)

        self.perform_model_checks(predictions)
        self.save_cv_split(prediction_model.cv_set)
        predictions[self.prediction_columns_to_save].to_pickle(self.experiment_path / "historicalpredictions.pkl")
        all_results.in_sample_predictions.to_pickle(self.experiment_path / "in_sample_predictions.pkl")
        all_results.model_params.to_csv(self.experiment_path / "model_params_history.csv", index=False)
        
        #all_results.model_params.to_csv(self.experiment_path / "historical_model_params.csv", index=False)
        #prediction_model.save(self.experiment_path / f"{model_name}.pkl")
        model_folder = self.experiment_path / model_name
        model_folder.mkdir()
        prediction_model.to_pickle(model_folder / f"{model_name}.pkl")
        if run_config is not None:
            with open(model_folder / "config.yaml", "w") as config_file:
                OmegaConf.save(run_config, config_file, resolve=True)

        if all_results.shaps is not None:
            all_results.shaps.to_parquet(self.experiment_path / "all_shaps.parquet")
            
    def save_unlabelled_prediction_artifacts_to_disk(
            self, unlabelled_predictions: pd.DataFrame, unlabelled_shaps: Optional[pd.DataFrame] = None
    ):
        """
        Save unlabelled prediction artifacts to disk.

        Parameters
        ----------
        unlabelled_predictions : pd.DataFrame
            DataFrame containing predictions for unlabelled data
        unlabelled_shaps : Optional[pd.DataFrame], default=None
            DataFrame containing SHAP values for unlabelled predictions, if available

        Notes
        -----
        Saves the following files:
        - recent_unlabelled_predictions.csv: Predictions for unlabelled data
        - recent_unlabelled_shaps.parquet: SHAP values for unlabelled predictions (if provided)
        """
        unlabelled_predictions[self.prediction_columns_to_save].to_csv(self.experiment_path / "recent_unlabelled_predictions.csv", index=False)
        if unlabelled_shaps is not None:
            unlabelled_shaps.to_parquet(self.experiment_path / "recent_unlabelled_shaps.parquet")

    def save_metrics_over_time_plot(self, data: pd.DataFrame, metric_dict: Dict[str,ScoringFunction] = None):
        if metric_dict is None:
            metric_dict = self.metric_functions  
        for resampling_period, resampling_code in self.resampling_to_frequency_code.items():
            for metric_name, metric in metric_dict.items():
                print(metric_name)
                metric_by_class_and_period = self.find_metrics_by_period(data, metric, resampling_code)
                metric_by_class_and_period.plot(figsize=(17,9))
                plt.title(f"{resampling_period} {metric_name}", fontsize=15)
                plt.ylabel(metric_name, fontsize=15)
                plt.savefig(self.metrics_save_path / f"{metric_name}-{resampling_period}.png", bbox_inches="tight")
                print(self.metrics_save_path/ f"{metric_name}_{resampling_period}.png")
                plt.close()

    def get_ml_metrics(self, true_label: pd.Series, predicted_label: pd.Series) -> Dict[str, float]:
        """
        Calculate machine learning metrics between true and predicted labels.

        Parameters
        ----------
        true_label : pd.Series
            Series containing the actual/true labels
        predicted_label : pd.Series
            Series containing the model's predicted labels

        Returns
        -------
        Dict[str, float]
            Dictionary mapping metric names to their calculated values based on 
            the metric functions defined in self.metric_functions
        """
        return {name: metric(true_label, predicted_label) for name, metric in self.metric_functions.items()}
    
    def save_feature_importances(self, feature_importances: pd.DataFrame, importance_type: str = "", file_name_suffix: str = ""):
        """
        Save feature importance scores and visualizations.

        Parameters
        ----------
        feature_importances : pd.DataFrame
            DataFrame containing feature names and importance scores, with columns 'feature' and 'score'
        importance_type : str, optional
            Type of feature importance metric used (e.g. 'gain', 'weight'), by default ""

        Notes
        -----
        Saves two files:
        - CSV file with raw feature importance scores at {metrics_save_path}/feature_importances_{importance_type}.csv
        - Bar plot visualization of top 30 features at {metrics_save_path}/feature_importances.png
        """
        feature_importances = feature_importances.sort_values("score", ascending=False)
        feature_importances.to_csv(self.metrics_save_path / f"feature_importances_{importance_type}.csv")
        fig, ax = plt.subplots(figsize=(10, 10), dpi=100)
        feature_importances = feature_importances.sort_values("score", ascending=False)
        try:
            feature_importance_title = "Feature Importance"
            if importance_type:
                feature_importance_title += f": {importance_type}"
            ax.set_title(feature_importance_title, fontsize=15)
            sns.barplot(x="score", y="feature", data=feature_importances.head(30))
            ax.set_xlabel(f"{importance_type} score")
            fig.savefig(self.metrics_save_path / "feature_importances.png", bbox_inches="tight")
            plt.close()
        except ValueError:
            logging.warning(f"Feature importances are probably empty:\n{feature_importances}")
            plt.close()
    
    def find_metrics_by_period(
        self, label_data: pd.DataFrame, metric: ScoringFunction, resampling_code: str
    ) -> Union[pd.Series, pd.DataFrame]:
        """
        Calculate metrics aggregated by time period.

        Parameters
        ----------
        label_data : pd.DataFrame
            DataFrame containing true labels, predicted labels, and prediction timestamps
        metric : ScoringFunction
            Function that calculates metric between true and predicted labels
        resampling_code : str
            Pandas resampling frequency code (e.g. 'D' for daily, 'M' for monthly)

        Returns
        -------
        Union[pd.Series, pd.DataFrame]
            Time series of metric values resampled to specified frequency
        """
        metric_by_period = label_data.resample(resampling_code, on=self.prediction_time_column_name).apply(
            lambda x: metric(x[self.true_label_name], x[self.predicted_label_name])
        )
        metric_by_period = metric_by_period.dropna(how="all")
        return metric_by_period
    
    
    def save_cv_split(self, cv_set: List[Tuple[pd.Series, pd.Series]]):
        """
        Save visualization of cross-validation splits.

        Creates a plot showing the training and testing set splits across time periods
        for each cross-validation fold.

        Parameters
        ----------
        cv_set : List[Tuple[pd.Series, pd.Series]]
            List of (train_indices, test_indices) tuples defining the CV splits
            Each tuple contains Series of datetime indices for train and test sets

        Saves
        -----
        Last CV Split.png : PNG file
            Plot showing train/test splits across time, with:
            - X-axis showing dates
            - Y-axis showing CV fold number 
            - Blue regions indicating training data
            - Red regions indicating test data
        """
        fig, ax = plt.subplots(figsize=(15, len(cv_set)), dpi=100)
        cmap_cv = plt.cm.coolwarm
        all_dates = np.sort(
            pd.concat([cv_set[i][j] for i in range(len(cv_set)) for j in range(len(cv_set[0]))]).unique()
        )
        tick_formatter = TickFormatter(all_dates)
        for ii, (train_idx, val_idx) in enumerate(cv_set):
            indices = np.nan * np.ones(all_dates.shape)
            indices[np.intersect1d(all_dates, train_idx, return_indices=True)[1]] = 1
            indices[np.intersect1d(all_dates, val_idx, return_indices=True)[1]] = 0
            # Scatterplot on indexes (not dates) needed to not have gaps caused by non-trading dates
            ax.scatter(
                range(len(indices)),
                ii * np.ones(len(indices)),
                c=indices,
                marker="_",
                lw=10,
                cmap=cmap_cv,
                vmin=-0.2,
                vmax=1.2,
            )
        ax.xaxis.set_major_formatter(plt.FuncFormatter(tick_formatter.format_func))
        ax.set(yticks=range(len(cv_set)), xlabel="Date", ylabel="CV iteration", ylim=(-0.5, len(cv_set) - 0.5))
        ax.set_title(self.inner_cv_name, fontsize=15)
        ax.legend(
            [Patch(color=cmap_cv(0.8)), Patch(color=cmap_cv(0.02))], ["Training set", "Testing set"], loc=(1.02, 0.8)
        )
        fig.savefig(self.metrics_save_path / "Last CV Split.png", bbox_inches="tight")
        plt.close(fig)

    
    @property
    @abstractmethod
    def metric_functions(self) -> Dict[str, ScoringFunction]:
        """
        Dictionary mapping metric names to their scoring functions.
        """
        pass

    @abstractmethod
    def generate_and_save_ml_artifacts(self, true_label: pd.Series, predicted_label: pd.Series):
        """
        Generate and save model artifacts based on true and predicted labels.

        This method should be implemented by subclasses to save any required model artifacts
        like plots, metrics, or other evaluation outputs using the true labels and model 
        predictions.

        Parameters
        ----------
        true_label : pd.Series
            Series containing the actual/ground truth values
        predicted_label : pd.Series
            Series containing the model's predicted values
        """
        pass

    @abstractmethod
    def perform_model_checks(self, data: pd.DataFrame):
        pass


class RegressionExperimentLogger(ExperimentLogger):
    @property
    def metric_functions(self) -> Dict[str, ScoringFunction]:
        return {
            "rmse": lambda x, y: root_mean_squared_error(x, y),
            "mae": mean_absolute_error,
            "correlation": lambda x, y: np.corrcoef(x, y)[0, 1],
        }
        #df_post_processed_2025-03-07.parquet
    
    def generate_and_save_ml_artifacts(self, true_label: pd.Series, predicted_label: pd.Series):
        overall_metric_df = pd.Series(self.get_ml_metrics(true_label, predicted_label))
        overall_metric_df.to_csv(self.metrics_save_path / "overall_metrics.csv")


    def save_residual_kde_plot(self, data: pd.DataFrame):
        """
        Generate and save a kernel density estimation (KDE) plot of prediction residuals.

        Creates a visualization of the residual distribution to help identify potential issues
        like skewness, multimodality, or other anomalies in the model's prediction errors.
        The plot is saved as a PNG file.

        Parameters
        ----------
        data : pd.DataFrame
            DataFrame containing both true values and predicted values in columns specified
            by self.true_label_name and self.predicted_label_name
        """
        residuals = data[self.true_label_name] - data[self.predicted_label_name]
        fig, ax = plt.subplots(figsize=(17, 9))
        sns.kdeplot(residuals, ax=ax)
        plt.title("Residual Histogram", fontsize=15)
        plt.savefig(self.metrics_save_path / "residuals_kde.png")
        plt.close()


    def save_fit_plot(self, data: pd.DataFrame):
        """
        Generate and save a time series plot comparing actual and predicted values.

        Creates a line plot showing both the true values and model predictions over time,
        allowing visual assessment of the model's predictive performance and any temporal
        patterns or discrepancies.

        Parameters
        ----------
        data : pd.DataFrame
            DataFrame containing:
            - True values in column specified by self.true_label_name
            - Predicted values in column specified by self.predicted_label_name  
            - Timestamps in column specified by self.prediction_time_column_name
        """
        data = data.set_index(self.prediction_time_column_name)
        fit_df = pd.DataFrame(
            {"true_value": data[self.true_label_name], "predicted_value": data[self.predicted_label_name]}
        )
        fit_df.plot(figsize=(17, 9))
        plt.title("True vs Predicted Values", fontsize=15)
        plt.savefig(self.metrics_save_path / "fit_plot.png")
        plt.close()


    def save_predicted_true_scatter_plot(self, data: pd.DataFrame, ols_line: bool = True):
        """
        Generate and save a scatter plot comparing actual versus predicted values.

        Creates a scatter plot to visualize the correlation between true values and model
        predictions. This helps assess the model's accuracy and identify any systematic
        biases or nonlinear relationships in the predictions.

        Parameters
        ----------
        data : pd.DataFrame
            DataFrame containing both true values and predicted values in columns specified
            by self.true_label_name and self.predicted_label_name
        """
        """
        data = data.rename(columns={self.predicted_label_name: "predicted_value", self.true_label_name: "actual_value"})
        fig, _ = plot_scatter(
            [data[x] for x in ["predicted_value", "actual_value"]],
            labs=["predicted_value", "actual_value"],
            title="Actual vs Predicted",
            fnum=1,
            ols_line=ols_line, 
            write_path=self.metrics_save_path / "actual_vs_predicted.png",
            s=0.1,
        )
        plt.close(fig)
        """
        # Rename columns for clarity
        data = data.rename(columns={self.predicted_label_name: "predicted_value", self.true_label_name: "actual_value"})
        
        # Create figure and axis
        fig, ax = plt.subplots(figsize=(10, 8))
        
        # Create scatter plot
        ax.scatter(data["predicted_value"], data["actual_value"], s=0.1, alpha=0.7)
        
        # Add labels and title
        ax.set_xlabel("Predicted Value")
        ax.set_ylabel("Actual Value")
        ax.set_title("Actual vs Predicted")
        
        # Add regression line if requested
        if ols_line:
            from sklearn.linear_model import LinearRegression
            # Reshape for sklearn
            x = data["predicted_value"].values.reshape(-1, 1)
            y = data["actual_value"].values
            # Fit regression model
            model = LinearRegression().fit(x, y)
            # Generate predictions for the line
            x_range = np.linspace(data["predicted_value"].min(), data["predicted_value"].max(), 100).reshape(-1, 1)
            y_pred = model.predict(x_range)
            # Plot regression line
            ax.plot(x_range, y_pred, color='red', linewidth=2)
        
        # Add y=x reference line
        min_val = min(data["predicted_value"].min(), data["actual_value"].min())
        max_val = max(data["predicted_value"].max(), data["actual_value"].max())
        ax.plot([min_val, max_val], [min_val, max_val], 'k--', alpha=0.5)
        
        # Save figure
        plt.savefig(self.metrics_save_path / "actual_vs_predicted.png", bbox_inches="tight")
        
        # Close figure to prevent display in notebooks
        plt.close(fig)




        
    def perform_model_checks(self, data: pd.DataFrame):
        self.save_metrics_over_time_plot(data)
        self.save_fit_plot(data)
        self.save_predicted_true_scatter_plot(data)
        self.save_residual_kde_plot(data)
        self.generate_and_save_ml_artifacts(data[self.true_label_name], data[self.predicted_label_name])
            
    

class TickFormatter:
    def __init__(self, all_dates: np.ndarray):
        self.data = all_dates

    def format_func(self, value, tick_number):
        if 0 <= int(value) < len(self.data):
            return np.datetime_as_string(self.data[int(value)], unit="D")
        else:
            return ""

