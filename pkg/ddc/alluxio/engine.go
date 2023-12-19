package alluxio

import (
	datav1alpha1 "Fluid-Datatable/api/v1alpha1"
	"Fluid-Datatable/pkg/ddc/base"
	cruntime "Fluid-Datatable/pkg/runtime"
	"Fluid-Datatable/pkg/utils/kubeclient"
	"fmt"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TODO one 判断是否需要修改以上三个地方，然后进行修改
// AlluxioEngine implements the Engine interface.
type AlluxioEngine struct {
	// *base.TemplateEngine
	runtime     *datav1alpha1.AlluxioRuntime
	name        string
	namespace   string
	runtimeType string
	Log         logr.Logger
	client.Client
	// gracefulShutdownLimits is the limit for the system to forcibly clean up.
	gracefulShutdownLimits int32
	retryShutdown          int32
	initImage              string
	MetadataSyncDoneCh     chan MetadataSyncResult
	runtimeInfo            base.RuntimeInfoInterface
	UnitTest               bool
	//lastCacheHitStates     *cacheHitStates
}

// Build function builds the Alluxio Engine
func Build(id string, ctx cruntime.ReconcileRequestContext) (base.Engine, error) {
	engine := &AlluxioEngine{
		name:                   ctx.Name,
		namespace:              ctx.Namespace,
		Client:                 ctx.Client,
		Log:                    ctx.Log,
		runtimeType:            ctx.RuntimeType,
		gracefulShutdownLimits: 5,
		retryShutdown:          0,
		MetadataSyncDoneCh:     nil,
		//lastCacheHitStates:     nil,
	}
	// var implement base.Implement = engine
	// engine.TemplateEngine = template
	if ctx.Runtime != nil {
		runtime, ok := ctx.Runtime.(*datav1alpha1.AlluxioRuntime)
		if !ok {
			return nil, fmt.Errorf("engine %s is failed to parse", ctx.Name)
		}
		engine.runtime = runtime
	} else {
		return nil, fmt.Errorf("engine %s is failed to parse", ctx.Name)
	}

	// Build and setup runtime info
	_, err := engine.getRuntimeInfo()
	if err != nil {
		return nil, fmt.Errorf("engine %s failed to get runtime info", ctx.Name)
	}

	template := base.NewTemplateEngine(engine, id, ctx)

	err = kubeclient.EnsureNamespace(ctx.Client, ctx.Namespace)
	return template, err
}
