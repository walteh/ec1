module github.com/walteh/ec1

go 1.24.3

replace (
	github.com/Code-Hex/vz/v3 => ../vz
	github.com/containerd/containerd/api => ../containerd/api
	github.com/containerd/containerd/v2 => ../containerd
	github.com/containers/gvisor-tap-vsock => ../gvisor-tap-vsock
	github.com/kata-containers/kata-containers/src/runtime => ../kata-containers/src/runtime
	github.com/lmittmann/tint => ../tint
)

replace gvisor.dev/gvisor => gvisor.dev/gvisor v0.0.0-20250509002459-06cdc4c49840

require (
	connectrpc.com/connect v1.18.1
	github.com/Code-Hex/vz/v3 v3.6.0
	github.com/Microsoft/hcsshim v0.13.0
	github.com/apex/log v1.9.0
	github.com/cavaliergopher/grab/v3 v3.0.1
	github.com/containers/common v0.63.0
	github.com/containers/gvisor-tap-vsock v0.8.6
	github.com/containers/image/v5 v5.35.0
	github.com/coreos/ignition/v2 v2.21.0
	github.com/crc-org/vfkit v0.6.2-0.20250415145558-4b7cae94e86a
	github.com/dustin/go-humanize v1.0.1
	github.com/fatih/color v1.18.0
	github.com/gin-gonic/gin v1.10.1
	github.com/go-openapi/errors v0.22.1
	github.com/go-openapi/loads v0.22.0
	github.com/go-openapi/runtime v0.28.0
	github.com/go-openapi/spec v0.21.0
	github.com/go-openapi/strfmt v0.23.0
	github.com/go-openapi/swag v0.23.1
	github.com/go-openapi/validate v0.24.0
	github.com/google/cel-go v0.25.0
	github.com/kata-containers/kata-containers/src/runtime v0.0.0-00010101000000-000000000000
	github.com/kdomanski/iso9660 v0.4.0
	github.com/lima-vm/go-qcow2reader v0.6.0
	github.com/lmittmann/tint v1.1.0
	github.com/mdlayher/vsock v1.2.1
	github.com/mholt/archives v0.1.2
	github.com/opencontainers/image-spec v1.1.1
	github.com/pkg/term v1.1.0
	github.com/prashantgupta24/mac-sleep-notifier v1.0.1
	github.com/prometheus/procfs v0.16.1
	github.com/rs/xid v1.6.0
	github.com/sirupsen/logrus v1.9.4-0.20230606125235-dd1b4c2e81af
	github.com/soheilhy/cmux v0.1.5
	github.com/spf13/pflag v1.0.6
	github.com/stretchr/testify v1.10.0
	github.com/superblocksteam/run v0.0.7
	github.com/u-root/u-root v0.14.0
	github.com/veqryn/slog-context v0.8.0
	github.com/walteh/retab/v2 v2.10.2
	github.com/walteh/run v0.0.0-20250510150917-6f8074766f03
	gitlab.com/tozd/go/errors v0.10.0
	go.pdmccormick.com/initramfs v0.1.2
	golang.org/x/crypto v0.38.0
	golang.org/x/mod v0.24.0
	golang.org/x/net v0.40.0
	golang.org/x/sync v0.14.0
	golang.org/x/sys v0.33.0
	google.golang.org/grpc v1.72.2
	google.golang.org/protobuf v1.36.6
	gvisor.dev/gvisor v0.0.0-20250509002459-06cdc4c49840
	kraftkit.sh v0.11.6
)

require (
	cel.dev/expr v0.23.1 // indirect
	github.com/Code-Hex/go-infinity-channel v1.0.0 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/STARRY-S/zip v0.2.1 // indirect
	github.com/andybalholm/brotli v1.1.2-0.20250424173009-453214e765f3 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.0 // indirect
	github.com/apparentlymart/go-cidr v1.1.0 // indirect
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/aws/aws-sdk-go v1.55.6 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver v3.5.1+incompatible // indirect
	github.com/bodgit/plumbing v1.3.0 // indirect
	github.com/bodgit/sevenzip v1.6.0 // indirect
	github.com/bodgit/windows v1.0.1 // indirect
	github.com/bytedance/sonic v1.11.6 // indirect
	github.com/bytedance/sonic/loader v0.1.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cilium/ebpf v0.18.0 // indirect
	github.com/cloudwego/base64x v0.1.4 // indirect
	github.com/cloudwego/iasm v0.2.0 // indirect
	github.com/containerd/cgroups v1.1.0 // indirect
	github.com/containerd/cgroups/v3 v3.0.5 // indirect
	github.com/containerd/console v1.0.4 // indirect
	github.com/containerd/fifo v1.1.0 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/containerd/ttrpc v1.2.7 // indirect
	github.com/containernetworking/plugins v1.7.1 // indirect
	github.com/containers/libtrust v0.0.0-20230121012942-c1716e8a8d01 // indirect
	github.com/containers/ocicrypt v1.2.1 // indirect
	github.com/containers/podman/v4 v4.9.4 // indirect
	github.com/containers/storage v1.58.0 // indirect
	github.com/coreos/go-semver v0.3.1 // indirect
	github.com/coreos/go-systemd/v22 v22.5.1-0.20231103132048-7d375ecc2b09 // indirect
	github.com/coreos/vcontext v0.0.0-20230201181013-d72178a18687 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/docker/docker v28.0.4+incompatible // indirect
	github.com/docker/go-connections v0.5.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/dsnet/compress v0.0.2-0.20230904184137-39efe44ab707 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.8 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-ini/ini v1.67.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/analysis v0.23.0 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.20.0 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/godbus/dbus/v5 v5.1.1-0.20230522191255-76236955d466 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8 // indirect
	github.com/google/btree v1.1.2 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/gopacket v1.1.19 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/insomniacslk/dhcp v0.0.0-20240829085014-a3a4c1f04475 // indirect
	github.com/intel-go/cpuid v0.0.0-20210602155658-5747e5cec0d9 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.18.1-0.20250502091416-dee68d8e897e // indirect
	github.com/klauspost/cpuid/v2 v2.2.10 // indirect
	github.com/klauspost/pgzip v1.2.6 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/linuxkit/virtsock v0.0.0-20220523201153-1a23e78aa7a2 // indirect
	github.com/mailru/easyjson v0.9.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mdlayher/socket v0.5.1 // indirect
	github.com/miekg/dns v1.1.65 // indirect
	github.com/minio/minlz v1.0.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/moby/sys/userns v0.1.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/nwaples/rardecode/v2 v2.1.0 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/opencontainers/cgroups v0.0.2 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/runc v1.3.0 // indirect
	github.com/opencontainers/runtime-spec v1.2.1 // indirect
	github.com/opencontainers/runtime-tools v0.9.1-0.20250303011046-260e151b8552 // indirect
	github.com/opencontainers/selinux v1.12.0 // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/pierrec/lz4/v4 v4.1.22 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_golang v1.22.0 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.62.0 // indirect
	github.com/safchain/ethtool v0.5.10 // indirect
	github.com/sergi/go-diff v1.3.2-0.20230802210424-5b0b94c5c0d3 // indirect
	github.com/sorairolake/lzip-go v0.3.5 // indirect
	github.com/sourcegraph/go-diff v0.7.0 // indirect
	github.com/stoewer/go-strcase v1.3.0 // indirect
	github.com/syndtr/gocapability v0.0.0-20200815063812-42c35b437635 // indirect
	github.com/therootcompany/xz v1.0.1 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/u-root/uio v0.0.0-20240224005618-d2acac8f3701 // indirect
	github.com/ugorji/go/codec v1.2.12 // indirect
	github.com/ulikunitz/xz v0.5.12 // indirect
	github.com/vincent-petithory/dataurl v1.0.0 // indirect
	github.com/vishvananda/netlink v1.3.1-0.20250303224720-0e7078ed04c8 // indirect
	github.com/vishvananda/netns v0.0.5 // indirect
	github.com/walteh/yaml v0.0.0-20250409173318-a722555a2a54 // indirect
	gitlab.com/nvidia/cloud-native/go-nvlib v0.0.0-20220601114329-47893b162965 // indirect
	go.mongodb.org/mongo-driver v1.17.2 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel v1.35.0 // indirect
	go.opentelemetry.io/otel/exporters/jaeger v1.0.0 // indirect
	go.opentelemetry.io/otel/metric v1.35.0 // indirect
	go.opentelemetry.io/otel/sdk v1.35.0 // indirect
	go.opentelemetry.io/otel/trace v1.35.0 // indirect
	go4.org v0.0.0-20230225012048-214862532bf5 // indirect
	golang.org/x/arch v0.8.0 // indirect
	golang.org/x/exp v0.0.0-20250210185358-939b2ce775ac // indirect
	golang.org/x/oauth2 v0.29.0 // indirect
	golang.org/x/text v0.25.0 // indirect
	golang.org/x/time v0.11.0 // indirect
	golang.org/x/tools v0.33.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250425173222-7b384671a197 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250428153025-10db94c68c34 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	sigs.k8s.io/yaml v1.4.0 // indirect
	tags.cncf.io/container-device-interface v1.0.1 // indirect
	tags.cncf.io/container-device-interface/specs-go v1.0.0 // indirect
)
