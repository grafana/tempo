module github.com/grafana/tempo

go 1.20

require (
	cloud.google.com/go/storage v1.28.1
	contrib.go.opencensus.io/exporter/prometheus v0.4.2
	github.com/alecthomas/kong v0.8.0
	github.com/alicebob/miniredis/v2 v2.21.0
	github.com/aws/aws-sdk-go v1.44.220
	github.com/cespare/xxhash v1.1.0
	github.com/cespare/xxhash/v2 v2.2.0
	github.com/cristalhq/hedgedhttp v0.7.0
	github.com/davecgh/go-spew v1.1.1
	github.com/drone/envsubst v1.0.3
	github.com/dustin/go-humanize v1.0.1
	github.com/facette/natsort v0.0.0-20181210072756-2cd4dd1e2dcb
	github.com/go-kit/log v0.2.1
	github.com/go-logfmt/logfmt v0.5.1
	github.com/go-redis/redis/v8 v8.11.5
	github.com/go-test/deep v1.0.8
	github.com/gogo/protobuf v1.3.2
	github.com/gogo/status v1.1.1
	github.com/golang/protobuf v1.5.3
	github.com/golang/snappy v0.0.4
	github.com/google/go-cmp v0.5.9
	github.com/google/uuid v1.3.0
	github.com/gorilla/mux v1.8.0
	github.com/grafana/dskit v0.0.0-20230804003603-740f56bd2934
	github.com/grafana/e2e v0.1.1-0.20221018202458-cffd2bb71c7b
	github.com/grpc-ecosystem/grpc-opentracing v0.0.0-20180507213350-8e809c8a8645
	github.com/hashicorp/go-hclog v1.4.0
	github.com/hashicorp/go-plugin v1.4.8
	github.com/jaegertracing/jaeger v1.41.0
	github.com/jedib0t/go-pretty/v6 v6.2.4
	github.com/json-iterator/go v1.1.12
	github.com/jsternberg/zap-logfmt v1.2.0
	github.com/klauspost/compress v1.16.7
	github.com/minio/minio-go/v7 v7.0.52
	github.com/mitchellh/mapstructure v1.5.0
	github.com/olekukonko/tablewriter v0.0.5
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/jaegerexporter v0.74.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/zipkinexporter v0.74.0
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/jaeger v0.74.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/jaegerreceiver v0.74.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver v0.74.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/opencensusreceiver v0.74.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/zipkinreceiver v0.74.0
	github.com/opentracing-contrib/go-grpc v0.0.0-20210225150812-73cb765af46e
	github.com/opentracing/opentracing-go v1.2.0
	github.com/pierrec/lz4/v4 v4.1.18
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.15.1
	github.com/prometheus/client_model v0.4.0
	github.com/prometheus/common v0.43.0
	github.com/prometheus/prometheus v1.8.2-0.20221021121301-51a44e6657c3
	github.com/prometheus/statsd_exporter v0.22.7 // indirect
	github.com/segmentio/fasthash v0.0.0-20180216231524-a72b379d632e
	github.com/sirupsen/logrus v1.9.0
	github.com/sony/gobreaker v0.4.1
	github.com/spf13/viper v1.14.0
	github.com/stretchr/testify v1.8.4
	github.com/uber-go/atomic v1.4.0
	github.com/uber/jaeger-client-go v2.30.0+incompatible
	github.com/willf/bloom v2.0.3+incompatible
	go.opencensus.io v0.24.0
	go.opentelemetry.io/collector v0.74.0
	go.opentelemetry.io/collector/component v0.74.0
	go.opentelemetry.io/collector/confmap v0.74.0
	go.opentelemetry.io/collector/consumer v0.74.0
	go.opentelemetry.io/collector/exporter/otlpexporter v0.74.0
	go.opentelemetry.io/collector/pdata v1.0.0-rc8
	go.opentelemetry.io/collector/receiver/otlpreceiver v0.74.0
	go.opentelemetry.io/collector/semconv v0.75.0
	go.opentelemetry.io/otel v1.14.0
	go.opentelemetry.io/otel/bridge/opencensus v0.37.0
	go.opentelemetry.io/otel/bridge/opentracing v1.10.0
	go.opentelemetry.io/otel/exporters/jaeger v1.10.0
	go.opentelemetry.io/otel/metric v0.37.0
	go.opentelemetry.io/otel/sdk v1.14.0
	go.opentelemetry.io/otel/trace v1.14.0
	go.uber.org/atomic v1.10.0
	go.uber.org/goleak v1.2.0
	go.uber.org/multierr v1.10.0
	go.uber.org/zap v1.24.0
	golang.org/x/sync v0.1.0
	golang.org/x/time v0.1.0
	google.golang.org/api v0.110.0
	google.golang.org/grpc v1.55.0
	google.golang.org/protobuf v1.30.0
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.7.1
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.3.0
	github.com/Azure/azure-sdk-for-go/sdk/storage/azblob v1.1.0
	github.com/evanphx/json-patch v4.12.0+incompatible
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da
	github.com/googleapis/gax-go/v2 v2.7.0
	github.com/grafana/gomemcache v0.0.0-20230316202710-a081dae0aba9
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl v0.74.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/filterprocessor v0.74.0
	github.com/parquet-go/parquet-go v0.17.0
	github.com/stoewer/parquet-cli v0.0.5
	go.opentelemetry.io/collector/exporter v0.74.0
	go.opentelemetry.io/collector/receiver v0.74.0
	golang.org/x/exp v0.0.0-20230321023759-10a507213a29
	golang.org/x/oauth2 v0.7.0
)

require (
	cloud.google.com/go v0.110.0 // indirect
	cloud.google.com/go/compute v1.18.0 // indirect
	cloud.google.com/go/compute/metadata v0.2.3 // indirect
	cloud.google.com/go/iam v0.12.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.3.0 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.0.0 // indirect
	github.com/Microsoft/go-winio v0.5.2 // indirect
	github.com/Shopify/sarama v1.38.1 // indirect
	github.com/VividCortex/gohistogram v1.0.0 // indirect
	github.com/alecthomas/participle/v2 v2.0.0-beta.5 // indirect
	github.com/alecthomas/units v0.0.0-20211218093645-b94a6e3cc137 // indirect
	github.com/alicebob/gopher-json v0.0.0-20200520072559-a9ecdc9d1d3a // indirect
	github.com/andybalholm/brotli v1.0.5 // indirect
	github.com/antonmedv/expr v1.12.3 // indirect
	github.com/apache/thrift v0.18.1 // indirect
	github.com/armon/go-metrics v0.4.0 // indirect
	github.com/asaskevich/govalidator v0.0.0-20210307081110-f21760c49a8d // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cenkalti/backoff/v4 v4.2.0 // indirect
	github.com/census-instrumentation/opencensus-proto v0.4.1 // indirect
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/go-systemd/v22 v22.4.0 // indirect
	github.com/dennwc/varint v1.0.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/eapache/go-resiliency v1.3.0 // indirect
	github.com/eapache/go-xerial-snappy v0.0.0-20230111030713-bf00bc1b83b6 // indirect
	github.com/eapache/queue v1.1.0 // indirect
	github.com/edsrzf/mmap-go v1.1.0 // indirect
	github.com/fatih/color v1.13.0 // indirect
	github.com/felixge/httpsnoop v1.0.3 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/go-kit/kit v0.12.0 // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-openapi/analysis v0.21.2 // indirect
	github.com/go-openapi/errors v0.20.2 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.20.0 // indirect
	github.com/go-openapi/loads v0.21.1 // indirect
	github.com/go-openapi/spec v0.20.6 // indirect
	github.com/go-openapi/strfmt v0.21.3 // indirect
	github.com/go-openapi/swag v0.22.1 // indirect
	github.com/go-openapi/validate v0.22.0 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/gogo/googleapis v1.4.1 // indirect
	github.com/golang-jwt/jwt/v4 v4.5.0 // indirect
	github.com/google/btree v1.0.1 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.2.3 // indirect
	github.com/gorilla/handlers v1.5.1 // indirect
	github.com/grafana/regexp v0.0.0-20221123153739-15dc172cd2db // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.15.2 // indirect
	github.com/hashicorp/consul/api v1.15.3 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-msgpack v0.5.5 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/go-sockaddr v1.0.2 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/hashicorp/memberlist v0.3.1 // indirect
	github.com/hashicorp/serf v0.9.8 // indirect
	github.com/hashicorp/yamux v0.0.0-20190923154419-df201c70410d // indirect
	github.com/iancoleman/strcase v0.2.0 // indirect
	github.com/inconshreveable/mousetrap v1.0.1 // indirect
	github.com/jcmturner/aescts/v2 v2.0.0 // indirect
	github.com/jcmturner/dnsutils/v2 v2.0.0 // indirect
	github.com/jcmturner/gofork v1.7.6 // indirect
	github.com/jcmturner/gokrb5/v8 v8.4.3 // indirect
	github.com/jcmturner/rpc/v2 v2.0.3 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/jpillora/backoff v1.0.0 // indirect
	github.com/julienschmidt/httprouter v1.3.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.4 // indirect
	github.com/knadh/koanf v1.5.0 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/magiconair/properties v1.8.6 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.16 // indirect
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/miekg/dns v1.1.50 // indirect
	github.com/minio/md5-simd v1.1.2 // indirect
	github.com/minio/sha256-simd v1.0.0 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/go-testing-interface v1.0.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/mostynb/go-grpc-compression v1.1.17 // indirect
	github.com/mwitkow/go-conntrack v0.0.0-20190716064945-2f068394615f // indirect
	github.com/oklog/run v1.1.0 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/kafkaexporter v0.74.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal v0.74.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter v0.74.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/sharedcomponent v0.74.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatautil v0.74.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/opencensus v0.74.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/zipkin v0.74.0 // indirect
	github.com/opencontainers/image-spec v1.0.3-0.20220512140940-7b36cea86235 // indirect
	github.com/opentracing-contrib/go-stdlib v1.0.0 // indirect
	github.com/openzipkin/zipkin-go v0.4.1 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pelletier/go-toml/v2 v2.0.5 // indirect
	github.com/pkg/browser v0.0.0-20210911075715-681adbf594b8 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/power-devops/perfstat v0.0.0-20210106213030-5aafc221ea8c // indirect
	github.com/prometheus/alertmanager v0.24.0 // indirect
	github.com/prometheus/common/sigv4 v0.1.0 // indirect
	github.com/prometheus/exporter-toolkit v0.8.2 // indirect
	github.com/prometheus/procfs v0.9.0 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475 // indirect
	github.com/rivo/uniseg v0.4.4 // indirect
	github.com/rs/cors v1.8.3 // indirect
	github.com/rs/xid v1.5.0 // indirect
	github.com/sean-/seed v0.0.0-20170313163322-e2103e2c3529 // indirect
	github.com/segmentio/encoding v0.3.6 // indirect
	github.com/sercand/kuberesolver/v4 v4.0.0 // indirect
	github.com/shirou/gopsutil/v3 v3.23.2 // indirect
	github.com/soheilhy/cmux v0.1.5 // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/spf13/afero v1.9.2 // indirect
	github.com/spf13/cast v1.5.0 // indirect
	github.com/spf13/cobra v1.6.1 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stretchr/objx v0.5.0 // indirect
	github.com/subosito/gotenv v1.4.1 // indirect
	github.com/tklauser/go-sysconf v0.3.11 // indirect
	github.com/tklauser/numcpus v0.6.0 // indirect
	github.com/uber/jaeger-lib v2.4.1+incompatible // indirect
	github.com/weaveworks/promrus v1.2.0 // indirect
	github.com/willf/bitset v1.1.11 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/yuin/gopher-lua v0.0.0-20210529063254-f4c35e4016d9 // indirect
	github.com/yusufpapurcu/wmi v1.2.2 // indirect
	go.etcd.io/etcd/api/v3 v3.5.5 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.5.5 // indirect
	go.etcd.io/etcd/client/v3 v3.5.5 // indirect
	go.mongodb.org/mongo-driver v1.10.2 // indirect
	go.opentelemetry.io/collector/featuregate v0.74.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.40.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.40.0 // indirect
	go.opentelemetry.io/contrib/propagators/b3 v1.15.0 // indirect
	go.opentelemetry.io/otel/exporters/prometheus v0.37.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v0.37.0 // indirect
	golang.org/x/crypto v0.9.0 // indirect
	golang.org/x/mod v0.9.0 // indirect
	golang.org/x/net v0.10.0 // indirect
	golang.org/x/sys v0.10.0 // indirect
	golang.org/x/text v0.9.0 // indirect
	golang.org/x/tools v0.7.0 // indirect
	golang.org/x/xerrors v0.0.0-20220907171357-04be3eba64a2 // indirect
	gonum.org/v1/gonum v0.12.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20230306155012-7f2fa6fef1f4 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
)

replace (
	github.com/bradfitz/gomemcache => github.com/themihai/gomemcache v0.0.0-20180902122335-24332e2d58ab
	k8s.io/api => k8s.io/api v0.25.0
	k8s.io/client-go => k8s.io/client-go v0.25.0
)

// Replace memberlist with our fork which includes some fixes that haven't been
// merged upstream yet:
// - https://github.com/hashicorp/memberlist/pull/260
// - https://github.com/grafana/memberlist/pull/3
replace github.com/hashicorp/memberlist => github.com/grafana/memberlist v0.3.1-0.20220708130638-bd88e10a3d91

// Pinned here until we upgrade jaeger.  grpc 1.55 removes grpc_testing
// which our current version of jaeger depends on.
replace google.golang.org/grpc => google.golang.org/grpc v1.54.0
