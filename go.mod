// force pipeline trigger
module gitlab.com/nunet/device-management-service

go 1.25

require (
	github.com/NVIDIA/go-nvml v0.12.4-0
	github.com/aws/aws-sdk-go-v2 v1.32.7
	github.com/docker/docker v27.4.1+incompatible
	github.com/fatih/structs v1.1.0
	github.com/gin-contrib/cors v1.7.3
	github.com/gin-gonic/gin v1.10.0
	github.com/go-viper/mapstructure/v2 v2.3.0
	github.com/google/uuid v1.6.0
	github.com/hashicorp/go-version v1.7.0
	github.com/iancoleman/strcase v0.3.0
	github.com/libp2p/go-libp2p v0.43.0
	github.com/libp2p/go-libp2p-kad-dht v0.34.0
	github.com/libp2p/go-libp2p-pubsub v0.15.0
	github.com/manifoldco/promptui v0.9.0
	github.com/multiformats/go-multiaddr v0.16.1
	github.com/oschwald/geoip2-golang v1.11.0
	github.com/ostafen/clover/v2 v2.0.0-alpha.3
	github.com/quic-go/quic-go v0.54.1
	github.com/robfig/cron/v3 v3.0.1
	github.com/shirou/gopsutil/v4 v4.24.12
	github.com/shoenig/go-m1cpu v0.1.6
	github.com/songgao/water v0.0.0-20200317203138-2b4b6d7c09d8
	github.com/spf13/afero v1.12.0
	github.com/spf13/cobra v1.8.1
	github.com/spf13/viper v1.20.1
	github.com/stretchr/testify v1.10.0
	github.com/whyrusleeping/multiaddr-filter v0.0.0-20160516205228-e903e4adabd7
	go.uber.org/zap v1.27.0
	google.golang.org/protobuf v1.36.7
)

require (
	github.com/Microsoft/go-winio v0.6.2
	github.com/benbjohnson/clock v1.3.5 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bytedance/sonic v1.12.6 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/davidlazar/go-crypto v0.0.0-20200604182044-b73af7476f6c // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.4.0
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-units v0.5.0 // indirect
	github.com/dustin/go-humanize v1.0.1
	github.com/flynn/noise v1.1.0 // indirect
	github.com/francoispqt/gojay v1.2.13 // indirect
	github.com/fsnotify/fsnotify v1.8.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.9 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.23.0
	github.com/goccy/go-json v0.10.4 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/gopacket v1.1.19
	github.com/hashicorp/golang-lru v1.0.2 // indirect
	github.com/huin/goupnp v1.3.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/ipfs/boxo v0.33.1 // indirect
	github.com/ipfs/go-cid v0.5.0
	github.com/ipfs/go-datastore v0.8.2 // indirect
	github.com/ipfs/go-log/v2 v2.8.0
	github.com/ipld/go-ipld-prime v0.21.0 // indirect
	github.com/jackpal/go-nat-pmp v1.0.2 // indirect
	github.com/jbenet/go-temp-err-catcher v0.1.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/koron/go-ssdp v0.0.6 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/libp2p/go-buffer-pool v0.1.0 // indirect
	github.com/libp2p/go-cidranger v1.1.0 // indirect
	github.com/libp2p/go-flow-metrics v0.3.0 // indirect
	github.com/libp2p/go-libp2p-asn-util v0.4.1 // indirect
	github.com/libp2p/go-libp2p-kbucket v0.7.0
	github.com/libp2p/go-libp2p-record v0.3.1
	github.com/libp2p/go-msgio v0.3.0
	github.com/libp2p/go-netroute v0.2.2 // indirect
	github.com/libp2p/go-reuseport v0.4.0 // indirect
	github.com/mailru/easyjson v0.9.0 // indirect
	github.com/marten-seemann/tcp v0.0.0-20210406111302-dfbc87cc63fd // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/miekg/dns v1.1.68
	github.com/mikioh/tcpinfo v0.0.0-20190314235526-30a79bb1804b // indirect
	github.com/mikioh/tcpopt v0.0.0-20190314235656-172688c1accc // indirect
	github.com/minio/sha256-simd v1.0.1 // indirect
	github.com/moby/term v0.0.0-20220808134915-39b0c02b01ae // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/mr-tron/base58 v1.2.0 // indirect
	github.com/multiformats/go-base32 v0.1.0 // indirect
	github.com/multiformats/go-base36 v0.2.0 // indirect
	github.com/multiformats/go-multiaddr-dns v0.4.1 // indirect
	github.com/multiformats/go-multiaddr-fmt v0.1.0
	github.com/multiformats/go-multibase v0.2.0
	github.com/multiformats/go-multicodec v0.9.2 // indirect
	github.com/multiformats/go-multihash v0.2.3
	github.com/multiformats/go-multistream v0.6.1
	github.com/multiformats/go-varint v0.0.7
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.0.2
	github.com/pbnjay/memory v0.0.0-20210728143218-7b4eea64cf58 // indirect
	github.com/pelletier/go-toml/v2 v2.2.3 // indirect
	github.com/pkg/errors v0.9.1
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/polydawn/refmt v0.89.0 // indirect
	github.com/prometheus/client_golang v1.23.0 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.65.0 // indirect
	github.com/prometheus/procfs v0.17.0 // indirect
	github.com/quic-go/qpack v0.5.1 // indirect
	github.com/quic-go/webtransport-go v0.9.0 // indirect
	github.com/sagikazarmark/locafero v0.7.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/spf13/cast v1.7.1 // indirect
	github.com/spf13/pflag v1.0.6
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/tklauser/go-sysconf v0.3.13 // indirect
	github.com/tklauser/numcpus v0.7.0 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.2.12 // indirect
	github.com/whyrusleeping/go-keyspace v0.0.0-20160322163242-5b898ac5add1 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/otel v1.37.0 // indirect
	go.opentelemetry.io/otel/metric v1.37.0 // indirect
	go.opentelemetry.io/otel/trace v1.37.0 // indirect
	go.uber.org/dig v1.19.0 // indirect
	go.uber.org/fx v1.24.0 // indirect
	go.uber.org/multierr v1.11.0
	golang.org/x/arch v0.12.0 // indirect
	golang.org/x/crypto v0.41.0
	golang.org/x/exp v0.0.0-20250811191247-51f88131bc50
	golang.org/x/mod v0.27.0 // indirect
	golang.org/x/net v0.43.0 // indirect
	golang.org/x/sync v0.16.0
	golang.org/x/sys v0.35.0
	golang.org/x/text v0.28.0 // indirect
	golang.org/x/tools v0.36.0 // indirect
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.1
	lukechampine.com/blake3 v1.4.1 // indirect
)

require (
	github.com/aws/aws-sdk-go-v2/config v1.27.43
	github.com/aws/aws-sdk-go-v2/feature/s3/manager v1.17.31
	github.com/aws/aws-sdk-go-v2/service/s3 v1.65.2
)

require (
	github.com/Azure/go-ansiterm v0.0.0-20210617225240-d185dfc1b5a1 // indirect
	github.com/armon/go-radix v1.0.0 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.6.6 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.17.41 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.17 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.21 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.21 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.1 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.3.21 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.12.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.4.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.12.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.18.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.24.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.28.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.32.2 // indirect
	github.com/cespare/xxhash v1.1.0 // indirect
	github.com/dgraph-io/badger/v3 v3.2103.2
	github.com/dgraph-io/ristretto v0.1.0 // indirect
	github.com/elastic/go-licenser v0.3.1 // indirect
	github.com/elastic/go-sysinfo v1.7.1 // indirect
	github.com/elastic/go-windows v1.0.0 // indirect
	github.com/golang/glog v1.2.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/snappy v0.0.5-0.20220116011046-fa5810519dcb // indirect
	github.com/google/flatbuffers v2.0.6+incompatible // indirect
	github.com/google/orderedcode v0.0.1 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/jcchavezs/porto v0.1.0 // indirect
	github.com/joeshaw/multierror v0.0.0-20140124173710-69b34d4ec901 // indirect
	github.com/libp2p/go-libp2p-routing-helpers v0.7.5 // indirect
	github.com/lufia/plan9stats v0.0.0-20240226150601-1dcf7310316a // indirect
	github.com/pion/datachannel v1.5.10 // indirect
	github.com/pion/dtls/v2 v2.2.12 // indirect
	github.com/pion/interceptor v0.1.40 // indirect
	github.com/pion/logging v0.2.3 // indirect
	github.com/pion/randutil v0.1.0 // indirect
	github.com/pion/rtcp v1.2.15 // indirect
	github.com/pion/rtp v1.8.19 // indirect
	github.com/pion/sctp v1.8.39 // indirect
	github.com/pion/sdp/v3 v3.0.13 // indirect
	github.com/pion/stun v0.6.1 // indirect
	github.com/pion/transport/v2 v2.2.10 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/santhosh-tekuri/jsonschema v1.2.4 // indirect
	github.com/satori/go.uuid v1.2.0 // indirect
	github.com/vishvananda/netlink v1.2.1-beta.2
	github.com/vishvananda/netns v0.0.4 // indirect
	github.com/vmihailenco/msgpack/v5 v5.3.5 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	go.elastic.co/fastjson v1.1.0 // indirect
	go.etcd.io/bbolt v1.3.6 // indirect
	golang.org/x/lint v0.0.0-20201208152925-83fdc39ff7b5 // indirect
	golang.org/x/term v0.34.0
	gonum.org/v1/gonum v0.16.0 // indirect
)

require github.com/avast/retry-go v3.0.0+incompatible

require (
	github.com/compose-spec/compose-go/v2 v2.8.1
	github.com/cucumber/godog v0.15.0
	github.com/lxc/incus v0.7.0
	github.com/natefinch/lumberjack v2.0.0+incompatible
	github.com/olivere/elastic/v7 v7.0.32
	github.com/quic-go/connect-ip-go v0.1.0
	github.com/yosida95/uritemplate/v3 v3.0.2
	go.elastic.co/apm/module/apmgin/v2 v2.6.2
	go.elastic.co/apm/module/apmhttp v1.15.0
	go.elastic.co/apm/module/apmhttp/v2 v2.6.2
	go.elastic.co/apm/v2 v2.6.2
	go.yaml.in/yaml/v3 v3.0.4
)

require (
	github.com/aws/smithy-go v1.22.1 // indirect
	github.com/creack/pty v1.1.18 // indirect
	github.com/cucumber/gherkin/go/v26 v26.2.0 // indirect
	github.com/cucumber/messages/go/v21 v21.0.1 // indirect
	github.com/dunglas/httpsfv v1.1.0 // indirect
	github.com/filecoin-project/go-clock v0.1.0 // indirect
	github.com/gofrs/uuid v4.3.1+incompatible // indirect
	github.com/gorilla/schema v1.2.1 // indirect
	github.com/gorilla/securecookie v1.1.2 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-memdb v1.3.4 // indirect
	github.com/kr/fs v0.1.0 // indirect
	github.com/libp2p/go-yamux/v5 v5.0.1 // indirect
	github.com/mattn/go-shellwords v1.0.12 // indirect
	github.com/muhlemmer/gu v0.3.1 // indirect
	github.com/oschwald/maxminddb-golang v1.13.0 // indirect
	github.com/pion/dtls/v3 v3.0.6 // indirect
	github.com/pion/ice/v4 v4.0.10 // indirect
	github.com/pion/mdns/v2 v2.0.7 // indirect
	github.com/pion/srtp/v3 v3.0.6 // indirect
	github.com/pion/stun/v3 v3.0.0 // indirect
	github.com/pion/transport/v3 v3.0.7 // indirect
	github.com/pion/turn/v4 v4.0.2 // indirect
	github.com/pion/webrtc/v4 v4.1.2 // indirect
	github.com/pkg/sftp v1.13.7 // indirect
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.1 // indirect
	github.com/xhit/go-str2duration/v2 v2.1.0 // indirect
	github.com/zitadel/oidc/v2 v2.12.0 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.uber.org/mock v0.5.2 // indirect
	golang.org/x/oauth2 v0.30.0 // indirect
	golang.org/x/time v0.12.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
	gopkg.in/square/go-jose.v2 v2.6.0 // indirect
	howett.net/plist v0.0.0-20181124034731-591f970eefbb // indirect
)

require (
	github.com/bytedance/sonic/loader v0.2.1 // indirect
	github.com/chzyer/readline v1.5.1 // indirect
	github.com/cloudwego/base64x v0.1.4 // indirect
	github.com/cloudwego/iasm v0.2.0 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/ebitengine/purego v0.8.1 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/wlynxg/anet v0.0.5 // indirect
	go.elastic.co/apm v1.15.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.62.0 // indirect
)
