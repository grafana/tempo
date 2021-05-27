.PHONY: docs
docs:
	docker pull grafana/grafana-docs-dev:latest
	docker run  --rm -it -v ${PWD}/tempo/website:/hugo/content/docs/tempo/latest -p 3002:3002 grafana/grafana-docs-dev:latest /bin/bash -c 'make server-quick'

.PHONY: docs-test
docs-test:
	docker pull grafana/docs-base:latest
	docker run --rm -it -v ${PWD}/tempo/website:/hugo/content/docs/tempo/latest -p 3002:3002 grafana/docs-base:latest /bin/bash -c 'make prod'
