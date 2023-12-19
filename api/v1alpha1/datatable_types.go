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

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DataTableSpec defines the desired state of DataTable
type DataTableSpec struct {
	Url string `json:"url,omitempty"`

	Schemas []Schema `json:"schemas,omitempty"`
}

// Runtime describes a runtime to be used to support dataset
type Runtime struct {

	// Name of the runtime object
	Name string `json:"name,omitempty"`

	// Namespace of the runtime object
	Namespace string `json:"namespace,omitempty"`

	// Category the runtime object belongs to (e.g. Accelerate)
	// Category common.Category `json:"category,omitempty"`

	// Runtime object's type (e.g. Alluxio)
	Type string `json:"type,omitempty"`

	// Runtime master replicas
	MasterReplicas int32 `json:"masterReplicas,omitempty"`
}

// ---------------------------------------------------------------------------
// Datatable defines the observed state of Dataset
// +kubebuilder:subresource:status
// NOTION: 修改这些字段需要重新安装crd, make install
type DataTableStatus struct {
	// the info of mount points have been mounted
	Schemas []Schema `json:"mounts,omitempty"`

	// Total in GB of dataset in the cluster
	UfsTotal string `json:"ufsTotal,omitempty"`

	// Dataset Phase. One of the four phases: `Bound`, `NotBound` and `Failed`
	Phase DatatablePhase `json:"phase,omitempty"` // 表明数据是否挂载

	// Conditions is an array of current observed conditions.
	Conditions []DatatableCondition `json:"conditions,omitempty"` // 没有omitempty表示这个字段是必须要的

	// CacheStatus represents the total resources of the dataset.
	CacheStates CacheStateList `json:"cacheStates,omitempty"`
}

// -----------------------------------------------------------------------------------------------------
// Schema indicates the database.
type Schema struct {
	SchemaName string  `json:"schemaName"`
	Tables     []Table `json:"tables,omitempty"`
}

type Table struct {
	TableName string `json:"tableName"`
	PreUrl    string `json:"preUrl,omitempty"`
	// column infos for this table
	ColumnName []string `json:"columnName,omitempty"`
	// partition infos for the partition
	PartitionColumn []map[string]string `json:"partitionColumn,omitempty"` // 每个map表示一个分区表（可能会有多个kv）
}

// -----------------------------------------------------------------------------------------------------
// DatatablePhase indicates whether the loading is behaving
type DatatablePhase string

const (
	// Bound to dataset, can't be released
	BoundDatasetPhase DatatablePhase = "Bound"
	// Failed, can't be deleted
	FailedDatasetPhase DatatablePhase = "Failed"
	// Not bound to runtime, can be deleted
	NotBoundDatasetPhase DatatablePhase = "NotBound"
)

// ----------------------------------------------------------------------------------------------------
// DatatableCondition describes the state of the cache at a certain point.
type DatatableCondition struct {
	// Type of cache condition.
	Type DatatableConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status v1.ConditionStatus `json:"status"`
	// The reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	Message string `json:"message,omitempty"`
	// The last time this condition was updated.
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
	// Last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}

// DatasetConditionType defines all kinds of types of cacheStatus.<br>
// one of the three types: `RuntimeScheduled`, `Ready` and `Initialized`
type DatatableConditionType string

const (
	// RuntimeScheduled means the runtime CRD has been accepted by the system,
	// But master and workers are not ready
	RuntimeScheduled DatatableConditionType = "RuntimeScheduled"

	// DatasetReady means the cache system for the dataset is ready.
	DatasetReady DatatableConditionType = "Ready"

	// DatasetUpdateReady means the cache system for the dataset is updated.
	DatasetUpdateReady DatatableConditionType = "UpdateReady"

	// DatasetUpdating means the cache system for the dataset is updating.
	DatasetUpdating DatatableConditionType = "Updating"

	// DatasetInitialized means the cache system for the dataset is Initialized.
	DatasetInitialized DatatableConditionType = "Initialized"
)

// -------------------------------------------------------------------------------------------------------------------
// CacheStateName is the name identifying various cacheStateName in a CacheStateNameList.
type CacheStateName string

// ResourceList is a set of (resource name, quantity) pairs.
type CacheStateList map[CacheStateName]string

// CacheStateName names must be not more than 63 characters, consisting of upper- or lower-case alphanumeric characters,
// with the -, _, and . characters allowed anywhere, except the first or last character.
// The default convention, matching that for annotations, is to use lower-case names, with dashes, rather than
// camel case, separating compound words.
// Fully-qualified resource typenames are constructed from a DNS-style subdomain, followed by a slash `/` and a name.
const (
	// Cached in bytes. (500Gi = 500GiB = 500 * 1024 * 1024 * 1024)
	Cached CacheStateName = "cached"
	// Memory, in bytes. (500Gi = 500GiB = 500 * 1024 * 1024 * 1024)
	// Cacheable CacheStateName = "cacheable"
	LowWaterMark CacheStateName = "lowWaterMark"
	// Memory, in bytes. (500Gi = 500GiB = 500 * 1024 * 1024 * 1024)
	HighWaterMark CacheStateName = "highWaterMark"
	// NonCacheable size, in bytes (e,g. 5Gi = 5GiB = 5 * 1024 * 1024 * 1024)
	NonCacheable CacheStateName = "nonCacheable"
	// Percentage represents the cache percentage over the total data in the underlayer filesystem.
	// 1.5 = 1500m
	CachedPercentage CacheStateName = "cachedPercentage"

	CacheCapacity CacheStateName = "cacheCapacity"

	// CacheHitRatio defines total cache hit ratio(both local hit and remote hit), it is a metric to learn
	// how much profit a distributed cache brings.
	CacheHitRatio CacheStateName = "cacheHitRatio"

	// LocalHitRatio defines local hit ratio. It represents how many data is requested from local cache worker
	LocalHitRatio CacheStateName = "localHitRatio"

	// RemoteHitRatio defines remote hit ratio. It represents how many data is requested from remote cache worker(s).
	RemoteHitRatio CacheStateName = "remoteHitRatio"

	// CacheThroughputRatio defines total cache hit throughput ratio, both local hit and remote hit are included.
	CacheThroughputRatio CacheStateName = "cacheThroughputRatio"

	// LocalThroughputRatio defines local cache hit throughput ratio.
	LocalThroughputRatio CacheStateName = "localThroughputRatio"

	// RemoteThroughputRatio defines remote cache hit throughput ratio.
	RemoteThroughputRatio CacheStateName = "remoteThroughputRatio"
)

// +kubebuilder:object:root=true

// DataTable is the Schema for the datatables API
type DataTable struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DataTableSpec   `json:"spec,omitempty"`
	Status DataTableStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DataTableList contains a list of DataTable
type DataTableList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DataTable `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DataTable{}, &DataTableList{})
}

// CacheableNodeAffinity defines constraints that limit what nodes this dataset can be cached to.
type CacheableNodeAffinity struct {
	// Required specifies hard node constraints that must be met.
	Required *v1.NodeSelector `json:"required,omitempty"`
}
