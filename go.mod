module halkyon.io/operator-framework

go 1.13

require (
	github.com/go-logr/logr v0.1.0
	github.com/hashicorp/go-hclog v0.0.0-20180709165350-ff2cf002a8dd
	github.com/hashicorp/go-plugin v1.0.1
	halkyon.io/api v1.0.0-rc.4.0.20200205214834-8964fac782cc
	k8s.io/api v0.0.0-20190918195907-bd6ac527cfd2
	k8s.io/apimachinery v0.17.0
	k8s.io/client-go v11.0.1-0.20190805182715-88a2adca7e76+incompatible
	k8s.io/code-generator v0.17.0 // indirect
	k8s.io/gengo v0.0.0-20191120174120-e74f70b9b27e // indirect
	sigs.k8s.io/controller-runtime v0.3.0
)

replace (
	k8s.io/api => k8s.io/api v0.0.0-20190805182251-6c9aa3caf3d6 // kubernetes-1.14.5
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190404173353-6a84e37a896d // kubernetes-1.14.5
	k8s.io/client-go => k8s.io/client-go v11.0.1-0.20190805182715-88a2adca7e76+incompatible
)
