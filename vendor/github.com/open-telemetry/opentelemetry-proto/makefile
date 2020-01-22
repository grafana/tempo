GENDIR := gen
OPENAPI_OUTDIR := "./$(GENDIR)/openapi"

# Find all .proto files.
PROTO_FILES := $(wildcard opentelemetry/proto/*/v1/*.proto opentelemetry/proto/collector/*/v1/*.proto)

# Function to execute a command. Note the empty line before endef to make sure each command
# gets executed separately instead of concatenated with previous one.
# Accepts command to execute as first parameter.
define exec-command
$(1)

endef

# CI build
.PHONY: ci
ci: gen-java gen-swagger

# Generate ProtoBuf implementation for Go.
.PHONY: gen-go
gen-go:
	$(foreach file,$(PROTO_FILES),$(call exec-command,protoc --go_out=plugins=grpc:$(GOPATH)/src $(file)))
	rm -rf ./$(GENDIR)/go
	cp -R $(GOPATH)/src/github.com/open-telemetry/opentelemetry-proto/$(GENDIR)/go ./gen/

# Generate ProtoBuf implementation for Java.
.PHONY: gen-java
gen-java:
	rm -rf ./$(GENDIR)/java
	mkdir -p ./$(GENDIR)/java
	$(foreach file,$(PROTO_FILES),$(call exec-command, protoc --java_out=./$(GENDIR)/java $(file)))

# Generate Swagger
.PHONY: gen-swagger
gen-swagger:
	mkdir -p $(OPENAPI_OUTDIR)
	protoc --plugin=protoc-gen-swagger=/usr/bin/protoc-gen-swagger --swagger_out=logtostderr=true,grpc_api_configuration=opentelemetry/proto/collector/trace/v1/trace_service_http.yaml:$(OPENAPI_OUTDIR) opentelemetry/proto/collector/trace/v1/trace_service.proto
