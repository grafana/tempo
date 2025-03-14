import numpy as np
import pandas as pd
from typing import Tuple, Optional, Iterable
from mlt.typing.cv_splitter import CVSplitter


class PredictionWindowManager(CVSplitter):
    """
    A class for managing prediction windows in time series data.

    This class provides functionality to split data into training and prediction windows,
    and to generate predictions for each window.

    """
    def __init__(
            self,
            gap_size: int,
            initial_train_size: int,
            training_frequency: str,
            num_bars_between_training: int,
            time_column_name: str,
            rolling_window_size: int,
    ):
        """
        Initialize the PredictionWindowManager.

        Args:
            gap_size: Number of periods between training and prediction windows
            initial_train_size: Number of periods in initial training window
            training_frequency: Frequency of retraining ('daily', 'weekly', etc)
            num_bars_between_training: Number of bars between retraining periods
            time_column_name: Name of timestamp column in data
            rolling_window_size: Size of rolling window for training data

        The PredictionWindowManager handles splitting time series data into training
        and prediction windows while maintaining proper temporal ordering and gaps.
        It ensures:
        - No data leakage between training and prediction periods
        - Consistent gap sizes between training and prediction
        - Regular retraining intervals
        - Rolling window approach for training data
        """

        self.gap_size = gap_size
        self.initial_train_size = initial_train_size
        self.training_frequency = training_frequency
        self.num_bars_between_training = num_bars_between_training
        self.scale_sizes()
        self.time_column_name = time_column_name
        self.rolling_window_size = rolling_window_size

    def get_n_splits(self) -> int:
        pass

    def scale_sizes(self):
        """
        Scale the sizes of the training and prediction windows based on the training frequency.
        """
        frequency_map = {
            "daily": 1,
            "weekly": 5,
            "monthly": 22,
        }

        if self.training_frequency not in frequency_map:
            available = ", ".join(frequency_map.keys())
            raise ValueError(
                f"Unsupported frequency '{self.training_frequency}'. " 
                f"Supported frequencies: {available}. "
            )
        denominator = frequency_map[self.training_frequency]

        self.gap_size = int(np.ceil(self.gap_size / denominator))
        self.initial_train_size = int(np.ceil(self.initial_train_size / denominator))
        self.num_bars_between_training = int(np.ceil(self.num_bars_between_training / denominator))

    def split(self, data: pd.DataFrame, labels: Optional[pd.Series] = None) -> Iterable[Tuple[np.ndarray, np.ndarray]]:

        if not data[self.time_column_name].is_monotonic_increasing:
            raise ValueError("Time column must be monotonic increasing")
        prediction_times = data[self.time_column_name].unique()
        num_prediction_times = len(prediction_times)
        indices = np.arange(num_prediction_times)

        if self.initial_train_size > num_prediction_times:
            raise ValueError(
                f"Initial training size must be less than or equal to the number of prediction times. \n"
                f"number of prediction times: {num_prediction_times}\n"
                f"initial_train_size: {self.initial_train_size}"
                )
        prediction_window_starts = range(
            self.initial_train_size + self.gap_size, num_prediction_times, self.num_bars_between_training
        )

        for prediction_start in prediction_window_starts:
            train_end = prediction_start - self.gap_size

            if self.rolling_window_size:
                train_times = prediction_times[indices[max(train_end - self.rolling_window_size, 0):train_end]]
            else:
                train_times = prediction_times[indices[:train_end]]

            test_times = prediction_times[indices[prediction_start : prediction_start + self.num_bars_between_training]]
            yield(
                data.loc[data[self.time_column_name].isin(train_times)].index,
                data.loc[data[self.time_column_name].isin(test_times)].index,
            )
