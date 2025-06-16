module github.com/grafana/tempo

go 1.24.3

require (
	cloud.google.com/go/storage v1.55.0
	github.com/alecthomas/kong v1.11.0
	github.com/alicebob/miniredis/v2 v2.35.0
	github.com/aws/aws-sdk-go v1.55.7
	github.com/cespare/xxhash/v2 v2.3.0
	github.com/cristalhq/hedgedhttp v0.9.1
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc
	github.com/drone/envsubst v1.0.3
	github.com/dustin/go-humanize v1.0.1
	github.com/facette/natsort v0.0.0-20181210072756-2cd4dd1e2dcb
	github.com/go-kit/log v0.2.1
	github.com/go-logfmt/logfmt v0.6.0
	github.com/go-redis/redis/v8 v8.11.5
	github.com/go-test/deep v1.1.1
	github.com/gogo/protobuf v1.3.2
	github.com/gogo/status v1.1.1
	github.com/golang/protobuf v1.5.4
	github.com/golang/snappy v1.0.0
	github.com/google/go-cmp v0.7.0
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1
	github.com/grafana/dskit v0.0.0-20250529170946-28928403e61e
	github.com/grafana/e2e v0.1.2-0.20250428181430-708d63bcc673
	github.com/hashicorp/go-hclog v1.6.3 // indirect
	github.com/jaegertracing/jaeger v1.67.0
	github.com/jedib0t/go-pretty/v6 v6.6.7
	github.com/json-iterator/go v1.1.12
	github.com/jsternberg/zap-logfmt v1.2.0
	github.com/klauspost/compress v1.18.0
	github.com/minio/minio-go/v7 v7.0.93
	github.com/mitchellh/mapstructure v1.5.1-0.20231216201459-8508981c8b6c // indirect
	github.com/olekukonko/tablewriter v0.0.5
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/jaeger v0.127.0
	github.com/opentracing-contrib/go-grpc v0.1.1 // indirect
	github.com/opentracing/opentracing-go v1.2.0
	github.com/pierrec/lz4/v4 v4.1.22
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.22.0
	github.com/prometheus/client_model v0.6.2
	github.com/prometheus/common v0.64.0
	github.com/prometheus/prometheus v0.304.1
	github.com/prometheus/statsd_exporter v0.26.1
	github.com/segmentio/fasthash v1.0.3
	github.com/sony/gobreaker v0.4.1
	github.com/spf13/viper v1.20.1
	github.com/stretchr/testify v1.10.0
	github.com/uber-go/atomic v1.4.0
	github.com/uber/jaeger-client-go v2.30.0+incompatible // indirect
	github.com/willf/bloom v2.0.3+incompatible
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/collector v0.127.0 // indirect
	go.opentelemetry.io/collector/component v1.33.0
	go.opentelemetry.io/collector/confmap v1.33.0
	go.opentelemetry.io/collector/consumer v1.33.0
	go.opentelemetry.io/collector/pdata v1.33.0
	go.opentelemetry.io/collector/semconv v0.124.0 // indirect
	go.opentelemetry.io/otel v1.36.0
	go.opentelemetry.io/otel/bridge/opencensus v1.36.0
	go.opentelemetry.io/otel/bridge/opentracing v1.36.0
	go.opentelemetry.io/otel/metric v1.36.0
	go.opentelemetry.io/otel/sdk v1.36.0
	go.opentelemetry.io/otel/trace v1.36.0
	go.uber.org/atomic v1.11.0
	go.uber.org/goleak v1.3.0
	go.uber.org/multierr v1.11.0
	go.uber.org/zap v1.27.0
	golang.org/x/sync v0.15.0
	golang.org/x/time v0.12.0
	google.golang.org/api v0.237.0
	google.golang.org/grpc v1.73.0
	google.golang.org/protobuf v1.36.6
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.18.0
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.10.1
	github.com/Azure/azure-sdk-for-go/sdk/storage/azblob v1.6.1
	github.com/evanphx/json-patch v5.9.11+incompatible
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8
	github.com/googleapis/gax-go/v2 v2.14.2
	github.com/grafana/gomemcache v0.0.0-20250318131618-74242eea118d
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0
	github.com/jaegertracing/jaeger-idl v0.6.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/zipkinexporter v0.127.0
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl v0.127.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/filterprocessor v0.127.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/jaegerreceiver v0.127.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver v0.127.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/opencensusreceiver v0.127.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/zipkinreceiver v0.127.0
	github.com/parquet-go/parquet-go v0.25.1
	github.com/twmb/franz-go v1.18.1
	github.com/twmb/franz-go/pkg/kadm v1.16.0
	github.com/twmb/franz-go/pkg/kfake v0.0.0-20250320172111-35ab5e5f5327
	github.com/twmb/franz-go/pkg/kmsg v1.11.2
	github.com/twmb/franz-go/plugin/kotel v1.6.0
	github.com/twmb/franz-go/plugin/kprom v1.2.1
	go.opentelemetry.io/collector/client v1.33.0
	go.opentelemetry.io/collector/component/componenttest v0.127.0
	go.opentelemetry.io/collector/config/configgrpc v0.127.0
	go.opentelemetry.io/collector/config/confighttp v0.127.0
	go.opentelemetry.io/collector/config/configopaque v1.33.0
	go.opentelemetry.io/collector/config/configtls v1.33.0
	go.opentelemetry.io/collector/exporter v0.127.0
	go.opentelemetry.io/collector/exporter/exportertest v0.127.0
	go.opentelemetry.io/collector/exporter/otlpexporter v0.127.0
	go.opentelemetry.io/collector/exporter/otlphttpexporter v0.127.0
	go.opentelemetry.io/collector/otelcol v0.127.0
	go.opentelemetry.io/collector/pdata/testdata v0.127.0
	go.opentelemetry.io/collector/processor v1.33.0
	go.opentelemetry.io/collector/receiver v1.33.0
	go.opentelemetry.io/collector/receiver/otlpreceiver v0.127.0
	go.opentelemetry.io/contrib/exporters/autoexport v0.61.0
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.61.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.36.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.36.0
	go.opentelemetry.io/proto/otlp v1.7.0
	golang.org/x/net v0.41.0
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250603155806-513f23925822
)

require (
	cel.dev/expr v0.23.0 // indirect
	cloud.google.com/go v0.121.1 // indirect
	cloud.google.com/go/auth v0.16.2 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	cloud.google.com/go/compute/metadata v0.7.0 // indirect
	cloud.google.com/go/iam v1.5.2 // indirect
	cloud.google.com/go/monitoring v1.24.2 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.11.1 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.4.2 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp v1.27.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric v0.51.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/internal/resourcemapping v0.51.0 // indirect
	github.com/IBM/sarama v1.45.1 // indirect
	github.com/alecthomas/participle/v2 v2.1.4 // indirect
	github.com/alecthomas/units v0.0.0-20240927000941-0f3dac36c52b // indirect
	github.com/andybalholm/brotli v1.1.1 // indirect
	github.com/antchfx/xmlquery v1.4.4 // indirect
	github.com/antchfx/xpath v1.3.4 // indirect
	github.com/apache/thrift v0.21.0 // indirect
	github.com/armon/go-metrics v0.4.1 // indirect
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/aws/aws-msk-iam-sasl-signer-go v1.0.4 // indirect
	github.com/aws/aws-sdk-go-v2 v1.36.3 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.28.2 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.17.67 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.30 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.12.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.12.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.25.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.30.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.33.19 // indirect
	github.com/aws/smithy-go v1.22.2 // indirect
	github.com/bboreham/go-loser v0.0.0-20230920113527-fcc2c21820a3 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cenkalti/backoff/v5 v5.0.2 // indirect
	github.com/census-instrumentation/opencensus-proto v0.4.1 // indirect
	github.com/cncf/xds/go v0.0.0-20250326154945-ae57f3c0d45f // indirect
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/dennwc/varint v1.0.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/eapache/go-resiliency v1.7.0 // indirect
	github.com/eapache/go-xerial-snappy v0.0.0-20230731223053-c322873962e3 // indirect
	github.com/eapache/queue v1.1.0 // indirect
	github.com/ebitengine/purego v0.8.4 // indirect
	github.com/edsrzf/mmap-go v1.2.0 // indirect
	github.com/elastic/go-grok v0.3.1 // indirect
	github.com/elastic/lunes v0.1.0 // indirect
	github.com/envoyproxy/go-control-plane/envoy v1.32.4 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.2.1 // indirect
	github.com/expr-lang/expr v1.17.3 // indirect
	github.com/fatih/color v1.16.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/foxboron/go-tpm-keyfiles v0.0.0-20250323135004-b31fac66206e // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-ini/ini v1.67.0 // indirect
	github.com/go-jose/go-jose/v4 v4.0.5 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-openapi/analysis v0.23.0 // indirect
	github.com/go-openapi/errors v0.22.0 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/loads v0.22.0 // indirect
	github.com/go-openapi/spec v0.21.0 // indirect
	github.com/go-openapi/strfmt v0.23.0 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/go-openapi/validate v0.24.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.2.1 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/gogo/googleapis v1.4.1 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.2 // indirect
	github.com/google/btree v1.1.2 // indirect
	github.com/google/go-tpm v0.9.5 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.6 // indirect
	github.com/gorilla/handlers v1.5.2 // indirect
	github.com/grafana/otel-profiling-go v0.5.1 // indirect
	github.com/grafana/pyroscope-go/godeltaprof v0.1.8 // indirect
	github.com/grafana/regexp v0.0.0-20240518133315-a468a5bfb3bc // indirect
	github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.3.1 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.26.3 // indirect
	github.com/hashicorp/consul/api v1.32.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-msgpack v0.5.5 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/go-sockaddr v1.0.7 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/hashicorp/go-version v1.7.0 // indirect
	github.com/hashicorp/golang-lru v1.0.2 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/hashicorp/memberlist v0.5.1 // indirect
	github.com/hashicorp/serf v0.10.1 // indirect
	github.com/iancoleman/strcase v0.3.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jcmturner/aescts/v2 v2.0.0 // indirect
	github.com/jcmturner/dnsutils/v2 v2.0.0 // indirect
	github.com/jcmturner/gofork v1.7.6 // indirect
	github.com/jcmturner/gokrb5/v8 v8.4.4 // indirect
	github.com/jcmturner/rpc/v2 v2.0.3 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/jpillora/backoff v1.0.0 // indirect
	github.com/julienschmidt/httprouter v1.3.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.10 // indirect
	github.com/knadh/koanf v1.5.0 // indirect
	github.com/knadh/koanf/v2 v2.2.0 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/lufia/plan9stats v0.0.0-20240909124753-873cd0166683 // indirect
	github.com/magefile/mage v1.15.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/mdlayher/socket v0.4.1 // indirect
	github.com/mdlayher/vsock v1.2.1 // indirect
	github.com/miekg/dns v1.1.65 // indirect
	github.com/minio/crc64nvme v1.0.1 // indirect
	github.com/minio/md5-simd v1.1.2 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/mostynb/go-grpc-compression v1.2.3 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/mwitkow/go-conntrack v0.0.0-20190716064945-2f068394615f // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/oklog/ulid/v2 v2.1.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal v0.127.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/exp/metrics v0.124.1 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter v0.127.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/kafka v0.127.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/sharedcomponent v0.127.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/core/xidutils v0.127.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/kafka/configkafka v0.127.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatautil v0.127.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/azure v0.127.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/opencensus v0.127.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/zipkin v0.127.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/deltatocumulativeprocessor v0.124.1 // indirect
	github.com/opentracing-contrib/go-stdlib v1.0.0 // indirect
	github.com/openzipkin/zipkin-go v0.4.3 // indirect
	github.com/pelletier/go-toml/v2 v2.2.3 // indirect
	github.com/philhofer/fwd v1.1.3-0.20240916144458-20a13a1f6b7c // indirect
	github.com/pires/go-proxyproto v0.7.0 // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/planetscale/vtprotobuf v0.6.1-0.20240319094008-0393e58bdf10 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/prometheus/alertmanager v0.28.1 // indirect
	github.com/prometheus/exporter-toolkit v0.14.0 // indirect
	github.com/prometheus/otlptranslator v0.0.0-20250320144820-d800c8b0eb07 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	github.com/prometheus/sigv4 v0.1.2 // indirect
	github.com/puzpuzpuz/xsync/v3 v3.5.1 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475 // indirect
	github.com/relvacode/iso8601 v1.6.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/rs/cors v1.11.1 // indirect
	github.com/rs/xid v1.6.0 // indirect
	github.com/sagikazarmark/locafero v0.7.0 // indirect
	github.com/sean-/seed v0.0.0-20170313163322-e2103e2c3529 // indirect
	github.com/sercand/kuberesolver/v6 v6.0.0 // indirect
	github.com/shirou/gopsutil/v4 v4.25.4 // indirect
	github.com/soheilhy/cmux v0.1.5 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/spf13/afero v1.12.0 // indirect
	github.com/spf13/cast v1.7.1 // indirect
	github.com/spf13/cobra v1.9.1 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	github.com/spiffe/go-spiffe/v2 v2.5.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/tinylib/msgp v1.3.0 // indirect
	github.com/tklauser/go-sysconf v0.3.14 // indirect
	github.com/tklauser/numcpus v0.8.0 // indirect
	github.com/twmb/murmur3 v1.1.8 // indirect
	github.com/ua-parser/uap-go v0.0.0-20241012191800-bbb40edc15aa // indirect
	github.com/uber/jaeger-lib v2.4.1+incompatible // indirect
	github.com/willf/bitset v1.1.11 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/yuin/gopher-lua v1.1.1 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	github.com/zeebo/errs v1.4.0 // indirect
	go.etcd.io/etcd/api/v3 v3.5.12 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.5.12 // indirect
	go.etcd.io/etcd/client/v3 v3.5.12 // indirect
	go.mongodb.org/mongo-driver v1.15.0 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/collector/component/componentstatus v0.127.0 // indirect
	go.opentelemetry.io/collector/config/configauth v0.127.0 // indirect
	go.opentelemetry.io/collector/config/configcompression v1.33.0 // indirect
	go.opentelemetry.io/collector/config/configmiddleware v0.127.0 // indirect
	go.opentelemetry.io/collector/config/confignet v1.33.0 // indirect
	go.opentelemetry.io/collector/config/configretry v1.33.0 // indirect
	go.opentelemetry.io/collector/config/configtelemetry v0.127.0 // indirect
	go.opentelemetry.io/collector/confmap/xconfmap v0.127.0 // indirect
	go.opentelemetry.io/collector/connector v0.127.0 // indirect
	go.opentelemetry.io/collector/connector/connectortest v0.127.0 // indirect
	go.opentelemetry.io/collector/connector/xconnector v0.127.0 // indirect
	go.opentelemetry.io/collector/consumer/consumererror v0.127.0 // indirect
	go.opentelemetry.io/collector/consumer/consumererror/xconsumererror v0.127.0 // indirect
	go.opentelemetry.io/collector/consumer/consumertest v0.127.0 // indirect
	go.opentelemetry.io/collector/consumer/xconsumer v0.127.0 // indirect
	go.opentelemetry.io/collector/exporter/exporterhelper/xexporterhelper v0.127.0 // indirect
	go.opentelemetry.io/collector/exporter/xexporter v0.127.0 // indirect
	go.opentelemetry.io/collector/extension v1.33.0 // indirect
	go.opentelemetry.io/collector/extension/extensionauth v1.33.0 // indirect
	go.opentelemetry.io/collector/extension/extensioncapabilities v0.127.0 // indirect
	go.opentelemetry.io/collector/extension/extensionmiddleware v0.127.0 // indirect
	go.opentelemetry.io/collector/extension/extensiontest v0.127.0 // indirect
	go.opentelemetry.io/collector/extension/xextension v0.127.0 // indirect
	go.opentelemetry.io/collector/featuregate v1.33.0 // indirect
	go.opentelemetry.io/collector/internal/fanoutconsumer v0.127.0 // indirect
	go.opentelemetry.io/collector/internal/sharedcomponent v0.127.0 // indirect
	go.opentelemetry.io/collector/internal/telemetry v0.127.0 // indirect
	go.opentelemetry.io/collector/pdata/pprofile v0.127.0 // indirect
	go.opentelemetry.io/collector/pipeline v0.127.0 // indirect
	go.opentelemetry.io/collector/pipeline/xpipeline v0.127.0 // indirect
	go.opentelemetry.io/collector/processor/processorhelper v0.127.0 // indirect
	go.opentelemetry.io/collector/processor/processortest v0.127.0 // indirect
	go.opentelemetry.io/collector/processor/xprocessor v0.127.0 // indirect
	go.opentelemetry.io/collector/receiver/receiverhelper v0.127.0 // indirect
	go.opentelemetry.io/collector/receiver/receivertest v0.127.0 // indirect
	go.opentelemetry.io/collector/receiver/xreceiver v0.127.0 // indirect
	go.opentelemetry.io/collector/service v0.127.0 // indirect
	go.opentelemetry.io/collector/service/hostcapabilities v0.127.0 // indirect
	go.opentelemetry.io/contrib/bridges/otelzap v0.10.0 // indirect
	go.opentelemetry.io/contrib/bridges/prometheus v0.61.0 // indirect
	go.opentelemetry.io/contrib/detectors/gcp v1.36.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace v0.60.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.61.0 // indirect
	go.opentelemetry.io/contrib/otelconf v0.15.0 // indirect
	go.opentelemetry.io/contrib/propagators/b3 v1.35.0 // indirect
	go.opentelemetry.io/contrib/propagators/jaeger v1.35.0 // indirect
	go.opentelemetry.io/contrib/samplers/jaegerremote v0.30.0 // indirect
	go.opentelemetry.io/otel/exporters/jaeger v1.17.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc v0.12.2 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp v0.12.2 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.36.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.36.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.36.0 // indirect
	go.opentelemetry.io/otel/exporters/prometheus v0.58.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdoutlog v0.12.2 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v1.36.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.36.0 // indirect
	go.opentelemetry.io/otel/log v0.12.2 // indirect
	go.opentelemetry.io/otel/log/logtest v0.0.0-20250602073710-889a4862b40f // indirect
	go.opentelemetry.io/otel/sdk/log v0.12.2 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.36.0 // indirect
	golang.org/x/crypto v0.39.0 // indirect
	golang.org/x/exp v0.0.0-20250106191152-7588d65b2ba8 // indirect
	golang.org/x/mod v0.25.0 // indirect
	golang.org/x/oauth2 v0.30.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	golang.org/x/tools v0.33.0 // indirect
	gonum.org/v1/gonum v0.16.0 // indirect
	google.golang.org/genproto v0.0.0-20250505200425-f936aa4a68b2 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250528174236-200df99c418a // indirect
	k8s.io/apimachinery v0.32.3 // indirect
	k8s.io/client-go v0.32.3 // indirect
	k8s.io/klog/v2 v2.130.1 // indirect
	k8s.io/utils v0.0.0-20241104100929-3ea5e8cea738 // indirect
	sigs.k8s.io/yaml v1.4.0 // indirect
)

replace (
	k8s.io/api => k8s.io/api v0.25.0
	k8s.io/client-go => k8s.io/client-go v0.25.0
)

// Replace memberlist with our fork which includes some fixes that haven't been
// merged upstream yet:
// - https://github.com/hashicorp/memberlist/pull/260
// - https://github.com/grafana/memberlist/pull/3
replace github.com/hashicorp/memberlist => github.com/grafana/memberlist v0.3.1-0.20220708130638-bd88e10a3d91
