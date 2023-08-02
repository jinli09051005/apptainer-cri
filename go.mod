module github.com/sylabs/singularity-cri

go 1.19

require (
	github.com/NVIDIA/gpu-monitoring-tools v0.0.0-20190227022151-81c885550fa1
	github.com/apptainer/apptainer v1.2.2
	github.com/containerd/cgroups v1.1.0
	github.com/containernetworking/cni v1.1.2
	github.com/creack/pty v1.1.18
	github.com/fsnotify/fsnotify v1.4.9
	github.com/golang/glog v1.1.0
	github.com/kr/pty v1.1.8
	github.com/kubernetes-sigs/cri-o v1.12.3
	github.com/opencontainers/image-spec v1.1.0-rc4
	github.com/opencontainers/runc v1.1.7
	github.com/opencontainers/runtime-spec v1.1.0
	github.com/opencontainers/runtime-tools v0.9.1-0.20221107090550-2e043c6bd626
	github.com/opencontainers/selinux v1.11.0
	github.com/stretchr/testify v1.8.2
	github.com/sylabs/scs-library-client v0.4.4
	github.com/sylabs/singularity v0.0.0-20190918134918-5d9975e95fa7
	github.com/tchap/go-patricia v2.2.6+incompatible
	golang.org/x/sys v0.10.0
	google.golang.org/grpc v1.55.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/client-go v0.0.0-20181010045704-56e7a63b5e38
	k8s.io/kubernetes v1.12.5
)

require (
	github.com/apptainer/sif/v2 v2.11.5 // indirect
	github.com/blang/semver v3.5.1+incompatible // indirect
	github.com/containernetworking/plugins v1.3.0 // indirect
	github.com/containers/storage v1.46.0 // indirect
	github.com/coreos/go-iptables v0.6.0 // indirect
	github.com/coreos/go-systemd v0.0.0-20180511133405-39ca1b05acc7 // indirect
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/cyphar/filepath-securejoin v0.2.3 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/elazarl/goproxy v0.0.0-20181111060418-2ce16c963a8a // indirect
	github.com/emicklei/go-restful v2.8.0+incompatible // indirect
	github.com/fatih/color v1.15.0 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-log/log v0.2.0 // indirect
	github.com/godbus/dbus v4.1.0+incompatible // indirect
	github.com/godbus/dbus/v5 v5.1.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/gofuzz v1.0.0 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.17 // indirect
	github.com/moby/sys/mountinfo v0.6.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/safchain/ethtool v0.3.0 // indirect
	github.com/satori/go.uuid v1.2.0 // indirect
	github.com/seccomp/containers-golang v0.6.0 // indirect
	github.com/seccomp/libseccomp-golang v0.10.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/sylabs/json-resp v0.9.0 // indirect
	github.com/sylabs/scs-key-client v0.3.0-0.20190509220229-bce3b050c4ec // indirect
	github.com/sylabs/sif v1.0.8 // indirect
	github.com/syndtr/gocapability v0.0.0-20200815063812-42c35b437635 // indirect
	github.com/vishvananda/netlink v1.2.1-beta.2 // indirect
	github.com/vishvananda/netns v0.0.4 // indirect
	golang.org/x/crypto v0.11.0 // indirect
	golang.org/x/net v0.11.0 // indirect
	golang.org/x/sync v0.1.0 // indirect
	golang.org/x/text v0.11.0 // indirect
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4 // indirect
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/api v0.0.0-20181121071145-b7bd5f2d334c // indirect
	k8s.io/apimachinery v0.0.0-20181126123124-70adfbae261e // indirect
	k8s.io/apiserver v0.0.0-20181121231732-e3c8fa95bba5 // indirect
	k8s.io/klog v0.2.0 // indirect
	k8s.io/utils v0.0.0-20181115163542-0d26856f57b3 // indirect
)

replace (
	github.com/sylabs/json-resp v0.9.0 => github.com/sylabs/json-resp v0.6.0
	github.com/sylabs/scs-key-client v0.3.0-0.20190509220229-bce3b050c4ec => github.com/sylabs/scs-key-client v0.3.1-0.20190509220229-bce3b050c4ec
	golang.org/x/crypto => github.com/sylabs/golang-x-crypto v0.0.0-20181006204705-4bce89e8e9a9
)
