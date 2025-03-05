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



def get_data(hydra_config) -> Tuple[pd.DataFrame, Dict[str, Any]]:
    """
    Generate a new labeled data set with a new featurizer
    :return: return DataFrame of the labeled data set
    """
    tik = time.perf_counter()
    logging.info("Loading Dataset")

    if hydra_config["run_settings"]["use_local_features"]:
        data_tags = {
            "DATASET_NAME": hydra_config.data_acquisition_config.joined_data_path,
            "DATASET_ID": "local_data_set",
            "DATASET_VERSION": "local_dataset",
        }
        processed_data = pd.read_parquet(
            Path(hydra_config.data_acquisition_config.data_path)
            / Path(hydra_config.data_acquisition_config.joined_data_path)
        )
    
    logging.info(f"Dataset Loaded in {time.perf_counter() - tik} Seconds")

    return processed_data, data_tags


def update_experiment_tags(
    experiment_tags: Dict[str, Any],
    hydra_config: DictConfig,
    training_data: pd.DataFrame,
    prediction_date_manager: PredictionWindowManager,
) -> Dict[str, Any]:
    """
    Update the experiment tags with information from the configuration, training data and prediction date manager

    :param experiment_tags: the existing experiment tags
    :param hydra_config: configuration for the experiment
    :param training_data: the training data
    :param prediction_date_manager: the prediction date manager
    :return: the updated experiment tags
    """
    column_definitions = hydra_config.column_definitions
    experiment_tags.update(
        {
            "multi_org": hydra_config.run_settings.multi_org,
            "target": hydra_config.target,
            "experiment_name": hydra_config.experiment_name,
            "prediction_dates": (
                training_data[column_definitions.prediction_time_column_name].min().strftime("%Y-%m-%d"),
                training_data[column_definitions.prediction_time_column_name].max().strftime("%Y-%m-%d"),
            ),
            "evaluation_dates": (
                training_data[column_definitions.evaluation_time_column_name].min().strftime("%Y-%m-%d"),
                training_data[column_definitions.evaluation_time_column_name].max().strftime("%Y-%m-%d"),
            ),
            "cv_type": hydra_config.cv.name,
            "search_cv_type": hydra_config.search_cv.name,
            "model_type": hydra_config.model.name,
            "initial_train_date_number": prediction_date_manager.initial_train_size,
            "number_of_training_periods_between_tuning": prediction_date_manager.num_bars_between_training,
            "sequence_length": hydra_config.model.get("sequence_length", default_value=None),
            "rolling_window_size": prediction_date_manager.rolling_window_size,
        }
    )
    
    return experiment_tags


def run(hydra_config: DictConfig):
    """
    Run the experiment
    :param hydra_config: configuration for the experiment
    """
    if not hydra_config.run_settings.multi_org and hydra_config.column_definitions.org_id_column_name is not None:
        raise ValueError("Single Asset Mode, org_id_column_name should be set to null in the config")

    run_settings = hydra_config.run_settings
    outer_cv_settings = hydra_config.outer_cv_settings
    column_definitions = hydra_config.column_definitions
    # optional field for tagging experiments
    experiment_tags = {}

    experiment_name = hydra_config.experiment_name
    experiment_path = Path(hydra_config.experiment_path) / Path(hydra_config.run_name)
    # Hydra will create the folders automatically
    if not GlobalHydra().is_initialized():
        experiment_path.mkdir(parents=True)
    (experiment_path / Path("ml_metrics")).mkdir(exist_ok=True)

    with open(experiment_path / "config.yaml", "w") as f:
        OmegaConf.save(hydra_config, resolve=True, f=f)

    logging.root.handlers = []
    # noinspection PyArgumentList
    logging.basicConfig(
        level=logging.INFO,
        format="[%(levelname)s] %(asctime)s - %(name)s - %(funcName)s: l%(lineno)d: %(message)s",
        handlers=[
            logging.FileHandler(experiment_path / Path("experiment.log")),
            logging.StreamHandler(),
        ],
    )

    # prepare the data
    data, data_tags = get_data(hydra_config)
    experiment_tags.update(data_tags)
    data_for_training = data.dropna(subset=[column_definitions.true_label_name])
    data_for_training = data_for_training.loc[
        data_for_training[column_definitions.prediction_time_column_name].between(
            run_settings.start_date, run_settings.end_date
        )
    ]
    prediction_cutoff = np.sort(data[column_definitions.prediction_time_column_name].unique())[
        -hydra_config.outer_cv_settings.num_recent_prediction_days
    ]
    data_for_prediction = data.loc[(data[column_definitions.prediction_time_column_name] >= prediction_cutoff)]

    if run_settings.multi_org and column_definitions.org_id_column_name not in data_for_training.columns:
        raise KeyError(
            f"{hydra_config.org_id_column_name} not in the multi-org data. "
            "Please check the training data and the name of the id column"
        )

    # instantiate the required objects
    search_cv_factory = instantiate(hydra_config.search_cv.object, _partial_=True)
    cv_splitter = instantiate(hydra_config.cv.object)
    prediction_model = instantiate(
        hydra_config.model.object,
        search_cv_factory=search_cv_factory,
        cv_splitter=cv_splitter,
    )

    prediction_date_manager = PredictionWindowManager(
        gap_size=outer_cv_settings.train_test_gap,
        initial_train_size=outer_cv_settings.initial_train_date_number,
        training_frequency=outer_cv_settings.training_frequency,
        num_bars_between_training=outer_cv_settings.number_of_dates_between_training,
        time_column_name=column_definitions.time_column_name,
        rolling_window_size=outer_cv_settings.rolling_window_size,
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
    experiment_tags = update_experiment_tags(
        experiment_tags=experiment_tags,
        hydra_config=hydra_config,
        training_data=data_for_training,
        prediction_date_manager=prediction_date_manager,
    )

    # run the experiment
    training_pipeline.generate_predictions_for_prediction_period(data_for_training)
    all_results = training_pipeline.all_results_
    last_results = training_pipeline.last_results_
    prediction_model = training_pipeline.prediction_model
    experiment_logger = instantiate(
        hydra_config.experiment_logger.object,
        experiment_path=experiment_path,
        prediction_columns_to_save=[col for col in column_definitions.id_columns if col is not None]
        + [column_definitions.true_label_name, column_definitions.predicted_label_name],
        prediction_time_column_name=column_definitions.prediction_time_column_name,
        true_label_name=column_definitions.true_label_name,
        predicted_label_name=column_definitions.predicted_label_name,
    )
    metrics = experiment_logger.get_ml_metrics(
        training_pipeline.all_results_.out_of_sample_predictions[column_definitions.true_label_name],
        training_pipeline.all_results_.out_of_sample_predictions[column_definitions.predicted_label_name],
    )
    experiment_logger.default_save_experiment_artifacts_to_disk(
        prediction_model=prediction_model,
        all_results=all_results,
        last_results=last_results,
        model_name="mlt_model",
        run_config=hydra_config,
    )
    unlabelled_predictions = prediction_model.get_predictions(data_for_prediction)
    if prediction_model.has_shap:
        unlabelled_shaps = prediction_model.get_shap_values(
            feature_data=data_for_prediction, predictions=unlabelled_predictions
        )
    else:
        unlabelled_shaps = None

    experiment_logger.save_unlabelled_prediction_artifacts_to_disk(unlabelled_predictions, unlabelled_shaps)



@hydra.main(version_base="1.3", config_path="conf", config_name="config")
def main(cfg: DictConfig):
    print("Uninstantiated Hydra Config:")
    print(OmegaConf.to_yaml(cfg, resolve=True))

    if cfg.run_settings.run:
        start = time.perf_counter()
        run(cfg)
        print(time.perf_counter() - start)
    else:
        logging.warning("Run the model is set to False in the modeling config. Not Running the pipeline")


if __name__ == "__main__":
    main()
