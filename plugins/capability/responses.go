package capability

import "k8s.io/apimachinery/pkg/runtime"

type IsReadyResponse struct {
	Ready   bool
	Message string
}

type UpdateResponse struct {
	NeedsUpdate bool
	Error       error
	Updated     runtime.Object
}

type BuildResponse struct {
	Built runtime.Object
}
