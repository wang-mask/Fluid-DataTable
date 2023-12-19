module Fluid-Datatable

go 1.13

require (
	github.com/dazheng/gohive v0.0.0-20190904024313-b1810177c8f2
	github.com/docker/go-units v0.4.0
	github.com/go-logr/logr v0.2.0
	github.com/go-redis/redis/v8 v8.11.5
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.1.1
	go.uber.org/zap v1.10.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.20.2
	k8s.io/apiextensions-apiserver v0.20.2 // indirect
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.18.5
	k8s.io/gengo v0.0.0-20221011193443-fad74ee6edd9
	sigs.k8s.io/controller-runtime v0.5.0
// k8s.io/kubernetes v1.18.19
)

replace k8s.io/api => k8s.io/api v0.18.5

replace k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.18.5

replace k8s.io/apimachinery => k8s.io/apimachinery v0.18.6-rc.0

replace k8s.io/apiserver => k8s.io/apiserver v0.18.5

replace k8s.io/cli-runtime => k8s.io/cli-runtime v0.18.5

replace k8s.io/client-go => k8s.io/client-go v0.18.5

replace k8s.io/cloud-provider => k8s.io/cloud-provider v0.18.5

replace k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.18.5

replace k8s.io/code-generator => k8s.io/code-generator v0.18.6-rc.0

replace k8s.io/component-base => k8s.io/component-base v0.18.5

replace k8s.io/cri-api => k8s.io/cri-api v0.18.6-rc.0

replace k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.18.5

replace k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.18.5

replace k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.18.5

replace k8s.io/kube-proxy => k8s.io/kube-proxy v0.18.5

replace k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.18.5

replace k8s.io/kubectl => k8s.io/kubectl v0.18.5

replace k8s.io/kubelet => k8s.io/kubelet v0.18.5

replace k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.18.5

replace k8s.io/metrics => k8s.io/metrics v0.18.5

replace sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.6.0
