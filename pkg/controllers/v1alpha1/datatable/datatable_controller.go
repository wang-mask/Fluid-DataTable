/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package datatable

import (
	datav1alpha1 "Fluid-Datatable/api/v1alpha1"
	"Fluid-Datatable/pkg/utils"
	datatableUtils "Fluid-Datatable/pkg/utils/datatable"
	execAlluxio "Fluid-Datatable/pkg/utils/exec"
	"Fluid-Datatable/pkg/utils/hive"
	re "Fluid-Datatable/pkg/utils/redisUtil"
	"context"
	"github.com/dazheng/gohive"
	"github.com/go-logr/logr"
	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"strings"
)

// DataTableReconciler reconciles a DataTable object
type DataTableReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

type reconcileRequestContext struct {
	context.Context
	Log       logr.Logger
	Datatable datav1alpha1.DataTable
	types.NamespacedName
	FinalizerName string
}

const (
	// finalizer for datable
	DataTableFinalizerName = "datatable-controller-finalizer"
)

// +kubebuilder:rbac:groups=data.fluid.io,resources=datatables,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=data.fluid.io,resources=datatables/status,verbs=get;update;patch
func (r *DataTableReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {

	ctx := reconcileRequestContext{
		Context:        context.Background(),
		Log:            r.Log.WithValues("datatable", req.NamespacedName),
		NamespacedName: req.NamespacedName,
		FinalizerName:  DataTableFinalizerName,
	}
	ctx.Log.V(1).Info("process the request", "request", req)

	notFound := false
	if err := r.Get(ctx, req.NamespacedName, &ctx.Datatable); err != nil {
		ctx.Log.Info("unable to fetch DataTable", "reason", err)
		if utils.IgnoreNotFound(err) != nil { // is not NOT FOUND err
			r.Log.Error(err, "fail to get datatable")
			return ctrl.Result{}, err
		} else { // is not found err
			notFound = true
		}
	} else {
		return r.reconcileDataTable(ctx)
	}

	if notFound {
		ctx.Log.V(1).Info("Not found.")
	}

	return ctrl.Result{}, nil
}

var redisclient *redis.Client

// var conn *gohive.Connection

// 放入缓存的需要额外添加
func (r *DataTableReconciler) reconcileDataTable(ctx reconcileRequestContext) (result ctrl.Result, err error) {
	log := ctx.Log.WithName("reconcileDatatable")
	log.V(1).Info("process the datatable", "datatable", ctx.Datatable)

	datatable := ctx.Datatable
	objectMeta := ctx.Datatable.ObjectMeta

	// TODO 查看alluxio集群是否创建
	var alluxioRuntimeList datav1alpha1.AlluxioRuntimeList
	if err := r.List(ctx, &alluxioRuntimeList); err != nil {
		r.Log.Error(err, "fail to get the alluxioruntime")
		return ctrl.Result{}, err
	}

	if len(alluxioRuntimeList.Items) == 0 {
		// 不存在Alluxio 集群就直接报错
		r.Log.Error(errors.New("fail to get the alluxio runtime, please run the alluxio runtime first"), "fail to get the alluxio runtime")
		return ctrl.Result{}, errors.New("fail to get the alluxio runtime")
	}

	alluxioPodName := alluxioRuntimeList.Items[0].Name + "-master-0"
	alluxioPodNamespace := alluxioRuntimeList.Items[0].Namespace
	alluxioContainerName := "alluxio-master"

	// 创建hive客户端进行修改
	// 连接hive metastore
	hostPort := datatable.Spec.Url
	schemas := datatable.Spec.Schemas

	// 表连接到客户端，需要在服务端启动hive --service metastore， hive --service hiveserver2  // 后续考虑使用连接池
	//if conn == nil {
	conn, err := hive.ConnectHiveMetaStore(hostPort)
	if err != nil {
		r.Log.Error(err, "fail to connect the hive service")
		conn.Close()
		return ctrl.Result{}, err
	}
	//	}

	// todo 添加是否放入的逻辑
	if datatable.Spec.IntelligentCache {
		//	如果空间足够，可以直接放入

		// 如果空间不够，连接redis，看是否需要放入
		// 如果没有连接到redis，进行连接
		if redisclient == nil {
			redisclient = re.ConnectRedis()
		}

		// 对每一张表进行判断其是否需要加入缓存中
		for _, schema := range datatable.Spec.Schemas {
			for _, table := range schema.Tables {
				// 如果该表在redis中进行处理，不在redis中的不进行考虑，直到
				schemaTableName := schema.SchemaName + "_" + table.TableName
				if re.IsinZset(redisclient, schemaTableName) { // 在zset中，表示该表之前执行过，可以进行比较
					if !re.IsExistInALluxio(redisclient, schemaTableName) {
						// 如果不在集群中，则开始判断是否需要加入,因为不在缓存中，在redis中没有其表大小，此时需要从hive中获取
						// https://www.cnblogs.com/foolaround/p/13048942.html
						tableSize, err := hive.GetTableSize(conn, schema.SchemaName, table.TableName)
						if err != nil {

						}

						// Step1 : 从pod中开始执行获取剩余空间总大小
						var freeSize int64
						// podName string,  containerName string, namespace string
						freeSize, err = execAlluxio.ExecAlluxioFreeSize(alluxioPodName, alluxioContainerName, alluxioPodNamespace)
						if err != nil {
							ctx.Log.Error(err, "")
						}
						// 如果集群剩余大小大于表大小
						if tableSize < freeSize {
							// 直接进行放入即可
						} else {
							// 如果集群大小小于表大小，一直获取表大小直到空间满足可以存放新的表。
							tableCollections := re.GetValue(redisclient)
							var freeTableCollection []string
							for _, tableTmp := range tableCollections { // 对所有的表进行判断进行驱逐
								if tableTmp == schemaTableName {
									freeTableCollection = nil
									break
								}

								if re.IsExistInALluxio(redisclient, tableTmp) {
									tSize := re.GetSize(redisclient, tableTmp)
									freeTableCollection = append(freeTableCollection, tableTmp)
									if freeSize+tSize < tableSize {
										freeSize = freeSize + tSize
										continue
									} else {
										break
									}
								}
							}

							// todo 对tableCollection中的表进行free操作
							for _, stName := range freeTableCollection {
								s := strings.Split(stName, "_")
								execAlluxio.ExecAlluxioTableUnmount(alluxioPodName, alluxioContainerName, alluxioPodNamespace, s[0], s[1])
							}
						}
					} else {
						// 在Alluxio中，不需要做任何操作
					}
				} else { // 不在zset中，则表示该表没有被使用过
					ctx.Log.Info("Table " + schema.SchemaName + "_" + table.TableName + " will not be cache due to no history query.")
				}
			}
		}
	}

	// 判断是否需要执行删除逻辑
	if !objectMeta.GetDeletionTimestamp().IsZero() {
		// 进行删除逻辑
		return r.ReconcileDatatableDeletion(ctx, conn)
	} else { // 此时没有删除集群操作
		if !utils.ContainsString(objectMeta.GetFinalizers(), ctx.FinalizerName) {
			return r.AddFinalizerAndRequeue(ctx)
		} else {
			ctx.Log.Info("The finalizer has been added!")
		}
	}

	// 修改表的位置信息
	var allLabelsToUpdate []string
	for _, schema := range schemas {
		// 判断是否需要更新datatable的内容
		labelToUpdate, err := hive.ChangeSchemaURLForCache(r.Client, datatable, conn, schema)
		if err != nil {
			r.Log.Error(err, "fail to change the table location")
			_ = conn.Close()
			return ctrl.Result{}, err
		}
		allLabelsToUpdate = append(allLabelsToUpdate, labelToUpdate...)
	}

	// 如果是刚创建的datatable，添加所有table的标签
	if len(datatable.Status.Schemas) == 0 {
		err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
			datatable, err := datatableUtils.GetDatatable(r.Client, ctx.Name, ctx.Namespace)
			if err != nil {
				return err
			}

			datatableToUpdate := datatable.DeepCopy()
			datatableToUpdate.Status.Schemas = datatableToUpdate.Spec.Schemas
			datatableToUpdate.Labels = map[string]string{} // need initialize

			for _, labelToUpdate := range allLabelsToUpdate {
				datatableToUpdate.Labels[labelToUpdate] = "true"
			}

			if !reflect.DeepEqual(datatable, datatableToUpdate) {
				err = r.Client.Update(context.TODO(), datatableToUpdate)
				if err != nil {
					return err
				}
			}
			return err
		})
		if err != nil {
			r.Log.Error(err, "fail to update the label of datatable", "datatable namespace", datatable.Namespace, "datatable name", datatable.Name)
			return ctrl.Result{}, err
		}
	}

	_ = conn.Close()
	return ctrl.Result{}, nil

}

func (r *DataTableReconciler) ReconcileDatatableDeletion(ctx reconcileRequestContext, conn *gohive.Connection) (ctrl.Result, error) {

	datatable := ctx.Datatable
	err := hive.ChangeSchemaURLForRecover(r.Client, datatable, conn)
	if err != nil {
		r.Log.Error(err, "fail to change the table location")
		return ctrl.Result{}, err
	}

	if !ctx.Datatable.ObjectMeta.GetDeletionTimestamp().IsZero() {
		finalizers := utils.RemoveString(ctx.Datatable.ObjectMeta.GetFinalizers(), ctx.FinalizerName)
		ctx.Datatable.ObjectMeta.SetFinalizers(finalizers)
		r.Log.Info("After clean up finalizer", "runtime", ctx.Datatable)
		if err := r.Update(ctx, &ctx.Datatable); err != nil {
			r.Log.Error(err, "Failed to remove finalizer")
			return utils.RequeueIfError(err)
		}
		ctx.Log.V(1).Info("Finalizer is removed", "runtime", ctx.Datatable)
	}

	return ctrl.Result{}, nil
	//utils.ChangeTableURL(conn, table, false)

}

func (r *DataTableReconciler) AddFinalizerAndRequeue(ctx reconcileRequestContext) (ctrl.Result, error) {
	log := ctx.Log.WithName("AddFinalizerAndRequeue")
	log.Info("add finalizer and requeue", "Runtime", ctx.NamespacedName)
	// objectMetaAccessor, isOM := ctx.Runtime.(metav1.ObjectMetaAccessor)
	// if !isOM {
	// 	return utils.RequeueIfError(fmt.Errorf("object is not ObjectMetaAccessor"))
	// }
	// objectMeta := objectMetaAccessor.GetObjectMeta()
	// objectMeta := ctx.Datatable.ObjectMeta  // 这个可能是更改了objectMeta，而不是ctx.Datatable.ObjectMeta

	prevGeneration := ctx.Datatable.ObjectMeta.GetGeneration()
	ctx.Datatable.ObjectMeta.SetFinalizers(append(ctx.Datatable.ObjectMeta.GetFinalizers(), ctx.FinalizerName))
	if err := r.Update(ctx, &ctx.Datatable); err != nil {
		ctx.Log.Error(err, "Failed to add finalizer", "StatusUpdateError", ctx, "datatable", ctx.Datatable)
		return utils.RequeueIfError(err)
	}
	// controllerutil.AddFinalizer(ctx.Runtime, finalizer)
	currentGeneration := ctx.Datatable.ObjectMeta.GetGeneration() // 表示当前对象目标状态的代码
	ctx.Log.Info("RequeueImmediatelyUnlessGenerationChanged", "prevGeneration", prevGeneration,
		"currentGeneration", currentGeneration)

	return utils.RequeueImmediatelyUnlessGenerationChanged(prevGeneration, currentGeneration)
}

func (r *DataTableReconciler) SetupWithManager(mgr ctrl.Manager, options controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		For(&datav1alpha1.DataTable{}).
		Complete(r)
}
