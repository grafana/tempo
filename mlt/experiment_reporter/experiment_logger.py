import logging 
from abc import ABC, abstractmethod
from pathlib import Path
from typing import Dict, Callable, Union, List, Tuple, Optional
from omegaconf import DictConfig, OmegaConf
import matplotlib.pyplot as plt
from matplotlib.patches import Patch
from sklearn.calibration import calibration_curve
import numpy as np
import pandas as pd
import seaborn as sns


from sklearn.metrics import (
    mean_absolute_error,
    root_mean_squared_error,
    r2_score,
    f1_score,
    precision_score,
    recall_score,
    confusion_matrix,
    ConfusionMatrixDisplay,
    classification_report
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
    
    def save_unlabelled_prediction_processed(
            self, unlabelled_predictions: pd.DataFrame, date: str = "2025-03-31"
    ):
        
        plot_df = unlabelled_predictions[unlabelled_predictions['evaluation_date'] == date].copy()

# Create scatter plot
        # Create figure and axis
        fig, ax = plt.subplots(figsize=(10, 6))
        
        # Create scatter plot
        ax.scatter(plot_df['overage_mrr'], plot_df['projected_amount_due'], alpha=0.5)
        ax.set_xlabel('Actual Overage MRR')
        ax.set_ylabel('Projected Amount Due') 
        ax.set_title(f'Projected vs Actual Values for {date}')

        # Add a perfect prediction line (y=x)
        max_val = max(plot_df['overage_mrr'].max(), plot_df['projected_amount_due'].max())
        ax.plot([0, max_val], [0, max_val], 'r--', label='Perfect Prediction')
        ax.legend()

        # Save figure
        fig.savefig(self.metrics_save_path / f"unlabelled_predictions_overage_mrr_{date}.png", bbox_inches="tight")
        plt.close(fig)

        # Calculate error metrics
        plot_df['absolute_error'] = abs(plot_df['projected_amount_due'] - plot_df['overage_mrr'])
        plot_df['percentage_error'] = (plot_df['absolute_error'] / plot_df['overage_mrr']) * 100

        stats = {
            'Mean Absolute Error': plot_df['absolute_error'].mean(),
            'Median Absolute Error': plot_df['absolute_error'].median(), 
            'Mean Percentage Error': plot_df['percentage_error'].mean(),
            'Median Percentage Error': plot_df['percentage_error'].median(),
            'Number of Predictions': len(plot_df)
        }

        correlation = plot_df['projected_amount_due'].corr(plot_df['overage_mrr'])
        stats['Correlation'] = correlation

        # Create figure for stats
        fig, ax = plt.subplots(figsize=(10, 6))
        ax.axis('off')
        
        # Create table
        cell_text = [[f"{value:.2f}" for value in stats.values()]]
        table = ax.table(cellText=cell_text,
                        colLabels=list(stats.keys()),
                        loc='center',
                        cellLoc='center')
        
        # Adjust table appearance
        table.auto_set_font_size(False)
        table.set_fontsize(9)
        table.scale(1.2, 2)
        
        # Add title
        plt.title(f'Prediction Statistics for {date}', pad=20)
        
        # Save figure
        fig.savefig(self.metrics_save_path / f"prediction_statistics_post_processing_{date}.png", bbox_inches="tight")
        plt.close(fig)

        unlabelled_predictions[self.prediction_columns_to_save].to_csv(self.experiment_path / "recent_unlabelled_predictions_post_processing.csv", index=False)
        


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
            "hit_rate": lambda x, y: (np.sign(x) == np.sign(y)).mean(),
            "r2": lambda x, y: r2_score(x, y),
        }

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


class ClassificationExperimentLogger(ExperimentLogger):

    def __init__(
            self, 
            experiment_path: Path,
            inner_cv_name: str,
            prediction_columns_to_save: List[str],
            class_labels: List[Any],
            label_names: List[str],
            probability_column_names: List[str],
            prediction_time_column_name: str,
            true_label_name: str,
            predicted_label_name: str,
    ):
        super().__init__(
            experiment_path,
            inner_cv_name,
            prediction_columns_to_save,
            prediction_time_column_name,
            true_label_name,
            predicted_label_name,
        )
       
        self.label_names = label_names
        self.labels = class_labels
        self.probability_column_names = probability_column_names

    @property
    def metric_functions(self) -> Dict[str, ScoringFunction]:
        return {
            "macro_f1_score": lambda x, y: f1_score(x, y, average="macro"),
            "precision_score": lambda x, y: precision_score(x, y, average="macro"),
            "recall_score": lambda x, y: recall_score(x, y, average="macro"),
         }
    
    def save_confusion_matrix(self, true_label: pd.Series, predicted_label: pd.Series):
        """
        Save confusion matrix for classification model.
        """
        normalization_options = [("Normalized", "all"), ("Not Normalized", None)]
        for title, normalize in normalization_options:
            fig, ax = plt.subplots(figsize=(10, 10))
            cm = confusion_matrix(true_label, predicted_label, normalize=normalize)
            display_labels = self.label_names[: cm.shape[0]]
            ConfusionMatrixDisplay(confusion_matrix=cm, display_labels=display_labels).plot(ax=ax)
            ax.set_title(f"Confusion Matrix {title}")
            fig.savefig(self.metrics_save_path / f"confusion_matrix_{title}.png", bbox_inches="tight")
            plt.close()
    
    def generate_and_save_ml_artifacts(self, true_label: pd.Series, predicted_label: pd.Series):
        self.save_classification_report(true_label, predicted_label)
        self.save_confusion_matrix(true_label, predicted_label)
        
    def save_classification_report(self, true_label: pd.Series, predicted_label: pd.Series):
        ml_report = classification_report(true_label, predicted_label, output_dict=True)
        pd.DataFrame(ml_report).trnaspose().to_csv(self.metrics_save_path / "classification_report.csv")

    def metric_helper(self, true_label: pd.Series, predicted_label: pd.Series, sum_axis: int):
        if len(true_label) == 0:
            return pd.Series([np.nan] * len(self.labels), index=self.label_names)
        mat = confusion_matrix(true_label, predicted_label, labels=self.labels)
        num_correct_labels_by_class = mat.sum(axis=sum_axis)
        accuracy_by_class = np.divide(
            mat.diagnal(),
            num_correct_labels_by_class,
            out=np.zeros_like(mat.diagonal(),dtype=float),
            where=num_correct_labels_by_class != 0,
        )

        return pd.Series(accuracy_by_class, index=self.label_names)
    
    def find_metrics_by_period(
            self, label_data: pd.DataFrame, metric: ScoringFunction, resampling_code: str) -> pd.DataFrame:
        if metric == recall_score:
            sum_axis = 1
        elif metric == precision_score:
            sum_axis = 0
       
        metric_by_period = label_data.resample(resampling_code, on=self.prediction_time_column_name).apply(
            lambda x: self.metric_helper(x[self.true_label_name], x[self.predicted_label_name], sum_axis)
        )
       
        metric_by_period = metric_by_period.dropna(how="all")
        return metric_by_period
    
    def save_reliability_plots(self, data: pd.DataFrame):

        fig, ax = plt.subplots(1, len(self.labels), figsize=(15,6), facecolor="w", edgecolor="k",sharey="all")
        for i, label_name in enumerate(self.label_names):
            label_for_plot = np.where(data[self.true_label_name] == self.labels[i], 1, 0)
            probability = data[self.probability_column_names[i]]

            fop, mpv = calibration_curve(label_for_plot, probability, n_bins=20)

            ax[i].plot([0, 1], [0, 1], linestyle="--")
            ax[i].plot(mpv, fop, marker=".", linewidth=2)
            ax[i].set_title(f"class {label_name} vs Rest", fontsize=15)
            ax[i].set_xlabel("Predicted Probability", fontsize=12)

        ax[0].set_ylabel("Observed Relative Frequency", fontsize=12)

        plt.savefig(self.metrics_save_path / "reliability_plots.png", bbox_inches="tight")
        plt.close()
    
    def save_certainty_histogram(self, data: pd.DataFrame):

        num_classes = len(self.labels)
        fig, ax = plt.subplots(num_classes, 1, figsize=(15, 15), facecolor="w", edgecolor="k", sharex="all")
        min_win_prob = round(1 / num_classes, 1)
        bins = np.array(range(int(min_win_prob * 100), 105, 5)) / 100

        for i, label in enumerate(self.labels):
            win_probabilities = data.loc[data[self.true_label_name] == label, self.probability_column_names].max(axis=1)
            ax[i].hist(win_probabilities, edgecolor="black", bins=bins)
            ax[i].set_title(self.label_names[i])
            ax[i].set_ylabel("Population", fontsize=15)
        ax[num_classes - 1].set_xlabel("Predicted Probability")
        ax[num_classes - 1].set_xticks(bins)
        plt.xticks(fontsize=12)

        plt.savefig(self.metrics_save_path / "certainty_histogram.png")
        plt.close()

    def save_label_population_over_time(self, data: pd.DataFrame):

        data = data[[self.prediction_time_column_name, self.true_label_name, self.predicted_label_name]].melt(
            id_vars=[self.prediction_time_column_name], var_name="label_type", value_name="applied_label"
        )
        original_labels = data.loc[data["label_type"] == self.ture_label_name, "applied_label"].unique()
        predicted_labels = data.loc[data["label_type"] == self.predicted_label_name, "applied_label"].unique()
        labels = [original_labels, predicted_labels]

        for freq_name, freq_code in self.resampling_to_frequency_code.items():
            if freq_code == "M":
                data["frequency"] = (
                    data[self.prediction_time_column_name].dt.year.astype(str)
                    + data[self.prediction_time_column_name].dt.month_name()
                )
            else:
                data["frequency"] = data[self.prediction_time_column_name].dt.year
            
            transformed_label_data = (
                data.groupby(["frequency", "label_type", "applied_label"])[self.prediction_time_column_name]
                .count()
                .unstack([1, 2])
                .fillna(0)
            )
            fig, ax = plt.subplots(2, 1, figsize=(15, 15), facecolor="w", edgecolor="k", sharex="col")

            for ind, label_type in enumerate([self.true_label_name, self.predicted_label_name]):
                transformed_label_data[label_type][labels[ind]].rename(dict(zip(self.labels, self.label_names))).plot(
                    ax=ax[ind],
                )
                ax[ind].set_ylabel(f"Population of {label_type.title()}", fontsize=15)
                ax[ind].legend()
                plt.xticks(fontsize=12)
                plt.yticks(fontsize=12)

            plt.savefig(self.metrics_save_path / f"label_population_over_time_{freq_name}.png", bbox_inches="tight")
            plt.close()
    
    def perform_model_checks(self, data: pd.DataFrame):
        self.save_metrics_over_time_plot(data, metric_dict={"accuracy": recall_score, "precision": precision_score})
        self.save_reliability_plots(data)
        self.save_certainty_histogram(data)
        self.save_label_population_over_time(data)
        self.generate_and_save_ml_artifacts(data[self.true_label_name], data[self.predicted_label_name])
    

class TickFormatter:
    def __init__(self, all_dates: np.ndarray):
        self.data = all_dates

    def format_func(self, value, tick_number):
        if 0 <= int(value) < len(self.data):
            return np.datetime_as_string(self.data[int(value)], unit="D")
        else:
            return ""

