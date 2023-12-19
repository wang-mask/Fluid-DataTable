package runtime

import (
	"context"

	//"github.com/fluid-cloudnative/fluid/pkg/common"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ReconcileRequestContext loads or applys the configuration state of a service.
type ReconcileRequestContext struct {
	context.Context
	types.NamespacedName
	Log logr.Logger
	//  *datav1alpha1.DataTable  // 原来是已经绑定的dataset
	Recorder record.EventRecorder
	client.Client
	RuntimeType   string
	FinalizerName string
	//common.Category
	Runtime runtime.Object
}
