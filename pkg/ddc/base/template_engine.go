package base

import (
	cruntime "Fluid-Datatable/pkg/runtime"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Use compiler to check if the struct implements all the interface
var _ Engine = (*TemplateEngine)(nil)

type TemplateEngine struct {
	Implement // engine 的具体实现
	Id        string
	client.Client
	Log     logr.Logger
	Context cruntime.ReconcileRequestContext
}

// NewTemplateEngine creates template engine
func NewTemplateEngine(impl Implement,
	id string,
	// client client.Client,
	// log logr.Logger,
	context cruntime.ReconcileRequestContext) *TemplateEngine {
	b := &TemplateEngine{
		Implement: impl,
		Id:        id,
		Context:   context,
		Client:    context.Client,
		// Log:       log,
	}
	b.Log = context.Log.WithValues("engine", context.RuntimeType).WithValues("id", id)
	return b
}

//--------------------------------------------------------------------------------------
// ID returns the id of the engine
func (t *TemplateEngine) ID() string {
	return t.Id
}

//Shutdown and clean up the engine
func (t *TemplateEngine) Shutdown() error {
	return t.Implement.Shutdown()
}

// Setup the ddc engine
func (b *TemplateEngine) Setup(ctx cruntime.ReconcileRequestContext) (ready bool, err error) {
	var (
		shouldSetupMaster  bool
		masterReady        bool
		shouldSetupWorkers bool
		workersReady       bool
	)

	b.Log.Info("Setup the ddc engine", "runtime", ctx.Runtime)
	// 1.Check if we should setup the master
	// shouldSetupMaster, err
	shouldSetupMaster, err = b.Implement.ShouldSetupMaster()
	if err != nil {
		return ready, err
	}
	if shouldSetupMaster {
		err = b.Implement.SetupMaster()
		if err != nil {
			b.Log.Error(err, "SetupMaster")
			return ready, err
		}
	}

	// 2.Check if the master is ready, then go forward to workers setup
	masterReady, err = b.Implement.CheckMasterReady()
	if err != nil {
		b.Log.Error(err, "Failed to check if it CheckMasterReady.")
		return ready, err
	}

	if !masterReady {
		return masterReady, err
	}

	//针对底层文件系统进行挂载datase
	/*	shouldCheckUFS, err := b.Implement.ShouldCheckUFS()
		if err != nil {
			b.Log.Error(err, "Failed to check if it requires checking ufs.")
			return ready, err
		}

		if shouldCheckUFS {
			err = b.Implement.PrepareUFS()
			if err != nil {
				b.Log.Error(err, "Failed to prepare ufs.")
				return ready, err
			}
		}*/

	// 3.Check if we should setup the workers
	shouldSetupWorkers, err = b.Implement.ShouldSetupWorkers()
	if err != nil {
		b.Log.Error(err, "Failed to check if it ShouldSetupWorkers.")
		return ready, err
	}

	// worker pod的调度
	if shouldSetupWorkers {
		err = b.Implement.SetupWorkers()
		if err != nil {
			b.Log.Error(err, "SetupWorker")
			return ready, err
		}
	}

	// 4.Check if the workers are ready
	workersReady, err = b.Implement.CheckWorkersReady()
	if err != nil {
		b.Log.Error(err, "Check if the workers are ready")
		return workersReady, err
	}

	if !workersReady {
		return workersReady, err
	}

	// 5.Check if the runtime is ready
	runtimeReady, err := b.Implement.CheckAndUpdateRuntimeStatus()
	if err != nil {
		b.Log.Error(err, "Check if the runtime is ready")
		return runtimeReady, err
	}

	if !runtimeReady {
		return runtimeReady, err
	}

	// 6.Update the dataset status from pending to bound
	/*	err = b.Implement.BindToDataset()
		if err != nil {
			b.Log.Error(err, "Bind the dataset")
			return workersReady, err
		}*/

	ready = true

	return ready, err
}
