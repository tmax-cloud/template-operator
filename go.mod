module github.com/youngind/hypercloud-operator

go 1.13

require (
	github.com/kubernetes-client/go v0.0.0-20200222171647-9dac5e4c5400
	github.com/operator-framework/operator-sdk v0.17.1
	github.com/spf13/pflag v1.0.5
	github.com/tidwall/gjson v1.6.0
	k8s.io/api v0.17.4
	k8s.io/apimachinery v0.17.4
	k8s.io/client-go v12.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.5.2
)

replace (
	k8s.io/client-go => k8s.io/client-go v0.17.4 // Required by prometheus-operator
)
