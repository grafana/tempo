module github.com/grafana/tempo

go 1.17

require (
	cloud.google.com/go/storage v1.18.2
	contrib.go.opencensus.io/exporter/prometheus v0.4.0
	github.com/Azure/azure-pipeline-go v0.2.3
	github.com/Azure/azure-storage-blob-go v0.13.0
	github.com/alecthomas/kong v0.2.11
	github.com/aws/aws-sdk-go v1.42.21
	github.com/cespare/xxhash v1.1.0
	github.com/cortexproject/cortex v1.10.1-0.20211214110443-07d6cfc4eb84
	github.com/cristalhq/hedgedhttp v0.7.0
	github.com/drone/envsubst v1.0.3
	github.com/dustin/go-humanize v1.0.0
	github.com/go-kit/log v0.2.0
	github.com/go-logfmt/logfmt v0.5.1
	github.com/go-test/deep v1.0.7
	github.com/gogo/protobuf v1.3.2
	github.com/gogo/status v1.1.0
	github.com/golang/protobuf v1.5.2
	github.com/golang/snappy v0.0.4
	github.com/google/flatbuffers v2.0.0+incompatible
	github.com/google/go-cmp v0.5.6
	github.com/google/uuid v1.3.0
	github.com/gorilla/mux v1.8.0
	github.com/grafana/dskit v0.0.0-20211021180445-3bd016e9d7f1
	github.com/grpc-ecosystem/grpc-opentracing v0.0.0-20180507213350-8e809c8a8645
	github.com/hashicorp/go-hclog v1.0.0
	github.com/hashicorp/go-plugin v1.4.3
	github.com/jaegertracing/jaeger v1.29.0
	github.com/jedib0t/go-pretty/v6 v6.2.4
	github.com/jsternberg/zap-logfmt v1.2.0
	github.com/klauspost/compress v1.13.6
	github.com/minio/minio-go/v7 v7.0.16-0.20211116163909-d00629356463
	github.com/olekukonko/tablewriter v0.0.2
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/jaegerexporter v0.41.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/zipkinexporter v0.40.0
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/jaeger v0.41.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/jaegerreceiver v0.40.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver v0.40.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/opencensusreceiver v0.40.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/zipkinreceiver v0.40.0
	github.com/opentracing-contrib/go-grpc v0.0.0-20210225150812-73cb765af46e
	github.com/opentracing/opentracing-go v1.2.0
	github.com/pierrec/lz4/v4 v4.1.12
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.32.1
	github.com/prometheus/prometheus v1.8.2-0.20211119115433-692a54649ed7
	github.com/prometheus/statsd_exporter v0.21.0 // indirect
	github.com/segmentio/fasthash v0.0.0-20180216231524-a72b379d632e
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/viper v1.9.0
	github.com/stretchr/testify v1.7.0
	github.com/thanos-io/thanos v0.22.0
	github.com/uber-go/atomic v1.4.0
	github.com/uber/jaeger-client-go v2.29.1+incompatible
	github.com/weaveworks/common v0.0.0-20210913144402-035033b78a78
	github.com/willf/bloom v2.0.3+incompatible
	go.opencensus.io v0.23.0
	go.opentelemetry.io/collector v0.41.0
	go.opentelemetry.io/collector/model v0.41.0
	go.opentelemetry.io/otel v1.2.0
	go.opentelemetry.io/otel/bridge/opencensus v0.25.0
	go.opentelemetry.io/otel/bridge/opentracing v1.2.0
	go.opentelemetry.io/otel/exporters/jaeger v1.2.0
	go.opentelemetry.io/otel/metric v0.25.0
	go.opentelemetry.io/otel/sdk v1.2.0
	go.opentelemetry.io/otel/trace v1.2.0
	go.uber.org/atomic v1.9.0
	go.uber.org/goleak v1.1.12
	go.uber.org/multierr v1.7.0
	go.uber.org/zap v1.19.1
	golang.org/x/time v0.0.0-20211116232009-f0f3c7e86c11
	google.golang.org/api v0.61.0
	google.golang.org/grpc v1.42.0
	google.golang.org/protobuf v1.27.1
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
)

require (
	cloud.google.com/go v0.99.0 // indirect
	cloud.google.com/go/bigtable v1.3.0 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest v0.11.22 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.17 // indirect
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.8 // indirect
	github.com/Azure/go-autorest/autorest/azure/cli v0.4.2 // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.0 // indirect
	github.com/Azure/go-autorest/logger v0.2.1 // indirect
	github.com/Azure/go-autorest/tracing v0.6.0 // indirect
	github.com/Masterminds/squirrel v0.0.0-20161115235646-20f192218cf5 // indirect
	github.com/NYTimes/gziphandler v1.1.1 // indirect
	github.com/PuerkitoBio/purell v1.1.1 // indirect
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578 // indirect
	github.com/Shopify/sarama v1.30.1 // indirect
	github.com/VividCortex/gohistogram v1.0.0 // indirect
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751 // indirect
	github.com/alecthomas/units v0.0.0-20210927113745-59d0afb8317a // indirect
	github.com/apache/thrift v0.15.0 // indirect
	github.com/armon/go-metrics v0.3.9 // indirect
	github.com/asaskevich/govalidator v0.0.0-20210307081110-f21760c49a8d // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/bradfitz/gomemcache v0.0.0-20190913173617-a41fca850d0b // indirect
	github.com/cenkalti/backoff/v4 v4.1.2 // indirect
	github.com/census-instrumentation/opencensus-proto v0.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/cncf/udpa/go v0.0.0-20201120205902-5459f2c99403 // indirect
	github.com/cncf/xds/go v0.0.0-20211001041855-01bcc9b48dfe // indirect
	github.com/coreos/etcd v3.3.25+incompatible // indirect
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/go-systemd v0.0.0-20191104093116-d3cd4ed1dbcf // indirect
	github.com/coreos/go-systemd/v22 v22.3.2 // indirect
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dennwc/varint v1.0.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/dimchansky/utfbom v1.1.1 // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/eapache/go-resiliency v1.2.0 // indirect
	github.com/eapache/go-xerial-snappy v0.0.0-20180814174437-776d5712da21 // indirect
	github.com/eapache/queue v1.1.0 // indirect
	github.com/edsrzf/mmap-go v1.0.0 // indirect
	github.com/envoyproxy/go-control-plane v0.10.1 // indirect
	github.com/envoyproxy/protoc-gen-validate v0.6.2 // indirect
	github.com/facette/natsort v0.0.0-20181210072756-2cd4dd1e2dcb // indirect
	github.com/fatih/color v1.12.0 // indirect
	github.com/felixge/fgprof v0.9.1 // indirect
	github.com/felixge/httpsnoop v1.0.2 // indirect
	github.com/fsnotify/fsnotify v1.5.1 // indirect
	github.com/fsouza/fake-gcs-server v1.7.0 // indirect
	github.com/go-kit/kit v0.12.0 // indirect
	github.com/go-openapi/analysis v0.20.1 // indirect
	github.com/go-openapi/errors v0.20.1 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.19.6 // indirect
	github.com/go-openapi/loads v0.21.0 // indirect
	github.com/go-openapi/runtime v0.21.0 // indirect
	github.com/go-openapi/spec v0.20.4 // indirect
	github.com/go-openapi/strfmt v0.21.1 // indirect
	github.com/go-openapi/swag v0.19.15 // indirect
	github.com/go-openapi/validate v0.20.3 // indirect
	github.com/go-redis/redis/v8 v8.11.4 // indirect
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/gocql/gocql v0.0.0-20200526081602-cd04bd7f22a7 // indirect
	github.com/gofrs/uuid v4.0.0+incompatible // indirect
	github.com/gogo/googleapis v1.4.1 // indirect
	github.com/golang-jwt/jwt/v4 v4.0.0 // indirect
	github.com/golang-migrate/migrate/v4 v4.7.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/google/btree v1.0.1 // indirect
	github.com/google/pprof v0.0.0-20211122183932-1daafda22083 // indirect
	github.com/googleapis/gax-go/v2 v2.1.1 // indirect
	github.com/gorilla/handlers v1.5.1 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.0.0-rc.2.0.20201207153454-9f6bf00c00a7 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.16.0 // indirect
	github.com/hailocab/go-hostpool v0.0.0-20160125115350-e80d13ce29ed // indirect
	github.com/hashicorp/consul/api v1.11.0 // indirect
	github.com/hashicorp/errwrap v1.0.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-msgpack v0.5.5 // indirect
	github.com/hashicorp/go-multierror v1.1.0 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/go-sockaddr v1.0.2 // indirect
	github.com/hashicorp/go-uuid v1.0.2 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/hashicorp/memberlist v0.2.4 // indirect
	github.com/hashicorp/serf v0.9.5 // indirect
	github.com/hashicorp/yamux v0.0.0-20190923154419-df201c70410d // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/jcmturner/aescts/v2 v2.0.0 // indirect
	github.com/jcmturner/dnsutils/v2 v2.0.0 // indirect
	github.com/jcmturner/gofork v1.0.0 // indirect
	github.com/jcmturner/gokrb5/v8 v8.4.2 // indirect
	github.com/jcmturner/rpc/v2 v2.0.3 // indirect
	github.com/jessevdk/go-flags v1.5.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/jpillora/backoff v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/julienschmidt/httprouter v1.3.0 // indirect
	github.com/klauspost/cpuid v1.3.1 // indirect
	github.com/knadh/koanf v1.3.3 // indirect
	github.com/lann/builder v0.0.0-20180802200727-47ae307949d0 // indirect
	github.com/lann/ps v0.0.0-20150810152359-62de8c46ede0 // indirect
	github.com/lib/pq v1.3.0 // indirect
	github.com/magiconair/properties v1.8.5 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.8 // indirect
	github.com/mattn/go-ieproxy v0.0.1 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/miekg/dns v1.1.43 // indirect
	github.com/minio/md5-simd v1.1.0 // indirect
	github.com/minio/sha256-simd v0.1.1 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/go-testing-interface v1.0.0 // indirect
	github.com/mitchellh/mapstructure v1.4.3 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/mostynb/go-grpc-compression v1.1.15 // indirect
	github.com/mwitkow/go-conntrack v0.0.0-20190716064945-2f068394615f // indirect
	github.com/ncw/swift v1.0.52 // indirect
	github.com/oklog/run v1.1.0 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/kafkaexporter v0.41.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal v0.41.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/sharedcomponent v0.41.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/opencensus v0.41.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/zipkin v0.41.0 // indirect
	github.com/opentracing-contrib/go-stdlib v1.0.0 // indirect
	github.com/openzipkin/zipkin-go v0.3.0 // indirect
	github.com/pelletier/go-toml v1.9.4 // indirect
	github.com/pierrec/lz4 v2.6.1+incompatible // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/alertmanager v0.23.1-0.20210914172521-e35efbddb66a // indirect
	github.com/prometheus/common/sigv4 v0.1.0 // indirect
	github.com/prometheus/exporter-toolkit v0.7.0 // indirect
	github.com/prometheus/node_exporter v1.0.0-rc.0.0.20200428091818-01054558c289 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475 // indirect
	github.com/rs/cors v1.8.0 // indirect
	github.com/rs/xid v1.2.1 // indirect
	github.com/sean-/seed v0.0.0-20170313163322-e2103e2c3529 // indirect
	github.com/sercand/kuberesolver v2.4.0+incompatible // indirect
	github.com/shurcooL/httpfs v0.0.0-20190707220628-8d4bc4ba7749 // indirect
	github.com/shurcooL/vfsgen v0.0.0-20200824052919-0d455de96546 // indirect
	github.com/soheilhy/cmux v0.1.5 // indirect
	github.com/sony/gobreaker v0.4.1 // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/spf13/afero v1.6.0 // indirect
	github.com/spf13/cast v1.4.1 // indirect
	github.com/spf13/cobra v1.2.1 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stretchr/objx v0.2.0 // indirect
	github.com/subosito/gotenv v1.2.0 // indirect
	github.com/uber/jaeger-lib v2.4.1+incompatible // indirect
	github.com/weaveworks/promrus v1.2.0 // indirect
	github.com/willf/bitset v1.1.11 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.0.2 // indirect
	github.com/xdg-go/stringprep v1.0.2 // indirect
	go.etcd.io/bbolt v1.3.6 // indirect
	go.etcd.io/etcd v3.3.25+incompatible // indirect
	go.etcd.io/etcd/api/v3 v3.5.0 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.5.0 // indirect
	go.etcd.io/etcd/client/v3 v3.5.0 // indirect
	go.mongodb.org/mongo-driver v1.7.5 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.27.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.27.0 // indirect
	go.opentelemetry.io/otel/internal/metric v0.25.0 // indirect
	go.opentelemetry.io/otel/sdk/export/metric v0.25.0 // indirect
	golang.org/x/crypto v0.0.0-20210921155107-089bfa567519 // indirect
	golang.org/x/net v0.0.0-20211123203042-d83791d6bcd9 // indirect
	golang.org/x/oauth2 v0.0.0-20211104180415-d3ed0bb246c8 // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c // indirect
	golang.org/x/sys v0.0.0-20211124211545-fe61309f8881 // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20211206160659-862468c7d6e0 // indirect
	gopkg.in/alecthomas/kingpin.v2 v2.2.6 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.63.2 // indirect
	rsc.io/binaryregexp v0.2.0 // indirect
)

// additional Cortex or upstream required replaces
replace (
	github.com/bradfitz/gomemcache => github.com/themihai/gomemcache v0.0.0-20180902122335-24332e2d58ab
	github.com/gocql/gocql => github.com/grafana/gocql v0.0.0-20200605141915-ba5dc39ece85
	k8s.io/api => k8s.io/api v0.20.4
	k8s.io/client-go => k8s.io/client-go v0.20.4
)

// Use a more recent version of go-redis that does not depend on OpenTelemetry anymore
replace github.com/go-redis/redis/v8 => github.com/go-redis/redis/v8 v8.11.0

// Pin to the latest release of grpc-go with GenerateAndRegisterManualResolver
// This function is used by jeagertracing/jaeger, but we can't update jaeger
// without updating the open-telemetry/collector as well
replace google.golang.org/grpc => google.golang.org/grpc v1.38.0

// Pin github.com/soheilhy/cmux to control grpc required version.
// Before v0.1.5 it contained examples in the root folder that imported grpc without a version,
// and therefore were importing grpc latest (which is problematic because we need <v1.29.1)
replace github.com/soheilhy/cmux => github.com/soheilhy/cmux v0.1.5

// Fix for injecting and extracting the propagation context
// Can be removed when https://github.com/open-telemetry/opentelemetry-go/pull/2141 is merged
replace go.opentelemetry.io/otel/bridge/opentracing => github.com/bboreham/opentelemetry-go/bridge/opentracing v0.13.1-0.20210728105346-81ca3b18d348

// Comment is from https://github.com/cortexproject/cortex/blob/07d6cfc4eb848fc410262773255dfec750c0a64f/go.mod#L83-L87
// We only pin this version to avoid problems with running go get: github.com/thanos-io/thanos@main. That
// currently fails because Thanos isn't merging release branches to main branch, and Go modules system is then
// confused about which version is the latest one. v0.22.0 was released in July, but latest tag reachable from main
// is v0.19.1. We pin version from early December here. Feel free to remove when updating to later version.
replace github.com/thanos-io/thanos v0.22.0 => github.com/thanos-io/thanos v0.19.1-0.20211208205607-d1acaea2a11a

// Replacing for an internal fork that exposes internal folders
// Some funtionalities of the collector have been made internal and it's more difficult to build and configure pipelines in the newer versions.
// This is a temporary solution while a new configuration design is discussed for the collector (ref: https://github.com/open-telemetry/opentelemetry-collector/issues/3482).
replace go.opentelemetry.io/collector => github.com/grafana/opentelemetry-collector v0.4.1-0.20211201123047-86905635e8f8
