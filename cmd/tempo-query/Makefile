.PHONY: jaeger

jaeger:
	docker run --rm \
		-v ./jaeger:/etc/jaeger \
		jaegertracing/jaeger:2.6.0 \
		--config=file:/etc/jaeger/config.yaml
