try:
    import xgboost_models
except ImportError:
    raise ImportError("XGBoost is not installed. Please install it with `pip install xgboost`.")

from mlt.models.xgboost_models.regression import MLTXGBoostRegression




