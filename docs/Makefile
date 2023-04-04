IMAGE = grafana/docs-base:latest

.PHONY: pull
pull:
	docker pull ${IMAGE}

.PHONY: docs
docs: pull
	docker pull ${IMAGE}
	docker run  --rm -it -v ${PWD}/sources/tempo:/hugo/content/docs/tempo/latest -p 3002:3002 $(IMAGE)

.PHONY: docs-test
docs-test: pull
	docker run --rm -it -v ${PWD}/sources/tempo:/hugo/content/docs/tempo/latest -p 3002:3002 $(IMAGE) /bin/bash -c 'make prod'
