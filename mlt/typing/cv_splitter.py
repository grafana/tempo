from typing import Protocol, Union, Tuple, Optional, Iterable
from abc import abstractmethod

import numpy as np
import pandas as pd


ArrayLike = Union[pd.DataFrame, np.ndarray, pd.Series]

class CVSplitter(Protocol):
    """
    A protocol for cross-validation splitters.
    """
    @abstractmethod
    def split(self, features: ArrayLike, labels: Optional[ArrayLike] = None) -> Iterable[Tuple[np.ndarray, np.ndarray]]:
        """
        Split the data into training and validation sets.
        """
        ...
        
    @abstractmethod
    def get_n_splits(self) -> int:
        """
        Get the number of splits for the cross-validation.
        """
        ...
