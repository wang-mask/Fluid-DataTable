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

package base

import (
	cruntime "Fluid-Datatable/pkg/runtime"
)

// SyncReplicas syncs the replicas
func (t *TemplateEngine) Sync(ctx cruntime.ReconcileRequestContext) (err error) {
	err = t.Implement.SyncMetadata()
	// Alluxio挂载时是没有元数据的，当用户开始list时才会真正开始拉取。
	// 同步底层存储系统和Alluxio,将Alluxio的文件元信息拉取过来
	// 只会执行一次，dataset的中的关键信息，数据集总大小，当前数据的被缓存等，缓存系统总容量。。
	// 将dataset中的ufs中文件元信息给同步到dataset中。
	// 可以忽略
	if err != nil {
		return
	}
	//
	_, err = t.Implement.CheckAndUpdateRuntimeStatus()
	if err != nil {
		return
	}

	/*	err = t.Implement.UpdateCacheOfDataset()
		if err != nil {
			return
		}*/

	// 1. Check healthy
	err = t.Implement.CheckRuntimeHealthy()
	if err != nil {
		return
	}

	// 2. Sync replicas
	err = t.Implement.SyncReplicas(ctx)
	if err != nil {
		return
	}

	// 3. Update runtime
	_, err = t.Implement.CheckAndUpdateRuntimeStatus()
	if err != nil {
		return
	}

	// 4. Update dataset mount point
	/*	ufsToUpdate := t.Implement.ShouldUpdateUFS()
		if ufsToUpdate != nil {
			if ufsToUpdate.ShouldUpdate() {
				var updateReady bool
				updateReady, err = t.Implement.UpdateOnUFSChange(ufsToUpdate) // 针对dataset的挂载点进行挂载
				if err != nil {
					return
				}
				if updateReady {
					err = utils.UpdateMountStatus(t.Client, t.Context.Name, t.Context.Namespace, datav1alpha1.BoundDatasetPhase)
					if err != nil {
						return
					}
				}
			}
		}*/

	return t.Implement.SyncScheduleInfoToCacheNodes()
}
