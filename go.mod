module github.com/kubeclipper/kubeclipper

go 1.20

require (
	github.com/Masterminds/sprig/v3 v3.2.2
	github.com/containerd/containerd v1.6.19
	github.com/coreos/go-oidc/v3 v3.6.0
	github.com/dlclark/regexp2 v1.10.0
	github.com/emicklei/go-restful v2.9.6+incompatible
	github.com/emicklei/go-restful-openapi v0.0.0-00010101000000-000000000000
	github.com/evanphx/json-patch v4.12.0+incompatible
	github.com/fatih/color v1.7.0
	github.com/go-logr/zapr v1.2.3
	github.com/go-openapi/loads v0.19.5
	github.com/go-openapi/spec v0.19.7
	github.com/go-openapi/strfmt v0.19.5
	github.com/go-openapi/validate v0.19.8
	github.com/go-redis/redis/v8 v8.11.5
	github.com/golang-jwt/jwt/v4 v4.2.0
	github.com/golang/mock v1.6.0
	github.com/google/go-containerregistry v0.12.1
	github.com/google/uuid v1.3.0
	github.com/gorilla/websocket v1.4.2
	github.com/minio/minio-go/v7 v7.0.61
	github.com/mitchellh/mapstructure v1.4.1
	github.com/moby/ipvs v1.0.1
	github.com/nats-io/nats-server/v2 v2.4.0
	github.com/nats-io/nats.go v1.12.0
	github.com/nicksnyder/go-i18n/v2 v2.2.0
	github.com/olekukonko/tablewriter v0.0.5
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.24.1
	github.com/open-policy-agent/opa v0.54.0
	github.com/pelletier/go-toml v1.9.5
	github.com/pkg/errors v0.9.1
	github.com/pkg/sftp v1.10.1
	github.com/prometheus/client_golang v1.16.0
	github.com/robfig/cron/v3 v3.0.1
	github.com/sethvargo/go-password v0.2.0
	github.com/shirou/gopsutil/v3 v3.21.10
	github.com/sirupsen/logrus v1.9.3
	github.com/spf13/cobra v1.7.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.8.1
	github.com/stretchr/testify v1.8.2
	github.com/subosito/gotenv v1.2.0
	github.com/txn2/txeh v1.4.0
	github.com/vbauerster/mpb/v8 v8.0.2
	github.com/vishvananda/netlink v1.1.1-0.20210330154013-f5de75959ad5
	go.uber.org/zap v1.24.0
	golang.org/x/crypto v0.11.0
	golang.org/x/oauth2 v0.10.0
	golang.org/x/sys v0.10.0
	golang.org/x/term v0.10.0
	golang.org/x/text v0.11.0
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	k8s.io/api v0.26.4
	k8s.io/apimachinery v0.26.4
	k8s.io/apiserver v0.26.4
	k8s.io/client-go v0.26.4
	k8s.io/component-base v0.26.4
	k8s.io/klog/v2 v2.100.1
	k8s.io/utils v0.0.0-20221128185143-99ec85e7a448
	sigs.k8s.io/yaml v1.3.0
)

require (
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.1.1 // indirect
	github.com/Microsoft/go-winio v0.6.0 // indirect
	github.com/Microsoft/hcsshim v0.9.7 // indirect
	github.com/NYTimes/gziphandler v1.1.1 // indirect
	github.com/OneOfOne/xxhash v1.2.8 // indirect
	github.com/PuerkitoBio/purell v1.1.1 // indirect
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578 // indirect
	github.com/StackExchange/wmi v1.2.1 // indirect
	github.com/VividCortex/ewma v1.2.0 // indirect
	github.com/acarl005/stripansi v0.0.0-20180116102854-5a71ef0e047d // indirect
	github.com/agnivade/levenshtein v1.1.1 // indirect
	github.com/antlr/antlr4/runtime/Go/antlr v1.4.10 // indirect
	github.com/asaskevich/govalidator v0.0.0-20200108200545-475eaeb16496 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/cenkalti/backoff/v4 v4.2.0 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/containerd/cgroups v1.0.4 // indirect
	github.com/containerd/continuity v0.3.0 // indirect
	github.com/containerd/fifo v1.0.0 // indirect
	github.com/containerd/stargz-snapshotter/estargz v0.12.1 // indirect
	github.com/containerd/ttrpc v1.1.0 // indirect
	github.com/containerd/typeurl v1.0.2 // indirect
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/go-systemd/v22 v22.3.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/docker/cli v20.10.20+incompatible // indirect
	github.com/docker/distribution v2.8.1+incompatible // indirect
	github.com/docker/docker v20.10.20+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.7.0 // indirect
	github.com/docker/go-events v0.0.0-20190806004212-e31b211e4f1c // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/emicklei/go-restful/v3 v3.9.0 // indirect
	github.com/felixge/httpsnoop v1.0.3 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-ini/ini v1.67.0 // indirect
	github.com/go-jose/go-jose/v3 v3.0.0 // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-openapi/analysis v0.19.10 // indirect
	github.com/go-openapi/errors v0.19.3 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.20.0 // indirect
	github.com/go-openapi/runtime v0.19.4 // indirect
	github.com/go-openapi/swag v0.19.14 // indirect
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/gogo/googleapis v1.4.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/cel-go v0.12.6 // indirect
	github.com/google/gnostic v0.5.7-v3refs // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/gopherjs/gopherjs v0.0.0-20191106031601-ce3c9ade29de // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.7.0 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/huandu/xstrings v1.3.1 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.16.7 // indirect
	github.com/klauspost/cpuid/v2 v2.2.5 // indirect
	github.com/kr/fs v0.1.0 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/magiconair/properties v1.8.5 // indirect
	github.com/mailru/easyjson v0.7.6 // indirect
	github.com/mattn/go-colorable v0.1.9 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/minio/highwayhash v1.0.1 // indirect
	github.com/minio/md5-simd v1.1.2 // indirect
	github.com/minio/sha256-simd v1.0.1 // indirect
	github.com/mitchellh/copystructure v1.0.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.0 // indirect
	github.com/moby/locker v1.0.1 // indirect
	github.com/moby/spdystream v0.2.0 // indirect
	github.com/moby/sys/mountinfo v0.6.2 // indirect
	github.com/moby/sys/signal v0.6.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/mxk/go-flowrate v0.0.0-20140419014527-cca7078d478f // indirect
	github.com/nats-io/jwt/v2 v2.0.3 // indirect
	github.com/nats-io/nkeys v0.3.0 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/nxadm/tail v1.4.8 // indirect
	github.com/onsi/ginkgo/v2 v2.6.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0-rc2 // indirect
	github.com/opencontainers/runc v1.1.4 // indirect
	github.com/opencontainers/runtime-spec v1.0.3-0.20210326190908-1c3f411f0417 // indirect
	github.com/opencontainers/selinux v1.10.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.3.0 // indirect
	github.com/prometheus/common v0.42.0 // indirect
	github.com/prometheus/procfs v0.10.1 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20200313005456-10cdbea86bc0 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/rs/xid v1.5.0 // indirect
	github.com/shopspring/decimal v1.2.0 // indirect
	github.com/smartystreets/assertions v1.0.1 // indirect
	github.com/spf13/afero v1.6.0 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/stoewer/go-strcase v1.2.0 // indirect
	github.com/tchap/go-patricia/v2 v2.3.1 // indirect
	github.com/tklauser/go-sysconf v0.3.9 // indirect
	github.com/tklauser/numcpus v0.3.0 // indirect
	github.com/vbatts/tar-split v0.11.2 // indirect
	github.com/vishvananda/netns v0.0.0-20210104183010-2eb08e3e575f // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/yashtewari/glob-intersection v0.2.0 // indirect
	go.etcd.io/etcd/api/v3 v3.5.5 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.5.5 // indirect
	go.etcd.io/etcd/client/v3 v3.5.5 // indirect
	go.mongodb.org/mongo-driver v1.3.0 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.35.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.37.0 // indirect
	go.opentelemetry.io/otel v1.14.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/internal/retry v1.14.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.14.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.14.0 // indirect
	go.opentelemetry.io/otel/metric v0.34.0 // indirect
	go.opentelemetry.io/otel/sdk v1.14.0 // indirect
	go.opentelemetry.io/otel/trace v1.14.0 // indirect
	go.opentelemetry.io/proto/otlp v0.19.0 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	golang.org/x/mod v0.8.0 // indirect
	golang.org/x/net v0.12.0 // indirect
	golang.org/x/sync v0.2.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	golang.org/x/tools v0.6.0 // indirect
	golang.org/x/xerrors v0.0.0-20220907171357-04be3eba64a2 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1 // indirect
	google.golang.org/grpc v1.56.1 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/kms v0.26.4 // indirect
	k8s.io/kube-openapi v0.0.0-20221012153701-172d655c2280 // indirect
	sigs.k8s.io/apiserver-network-proxy/konnectivity-client v0.0.36 // indirect
	sigs.k8s.io/json v0.0.0-20220713155537-f223a00ba0e2 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3 // indirect
)

replace (
	cloud.google.com/go => cloud.google.com/go v0.110.0
	github.com/coreos/bbolt => github.com/coreos/bbolt v1.3.2
	github.com/coreos/go-systemd => github.com/coreos/go-systemd v0.0.0-20190321100706-95778dfbb74e
	github.com/coreos/go-systemd/v22 => github.com/coreos/go-systemd/v22 v22.3.2
	github.com/coreos/pkg => github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f
	github.com/emicklei/go-restful => github.com/emicklei/go-restful v2.14.3+incompatible
	github.com/emicklei/go-restful-openapi => github.com/emicklei/go-restful-openapi v1.4.1
	github.com/evanphx/json-patch => github.com/evanphx/json-patch v4.11.0+incompatible
	github.com/fatih/color => github.com/fatih/color v1.13.0
	github.com/go-openapi/analysis => github.com/go-openapi/analysis v0.19.10
	github.com/go-openapi/errors => github.com/go-openapi/errors v0.19.4
	github.com/go-openapi/jsonpointer => github.com/go-openapi/jsonpointer v0.19.3
	github.com/go-openapi/jsonreference => github.com/go-openapi/jsonreference v0.19.3
	github.com/go-openapi/loads => github.com/go-openapi/loads v0.19.5
	github.com/go-openapi/runtime => github.com/go-openapi/runtime v0.19.15
	github.com/go-openapi/spec => github.com/go-openapi/spec v0.19.7
	github.com/go-openapi/strfmt => github.com/go-openapi/strfmt v0.19.5
	github.com/go-openapi/swag => github.com/go-openapi/swag v0.19.9
	github.com/go-openapi/validate => github.com/go-openapi/validate v0.19.8
	github.com/golang-jwt/jwt/v4 => github.com/golang-jwt/jwt/v4 v4.1.0
	github.com/golang/mock => github.com/golang/mock v1.5.0
	github.com/google/uuid => github.com/google/uuid v1.1.2
	github.com/mattn/go-colorable => github.com/mattn/go-colorable v0.1.12
	github.com/nats-io/nats-server/v2 => github.com/nats-io/nats-server/v2 v2.4.0
	github.com/nats-io/nats.go => github.com/nats-io/nats.go v1.12.0
	github.com/olekukonko/tablewriter => github.com/olekukonko/tablewriter v0.0.5
	github.com/onsi/ginkgo => github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega => github.com/onsi/gomega v1.10.1
	github.com/pkg/errors => github.com/pkg/errors v0.9.1
	github.com/pkg/sftp => github.com/pkg/sftp v1.11.0
	github.com/shirou/gopsutil/v3 => github.com/shirou/gopsutil/v3 v3.21.10
	github.com/spf13/cobra => github.com/spf13/cobra v1.2.1
	github.com/spf13/pflag => github.com/spf13/pflag v1.0.5
	github.com/spf13/viper => github.com/spf13/viper v1.8.1
	github.com/stretchr/testify => github.com/stretchr/testify v1.7.0
	github.com/vishvananda/netlink => github.com/vishvananda/netlink v1.1.0
	github.com/vishvananda/netns => github.com/vishvananda/netns v0.0.0-20200728191858-db3c7e526aae
	go.uber.org/zap => go.uber.org/zap v1.17.0
	google.golang.org/grpc => google.golang.org/grpc v1.38.0
	google.golang.org/protobuf => google.golang.org/protobuf v1.26.0
	gopkg.in/ini.v1 => gopkg.in/ini.v1 v1.62.0
	gopkg.in/natefinch/lumberjack.v2 => gopkg.in/natefinch/lumberjack.v2 v2.0.0
	gopkg.in/yaml.v2 => gopkg.in/yaml.v2 v2.4.0
	k8s.io/api => k8s.io/api v0.26.4
	k8s.io/apimachinery => k8s.io/apimachinery v0.26.4
	k8s.io/apiserver => k8s.io/apiserver v0.26.4
	k8s.io/client-go => k8s.io/client-go v0.26.4
	k8s.io/component-base => k8s.io/component-base v0.26.4
)
