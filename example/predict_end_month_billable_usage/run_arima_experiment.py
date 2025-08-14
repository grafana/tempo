import logging
from pathlib import Path
import time
from typing import Any, Dict, Tuple
import hydra
from hydra.core.global_hydra import GlobalHydra
import numpy as np
import pandas as pd
from hydra.utils import instantiate
from omegaconf import DictConfig, OmegaConf

from mlt.cross_validation_splitters.prediction_window_manager import PredictionWindowManager
from mlt.experiment_pipeline import TrainingPipeline
from mlt.models.time_series_models.arima import MLTArimaModel

def get_data(hydra_config) -> Tuple[pd.DataFrame, Dict[str, Any]]:
    """
    Load the dataset for ARIMA training.
    """
    tik = time.perf_counter()
    logging.info("Loading Dataset")

    data_path = Path(hydra_config.data_acquisition_config.data_path)
    file_name = hydra_config.data_acquisition_config.joined_data_path
    processed_data = pd.read_parquet(data_path / file_name)
    
    logging.info(f"Dataset Loaded in {time.perf_counter() - tik} Seconds")

    return processed_data, {"DATASET_NAME": file_name}

def run(hydra_config: DictConfig):
    """
    Run the ARIMA experiment.
    """
    run_settings = hydra_config.run_settings
    outer_cv_settings = hydra_config.outer_cv_settings
    column_definitions = hydra_config.column_definitions
    experiment_name = hydra_config.experiment_name
    experiment_path = Path(hydra_config.experiment_path) / Path(hydra_config.run_name)
    
    if not GlobalHydra().is_initialized():
        experiment_path.mkdir(parents=True)
    (experiment_path / "ml_metrics").mkdir(exist_ok=True)

    with open(experiment_path / "config.yaml", "w") as f:
        OmegaConf.save(hydra_config, resolve=True, f=f)

    logging.basicConfig(
        level=logging.INFO,
        format="[%(levelname)s] %(asctime)s - %(name)s - %(funcName)s: %(message)s",
        handlers=[logging.FileHandler(experiment_path / "experiment.log"), logging.StreamHandler()],
    )
    
    data, data_tags = get_data(hydra_config)
    data_for_training = data.dropna(subset=[column_definitions.true_label_name])
    
    prediction_date_manager = PredictionWindowManager(
        gap_size=outer_cv_settings.train_test_gap,
        initial_train_size=outer_cv_settings.initial_train_date_number,
        training_frequency=outer_cv_settings.training_frequency,
        num_bars_between_training=outer_cv_settings.number_of_dates_between_training,
        time_column_name=column_definitions.time_column_name,
        rolling_window_size=outer_cv_settings.rolling_window_size,
    )
    
    if hydra_config.model.name == "arima_regressor":
        prediction_model = instantiate(
            hydra_config.model.object,
            non_feature_columns=hydra_config.column_definitions.non_feature_columns,
            model_type=hydra_config.model_type.name,
            model_name=hydra_config.model.name,
            model_parameter_grid=hydra_config.model.object.model_parameter_grid,
            prediction_time_column_name=hydra_config.column_definitions.prediction_time_column_name,
            true_label_name=hydra_config.column_definitions.true_label_name,
            predicted_label_name=hydra_config.column_definitions.predicted_label_name,
        )
    else:
        prediction_model = instantiate(
            hydra_config.model.object,
            search_cv_factory=search_cv_factory,
            cv_splitter=cv_splitter,
        )

    training_pipeline = TrainingPipeline(
        prediction_model=prediction_model,
        prediction_date_manager=prediction_date_manager,
        tuning_frequency=outer_cv_settings.number_of_training_periods_between_tuning,
        min_shap_date=outer_cv_settings.min_shap_date,
        prediction_time_column_name=column_definitions.prediction_time_column_name,
        num_outer_cv_jobs=outer_cv_settings.n_outer_jobs,
        is_smoke_test=run_settings.smoke_test,
    )
    
    # Run the ARIMA model experiment
    training_pipeline.generate_predictions_for_prediction_period(data_for_training)
    
   # Save results
    all_results = training_pipeline.all_results_
    last_results = training_pipeline.last_results_
    prediction_model = training_pipeline.prediction_model

    # Instantiate Experiment Logger
    experiment_logger = instantiate(
        hydra_config.experiment_logger.object,
        experiment_path=experiment_path,
        prediction_columns_to_save=[
            col for col in column_definitions.id_columns if col is not None
        ] + [column_definitions.true_label_name, column_definitions.predicted_label_name],
        prediction_time_column_name=column_definitions.prediction_time_column_name,
        true_label_name=column_definitions.true_label_name,
        predicted_label_name=column_definitions.predicted_label_name,
    )

    # Compute Metrics
    metrics = experiment_logger.get_ml_metrics(
        training_pipeline.all_results_.out_of_sample_predictions[column_definitions.true_label_name],
        training_pipeline.all_results_.out_of_sample_predictions[column_definitions.predicted_label_name],
    )

    experiment_logger = instantiate(
        hydra_config.experiment_logger.object,
        experiment_path=experiment_path,
        prediction_columns_to_save=[col for col in column_definitions.id_columns if col is not None]
        + [column_definitions.true_label_name, column_definitions.predicted_label_name],
        prediction_time_column_name=column_definitions.prediction_time_column_name,
        true_label_name=column_definitions.true_label_name,
        predicted_label_name=column_definitions.predicted_label_name,
    )

    # Save model artifacts
    model_folder = experiment_path / "saved_models"
    model_folder.mkdir(exist_ok=True)
    prediction_model.to_pickle(model_folder / "mlt_arima_model.pkl")

    # Save predictions to a CSV file for plotting
    prediction_output_path = experiment_path / "predictions.csv"

    # Save Out-of-Sample Predictions (Real Forecasts)
    #all_results.out_of_sample_predictions.to_csv(prediction_output_path, index=False)
    # Save predictions for later analysis
    all_results.out_of_sample_predictions.to_csv(experiment_path / "out_of_sample_predictions.csv", index=False)
    all_results.in_sample_predictions.to_csv(experiment_path / "in_sample_predictions.csv", index=False)

    # Save In-Sample Predictions (Model Fit Evaluation)
    #in_sample_output_path = experiment_path / "in_sample_predictions.csv"
    #all_results.in_sample_predictions.to_csv(in_sample_output_path, index=False)

    # Log the save locations
    logging.info(f"Saved out-of-sample predictions to {prediction_output_path}")
    logging.info(f"Saved in-sample predictions to {in_sample_output_path}")

    # Save predictions
    #unlabelled_predictions = prediction_model.get_predictions(data)
    #unlabelled_predictions.to_pickle(experiment_path / "historicalpredictions.pkl")
    forecast_steps = 10  # Define how many steps ahead to forecast
    forecast = prediction_model.forecast(steps=forecast_steps)
    print(f"\nForecast for next {forecast_steps} periods:")
    print(forecast)

    # Save model parameters
    if all_results.model_params is not None:
        all_results.model_params.to_csv(experiment_path / "model_params_history.csv", index=False)

    # Save SHAP values (if available)
    if all_results.shaps is not None:
        all_results.shaps.to_parquet(experiment_path / "all_shaps.parquet")

    logging.info("Experiment Completed Successfully and Results Saved.")

@hydra.main(version_base="1.3", config_path="conf", config_name="config")
def main(cfg: DictConfig):
    print("Hydra Config:")
    print(OmegaConf.to_yaml(cfg, resolve=True))

    if cfg.run_settings.run:
        start = time.perf_counter()
        run(cfg)
        print(time.perf_counter() - start)
    else:
        logging.warning("Run setting is False. Not executing the pipeline.")

if __name__ == "__main__":
    main()
