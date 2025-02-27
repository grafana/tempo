import pandas as pd
import numpy as np
from typing import List, Dict
from dataclasses import dataclass


@dataclass
class ShapValues:
    """
    A data class to store all the information relevant to shap values
    scaled_shap_values: dict that maps classes to array of shap values (n_samples * n_features)
    feature_names: the list of feature names
    metadata: dataframe of metadata
    date_col: the name of the date column in the features dataframe. Defaults to "prediction_date"
    predicted_label_name: name of the column to put the label predictions in
    """
    scaled_shap_values: Dict[str, np.array]
    feature_names: List[str]
    metadata: pd.DataFrame
    date_col: str
    predicted_label_name: str


    def to_dataframe(self) -> pd.DataFrame:
        """
        Convert shap values from dict of arrays to pandas dataframe to easily save on disk or
        push to the database.
        :return: dataframe of shap values of size (n_classes * n_samples) * n_features
        """

        metadata_df = self.metadata[[self.date_col]].reset_index(drop=True)
        all_shap_values = pd.DataFrame(columns=self.feature_names)
        for cl in self.scaled_shap_values.keys():
            class_shap_values = pd.DataFrame(self.scaled_shap_values[cl], columns=self.feature_names)
            class_shap_values[self.predicted_label_name] = cl
            class_shap_values = pd.concat([metadata_df, class_shap_values], axis=1)
            all_shap_values = pd.concat([all_shap_values, class_shap_values], ignore_index=True)
        all_shap_values[self.predicted_label_name] = all_shap_values[self.predicted_label_name].astype("float")
        return all_shap_values
    

def scale_shap_values(shap_values: np.array, expected_value: float, model_prediction_prob: float) -> np.array:
    """
    Scale shap values for each prediction in one predicted class. Shap values are originally in log-odd space.
    Transform them into probability space.
    Scale the given shap values such that their sum is equal to abs(model_prediction_prob - expected_value)
    Similar to what is suggested here https://github.com/slundberg/shap/issues/29#issuecomment-374928027
    :param shap_values: array of shap values for one class (of size #features * #predictions) in log-odd space
    :param expected_value: expected value in probability space
    :param model_prediction_prob: prediction probability
    :return: shap values in probability space
    """
    # Compute distance between actual output and expected value in log_odd space
    log_odd_distance = shap_values.sum(axis=1)
    # Compute the distance between the model_prediction and the transformed expected value
    prob_distance = model_prediction_prob - expected_value
    # Convert the original shap values to the new scale
    shap_values_transformed = (shap_values * prob_distance.reshape(-1, 1)) / log_odd_distance.reshape(-1, 1)

    return shap_values_transformed


def scale_all_shap_values(
    expected_values: Dict[str, float], shap_values: Dict[str, np.array], model_prediction_probs: pd.DataFrame
) -> Dict[str, np.array]:
    """
    Scale shap values for all the predictions. See scale_shap_values for more detail on scaling.
    :param expected_values: dict that maps classes to expected values for that class
    :param shap_values: dict that maps classes to array of shap values (n_samples * n_features)
    :param model_prediction_probs: dataframe of predictions of size n_sample * n_classes
    :return: dict that maps classes to array of scaled shap values (n_samples * n_features)
    """
    
    all_shap_values = {}
    for cl, values in shap_values.items():
        all_shap_values[cl] = scale_shap_values(values, expected_values[cl], model_prediction_probs[cl].to_numpy())

    return all_shap_values