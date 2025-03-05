import math
import numpy as np
import pandas as pd
import numbers

from abc import abstractmethod
from typing import Tuple, List, Iterable
from mlt.typing.cv_splitter import CVSplitter


class BaseTimeSeriesCV(CVSplitter):
    """
    Base class for time series cross-validation splitters.

    This class provides common functionality for time series cross-validation,
    including embargo periods, time-based splitting, and validation of inputs.
    It implements the CVSplitter protocol for consistent cross-validation behavior.

    Args:
        n_splits: Number of cross-validation splits (must be > 1)
        pred_time_column_name: Name of column containing prediction timestamps
        eval_time_column_name: Name of column containing evaluation timestamps
        embargo_td: Timedelta specifying embargo period between train/test sets
        split_by_time: Whether to split based on time rather than indices

    The class enforces:
    - Monotonically increasing prediction times
    - Proper DataFrame/Series input types
    - Matching indices between features and labels
    - Valid number of splits
    - Embargo periods between training and test sets

    Subclasses must implement:
    - get_n_splits(): Returns number of CV splits
    - _split(): Core splitting logic for the specific CV strategy
    """



    def __init__(
        self,
        n_splits: 10,
        pred_time_column_name: None,
        eval_time_column_name: None, 
        embargo_td: pd.Timedelta = pd.Timedelta(minutes=0),
        split_by_time: bool = False,
    ):
        
        
        if not isinstance(n_splits, numbers.Integral):
            raise ValueError(f"n_splits must be an integer, got {type(n_splits)}")
        n_splits = int(n_splits)
        if n_splits <= 1:
            raise ValueError(f"n_splits must be greater than 1, got {n_splits}")
        
        self.n_splits = n_splits
        self.pred_time_column_name = pred_time_column_name
        self.eval_time_column_name = eval_time_column_name
        self.embargo_td = embargo_td
        self.split_by_time = split_by_time

    def _check_split_arguments(self, X: pd.DataFrame, y: pd.Series = None):
        if not X[self.pred_time_column_name].is_monotonic_increasing:
            raise ValueError(f"pred_time_column_name must be monotonic increasing, got {X[self.pred_time_column_name].head()}")
        if not isinstance(X, pd.DataFrame) and not isinstance(X, pd.Series):
            raise ValueError(f"X must be a pandas DataFrame or Series, got {type(X)}")
        if not isinstance(y, pd.Series) and y is not None:
            raise ValueError(f"y must be a pandas Series, got {type(y)}")
        if y is not None and (X.index ==y.index).sum() != len(X):
            raise ValueError(f"X and y must have the same index")
        
    @abstractmethod
    def get_n_splits(self) -> int:
        ...

   
    def split(self, X: pd.DataFrame, y: pd.Series = None) -> Iterable[Tuple[np.ndarray, np.ndarray]]:
        """
        Split data into training and test sets.

        This method performs the core splitting logic by:
        1. Validating input data types and formats
        2. Extracting prediction and evaluation times
        3. Delegating to _split() for specific CV strategy implementation

        Args:
            X: Features DataFrame with prediction and evaluation time columns
            y: Optional target Series that matches X's index

        Returns:
            Iterable of (train_indices, test_indices) tuples containing array indices

        Raises:
            ValueError: If inputs don't meet validation requirements around types,
                      indices matching, or time column ordering
        """
        self._check_split_arguments(X, y)
        eval_times = X[self.eval_time_column_name]
        pred_times = X[self.pred_time_column_name]
        yield from self._split(X, y, pred_times, eval_times)

    @abstractmethod
    def _split(self, X: pd.DataFrame, y: pd.Series = None, pred_times: pd.Series = None, eval_times: pd.Series = None):
        ...

    def embargo(
            self, 
            all_indices : np.ndarray,
            train_indices : np.ndarray,
            test_indices : np.ndarray,
            test_fold_end : int,
            pred_times: pd.Series,
            eval_times: pd.Series,
    ) -> np.ndarray:
        

        """
        Embargo the test indices based on the embargo time delta.

        This method implements an embargo period after each test set to prevent data leakage.
        It removes training samples that occur within the embargo period after the test set.

        Args:
            all_indices: Array of all data indices
            train_indices: Array of training set indices
            test_indices: Array of test set indices 
            test_fold_end: Index marking the end of the test fold
            pred_times: Series of prediction times
            eval_times: Series of evaluation times

        Returns:
            Modified train_indices with samples in embargo period removed

        The embargo period helps prevent using training data that is too close in time
        to the test samples, which could lead to data leakage in financial applications.
        """
        ...
        
        last_test_eval_time = eval_times.iloc[test_indices[test_indices <= test_fold_end].max()]
        min_train_index = len(pred_times[pred_times <= last_test_eval_time + self.embargo_td])
        if min_train_index < all_indices.shape[0]:
            allowed_indices = np.concatenate([all_indices[:test_fold_end], all_indices[min_train_index:]])
            train_indices = np.intersect1d(train_indices, allowed_indices)
        return train_indices
    
    @staticmethod
    def purge(
        all_indices : np.ndarray,
        train_indices : np.ndarray,
        test_fold_start : int,
        test_fold_end : int,
        pred_times: pd.Series,
        eval_times: pd.Series,
    ) -> np.ndarray:
        """
        Purge training samples that overlap with the test period.

        This method removes training samples whose evaluation times overlap with the prediction
        period of test samples to prevent data leakage.

        Args:
            all_indices: Array of all data indices
            train_indices: Array of training set indices
            test_fold_start: Index marking the start of the test fold
            test_fold_end: Index marking the end of the test fold
            pred_times: Series of prediction times
            eval_times: Series of evaluation times

        Returns:
            Modified train_indices with overlapping samples removed

        The purging process helps ensure that training samples do not use information from
        time periods that overlap with the test set predictions, which is important for
        maintaining realistic backtesting results in financial applications.
        """
        time_test_fold_start = pred_times.iloc[test_fold_start]

        train_indices_1 = np.intersect1d(train_indices, all_indices[eval_times < time_test_fold_start])
        train_indices_2 = np.intersect1d(train_indices, all_indices[test_fold_end:])
        if test_fold_end < all_indices.shape[0]:
            eval_time_test_end = eval_times.iloc[test_fold_end]
            train_indices_2 = np.intersect1d(train_indices_2, all_indices[pred_times >= eval_time_test_end])
        return np.concatenate([train_indices_1, train_indices_2])
    


class PurgedWalkForwardCV(BaseTimeSeriesCV):

    """
    Purged walk-forward cross validation splitter for time series data.

    This class implements walk-forward cross validation with purging and embargo to prevent data leakage
    in financial time series applications. It extends BaseTimeSeriesCV to provide:

    - Walk-forward splitting: Data is split sequentially to maintain temporal ordering
     - Purging: Removes training samples that overlap with test periods
    - Configurable train/test splits: Allows controlling size of training and test windows
    - Time-based or index-based splitting options

     Parameters
       ----------
    n_splits : int
        Total number of splits in the dataset
    pred_time_column_name : str
        Name of column containing prediction timestamps
    eval_time_column_name : str  
        Name of column containing evaluation timestamps
    n_test_splits : int, default=1
        Number of splits to use for each test set
    min_train_splits : int, default=2
        Minimum number of splits required for training
    max_train_splits : int, default=None
        Maximum number of splits to use for training. If None, uses n_splits - n_test_splits
    split_by_time : bool, default=False
        If True, splits are determined by time periods rather than indices

    The splitter ensures that:
    - Training data comes strictly before test data
    - Overlapping samples between train and test are properly purged
    - Training window size can grow up to max_train_splits
    - Test window size remains constant at n_test_splits
    """

    def __init__(
        self,
        n_splits: int,
        pred_time_column_name: str,
        eval_time_column_name: str,
        n_test_splits: int = 1,
        min_train_splits: int =2,
        max_train_splits: int = None, 
        split_by_time: bool = False, 
    ):
            
        super().__init__(n_splits, pred_time_column_name, eval_time_column_name, split_by_time = split_by_time)
        if not isinstance(n_test_splits, numbers.Integral):
            raise ValueError(f"n_test_splits must be an integer, got {type(n_test_splits)}")
           
        n_test_splits = int(n_test_splits)
        if n_test_splits <= 0 or n_test_splits >= self.n_splits - 1:
            raise ValueError(f"n_test_splits must be greater than 0 and less than n_splits - 1, got {n_test_splits}")
        self.n_test_splits = n_test_splits

        if not isinstance(min_train_splits, numbers.Integral):
            raise ValueError(f"min_train_splits must be an integer, got {type(min_train_splits)}")
        min_train_splits = int(min_train_splits)
        if min_train_splits <= 0 or min_train_splits >= self.n_splits - self.n_test_splits:
            raise ValueError(f"min_train_splits must be greater than 0 and less than n_splits - n_test_splits, got {min_train_splits}")
        self.min_train_splits = min_train_splits

        if max_train_splits is None:
            max_train_splits = self.n_splits - self.n_test_splits
        if not isinstance(max_train_splits, numbers.Integral):
            raise ValueError(f"max_train_splits must be an integer, got {type(max_train_splits)}")
        max_train_splits = int(max_train_splits)
        if max_train_splits <= 0 or max_train_splits > self.n_splits - self.n_test_splits:
            raise ValueError(f"max_train_splits must be greater than 0 and less than n_splits - n_test_splits, got {max_train_splits}")
        if self.min_train_splits > max_train_splits:
            raise ValueError(f"min_train_splits must be less than or equal to max_train_splits, got {self.min_train_splits} and {max_train_splits}")
        self.max_train_splits = max_train_splits

    def get_n_splits(self) -> int:
        return self.n_splits - self.n_test_splits - self.min_train_splits +1 
        
    def _split(
            self, X: pd.DataFrame, y: pd.Series = None, pred_times: pd.Series = None, eval_times: pd.Series = None
    ) -> Iterable[Tuple[np.ndarray, np.ndarray]]:
        """Split the data into training and test sets.

        This method generates training and test splits according to the PurgedWalkForward CV scheme:
        - Training data comes strictly before test data
        - Training window size grows from min_train_splits up to max_train_splits
        - Test window size remains constant at n_test_splits
        - Overlapping samples between train and test are purged based on prediction and evaluation times

        Parameters
        ----------
        X : pd.DataFrame
            The input data to split
        y : pd.Series, optional
            The target variable, not used in the split
        pred_times : pd.Series, optional
            Series containing prediction timestamps for each sample
        eval_times : pd.Series, optional
            Series containing evaluation timestamps for each sample

        Yields
        ------
        train_indices : np.ndarray
            Indices of training data for current split
        test_indices : np.ndarray
            Indices of test data for current split
        """
        indices = np.arange(X.shape[0])
        fold_bounds = self._compute_fold_bounds(X)
        for i in range(self.min_train_splits, self.n_splits - self.n_test_splits + 1):
            train_start = fold_bounds[max(0, i - self.max_train_splits)]
            train_end = fold_bounds[i]
            train_indices = indices[slice(train_start, train_end)]

            if (test_end_bound := i + self.n_test_splits) < self.n_splits:
                test_end = fold_bounds[test_end_bound]
            else:
                #this is the last split, use all remaining data for testing
                test_end = None
            test_indices = indices[slice(fold_bounds[i], test_end)]
            purged_train_indices = self.purge(
                indices, train_indices, test_indices.min(), test_indices.max(), pred_times, eval_times
            )
            yield purged_train_indices, test_indices
    
    def _compute_fold_bounds(self, X: pd.DataFrame) -> List[int]:
        """
        Compute the bounds of each fold in the dataset.

        This method calculates the indices where each fold starts and ends based on the
        prediction and evaluation timestamps. It ensures that each fold is properly
            
        """
        if self.split_by_time:
            pred_times  = X[self.pred_time_column_name]
            full_time_span = pred_times.max() - pred_times.min()
            
            fold_time_span = full_time_span / self.n_splits
            fold_bounds_times = [pred_times.iloc[0] + fold_time_span * n for n in range(self.n_splits)]
            return pred_times.searchsorted(fold_bounds_times)
        else:
            return [fold[0] for fold in np.array_split(np.arange(X.shape[0]), self.n_splits)]
        
# add purged and embargoed combinatorial cross validation in the future 

            
            

           
           
           
           

           
    
    
        

