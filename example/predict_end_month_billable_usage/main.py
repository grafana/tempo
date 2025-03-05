from hydra.utils import instantiate
from omegaconf import DictConfig, OmegaConf
import hydra


@hydra.main(version_base="1.3", config_path="conf", config_name="config")
def main(hydra_config: DictConfig):
    print(OmegaConf.to_yaml(hydra_config, resolve=True))
    search_cv_factory = instantiate(hydra_config.search_cv.object, _partial_=True)
    cv_splitter = instantiate(hydra_config.cv.object)
    prediction_model = instantiate(
        hydra_config.model.object,
        search_cv_factory=search_cv_factory,
        cv_splitter=cv_splitter,
    )
    print(prediction_model)


if __name__ == "__main__":
    main()
