module github.com/platform9/vjailbreak/pkg/vpwned

go 1.24.10

replace github.com/platform9/vjailbreak/pkg/common/utils => ../common/utils

replace github.com/platform9/vjailbreak/pkg/common/validation => ../common/validation

replace github.com/platform9/vjailbreak/pkg/common/openstack => ../common/openstack

require (
	github.com/bougou/go-ipmi v0.7.6
	github.com/canonical/gomaasclient v0.12.0
	github.com/devans10/pugo/flasharray v0.0.0-20241116160615-6bb8c469c9a0
	github.com/google/go-github/v63 v63.0.0
	github.com/gophercloud/gophercloud/v2 v2.9.0
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.3
	github.com/juju/errors v1.0.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/pkg/errors v0.9.1
	github.com/platform9/vjailbreak/k8s/migration v0.0.0-20251203111109-fd5964e9ea7c
	github.com/platform9/vjailbreak/pkg/common/utils v0.0.0-00010101000000-000000000000
	github.com/platform9/vjailbreak/pkg/common/validation v0.0.0-00010101000000-000000000000
	github.com/sirupsen/logrus v1.9.3
	github.com/spf13/cobra v1.9.1
	github.com/spf13/viper v1.20.1
	github.com/vmware/govmomi v0.51.0
	go.uber.org/mock v0.5.1
	golang.org/x/mod v0.29.0
	google.golang.org/genproto/googleapis/api v0.0.0-20250929231259-57b25ae835d4
	google.golang.org/grpc v1.75.1
	google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.5.1
	google.golang.org/protobuf v1.36.10
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.33.3
	k8s.io/apiextensions-apiserver v0.33.0
	k8s.io/apimachinery v0.33.3
	k8s.io/client-go v0.33.1
	k8s.io/klog/v2 v2.130.1
	sigs.k8s.io/controller-runtime v0.21.0
	sigs.k8s.io/yaml v1.5.0
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/emicklei/go-restful/v3 v3.12.2 // indirect
	github.com/evanphx/json-patch/v5 v5.9.11 // indirect
	github.com/fsnotify/fsnotify v1.8.0 // indirect
	github.com/fxamacker/cbor/v2 v2.9.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/zapr v1.3.0 // indirect
	github.com/go-openapi/jsonpointer v0.21.1 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/swag v0.23.1 // indirect
	github.com/go-viper/mapstructure/v2 v2.2.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/google/btree v1.1.3 // indirect
	github.com/google/gnostic-models v0.6.9 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.7 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/juju/collections v1.0.4 // indirect
	github.com/juju/gomaasapi/v2 v2.3.0 // indirect
	github.com/juju/loggo v1.0.0 // indirect
	github.com/juju/mgo/v2 v2.0.2 // indirect
	github.com/juju/schema v1.2.0 // indirect
	github.com/juju/version v0.0.0-20210303051006-2015802527a8 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/mailru/easyjson v0.9.0 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/olekukonko/tablewriter v1.0.5 // indirect
	github.com/pelletier/go-toml/v2 v2.2.3 // indirect
	github.com/platform9/vjailbreak/pkg/common/openstack v0.0.0-00010101000000-000000000000 // indirect
	github.com/platform9/vjailbreak/v2v-helper v0.0.0-20250718102048-de8740c10909 // indirect
	github.com/prometheus/client_golang v1.22.0 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.62.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/sagikazarmark/locafero v0.7.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.12.0 // indirect
	github.com/spf13/cast v1.7.1 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/oauth2 v0.30.0 // indirect
	golang.org/x/sync v0.18.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/term v0.37.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	golang.org/x/time v0.12.0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.4.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250929231259-57b25ae835d4 // indirect
	gopkg.in/evanphx/json-patch.v4 v4.12.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/mgo.v2 v2.0.0-20190816093944-a6b53ec6cb22 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/kube-openapi v0.0.0-20250610211856-8b98d1ed966a // indirect
	k8s.io/utils v0.0.0-20250604170112-4c0f3b243397 // indirect
	sigs.k8s.io/json v0.0.0-20241014173422-cfa47c3a1cc8 // indirect
	sigs.k8s.io/randfill v1.0.0 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.7.0 // indirect
)

replace github.com/bougou/go-ipmi => github.com/bougou/go-ipmi v0.7.4

replace github.com/olekukonko/tablewriter => github.com/olekukonko/tablewriter v0.0.5

replace github.com/platform9/vjailbreak/v2v-helper => ../../v2v-helper

replace github.com/platform9/vjailbreak/pkg/common/validation => ../common/validation

replace github.com/platform9/vjailbreak/k8s/migration => ../../k8s/migration

replace github.com/platform9/vjailbreak/pkg/common/openstack => ../common/openstack
