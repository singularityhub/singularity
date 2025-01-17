module github.com/sylabs/singularity

go 1.16

require (
	github.com/AdamKorcz/go-fuzz-headers v0.0.0-20210319161527-f761c2329661 // indirect
	github.com/Netflix/go-expect v0.0.0-20190729225929-0e00d9168667
	github.com/ProtonMail/go-crypto v0.0.0-20220113124808-70ae35bab23f
	github.com/adigunhammedolalekan/registry-auth v0.0.0-20200730122110-8cde180a3a60
	github.com/apex/log v1.9.0
	github.com/blang/semver/v4 v4.0.0
	github.com/buger/jsonparser v1.1.1
	github.com/bugsnag/bugsnag-go v1.5.1 // indirect
	github.com/bugsnag/panicwrap v1.2.0 // indirect
	github.com/containerd/containerd v1.6.2
	github.com/containernetworking/cni v1.0.1
	github.com/containernetworking/plugins v1.1.1
	github.com/containers/common v0.47.5
	github.com/containers/image/v5 v5.20.0
	github.com/cyphar/filepath-securejoin v0.2.3
	github.com/docker/docker v20.10.14+incompatible
	github.com/fatih/color v1.13.0
	github.com/go-log/log v0.2.0
	github.com/google/uuid v1.3.0
	github.com/gosimple/slug v1.12.0
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0 // indirect
	github.com/kr/pty v1.1.8 // indirect
	github.com/moby/sys/mount v0.2.0 // indirect
	github.com/moby/term v0.0.0-20210619224110-3f7ff695adc6
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.0.3-0.20211202193544-a5463b7f9c84
	github.com/opencontainers/runc v1.1.0
	github.com/opencontainers/runtime-spec v1.0.3-0.20210326190908-1c3f411f0417
	github.com/opencontainers/runtime-tools v0.9.1-0.20210326182921-59cdde06764b
	github.com/opencontainers/selinux v1.10.0
	github.com/opencontainers/umoci v0.4.7
	github.com/pelletier/go-toml v1.9.4
	github.com/pkg/errors v0.9.1
	github.com/seccomp/libseccomp-golang v0.9.2-0.20211028222634-77bddc247e72
	github.com/spf13/cobra v1.4.0
	github.com/spf13/pflag v1.0.5
	github.com/sylabs/json-resp v0.8.1
	github.com/sylabs/scs-build-client v0.4.1
	github.com/sylabs/scs-key-client v0.7.2
	github.com/sylabs/scs-library-client v1.2.2
	github.com/sylabs/sif/v2 v2.4.1
	github.com/urfave/cli v1.22.5 // indirect
	github.com/vbauerster/mpb/v4 v4.12.2
	github.com/vbauerster/mpb/v6 v6.0.4
	github.com/yvasiyarov/go-metrics v0.0.0-20150112132944-c25f46c4b940 // indirect
	github.com/yvasiyarov/gorelic v0.0.6 // indirect
	github.com/yvasiyarov/newrelic_platform_go v0.0.0-20160601141957-9c099fbc30e9 // indirect
	golang.org/x/sys v0.0.0-20220128215802-99c3d69c2c27
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211
	gopkg.in/yaml.v2 v2.4.0
	gotest.tools/v3 v3.1.0
	mvdan.cc/sh/v3 v3.4.3
	oras.land/oras-go v1.1.0
)

// Temporarily force an image-spec that has the main branch commits not in 1.0.2 which is being brought in by oras-go
// These commits are needed by containers/image/v5 and the replace is necessary given how image-spec v1.0.2 has been
// tagged / rebased.
replace github.com/opencontainers/image-spec => github.com/opencontainers/image-spec v1.0.2-0.20211117181255-693428a734f5
