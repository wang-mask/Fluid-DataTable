/*
Copyright 2021 The Fluid Authors.

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

package juicefs

import (
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fluid-cloudnative/fluid/pkg/utils/fake"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	datav1alpha1 "github.com/fluid-cloudnative/fluid/api/v1alpha1"
)

func TestJuiceFSEngine_transform(t *testing.T) {
	juicefsSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "fluid",
		},
		Data: map[string][]byte{
			"metaurl": []byte("test"),
		},
	}
	testObjs := []runtime.Object{}
	testObjs = append(testObjs, (*juicefsSecret).DeepCopy())

	client := fake.NewFakeClientWithScheme(testScheme, testObjs...)
	engine := JuiceFSEngine{
		name:      "test",
		namespace: "fluid",
		Client:    client,
		Log:       fake.NullLogger(),
		runtime: &datav1alpha1.JuiceFSRuntime{
			Spec: datav1alpha1.JuiceFSRuntimeSpec{
				Fuse: datav1alpha1.JuiceFSFuseSpec{},
			},
		},
	}
	ctrl.SetLogger(zap.New(func(o *zap.Options) {
		o.Development = true
	}))

	var tests = []struct {
		runtime *datav1alpha1.JuiceFSRuntime
		dataset *datav1alpha1.Dataset
		value   *JuiceFS
	}{
		{&datav1alpha1.JuiceFSRuntime{
			Spec: datav1alpha1.JuiceFSRuntimeSpec{
				Fuse: datav1alpha1.JuiceFSFuseSpec{},
				Worker: datav1alpha1.JuiceFSCompTemplateSpec{
					Replicas:     2,
					Resources:    corev1.ResourceRequirements{},
					Options:      nil,
					Env:          nil,
					Enabled:      false,
					NodeSelector: nil,
				},
			},
		}, &datav1alpha1.Dataset{
			Spec: datav1alpha1.DatasetSpec{
				Mounts: []datav1alpha1.Mount{{
					MountPoint: "juicefs:///mnt/test",
					Name:       "test",
					EncryptOptions: []datav1alpha1.EncryptOption{{
						Name: "metaurl",
						ValueFrom: datav1alpha1.EncryptOptionSource{
							SecretKeyRef: datav1alpha1.SecretKeySelector{
								Name: "test",
								Key:  "metaurl",
							},
						},
					}},
				}},
			},
		}, &JuiceFS{}},
	}
	for _, test := range tests {
		err := engine.transformFuse(test.runtime, test.dataset, test.value)
		if err != nil {
			t.Errorf("error %v", err)
		}
	}
}

func TestJuiceFSEngine_transformTolerations(t *testing.T) {
	type fields struct {
		name      string
		namespace string
	}
	type args struct {
		dataset *datav1alpha1.Dataset
		value   *JuiceFS
	}
	var tests = []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "test",
			fields: fields{
				name:      "",
				namespace: "",
			},
			args: args{
				dataset: &datav1alpha1.Dataset{Spec: datav1alpha1.DatasetSpec{
					Tolerations: []corev1.Toleration{{
						Key:      "a",
						Operator: corev1.TolerationOpEqual,
						Value:    "b",
					}},
				}},
				value: &JuiceFS{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := &JuiceFSEngine{
				name:      tt.fields.name,
				namespace: tt.fields.namespace,
			}
			j.transformTolerations(tt.args.dataset, tt.args.value)
			if len(tt.args.value.Tolerations) != len(tt.args.dataset.Spec.Tolerations) {
				t.Errorf("transformTolerations() tolerations = %v", tt.args.value.Tolerations)
			}
		})
	}
}

func TestJuiceFSEngine_transformPodMetadata(t *testing.T) {
	engine := &JuiceFSEngine{Log: fake.NullLogger()}

	type testCase struct {
		Name    string
		Runtime *datav1alpha1.JuiceFSRuntime
		Value   *JuiceFS

		wantValue *JuiceFS
	}

	testCases := []testCase{
		{
			Name: "set_common_labels_and_annotations",
			Runtime: &datav1alpha1.JuiceFSRuntime{
				Spec: datav1alpha1.JuiceFSRuntimeSpec{
					PodMetadata: datav1alpha1.PodMetadata{
						Labels:      map[string]string{"common-key": "common-value"},
						Annotations: map[string]string{"common-annotation": "val"},
					},
				},
			},
			Value: &JuiceFS{},
			wantValue: &JuiceFS{
				Worker: Worker{
					Labels:      map[string]string{"common-key": "common-value"},
					Annotations: map[string]string{"common-annotation": "val"},
				},
				Fuse: Fuse{
					Labels:      map[string]string{"common-key": "common-value"},
					Annotations: map[string]string{"common-annotation": "val"},
				},
			},
		},
		{
			Name: "set_master_and_workers_labels_and_annotations",
			Runtime: &datav1alpha1.JuiceFSRuntime{
				Spec: datav1alpha1.JuiceFSRuntimeSpec{
					PodMetadata: datav1alpha1.PodMetadata{
						Labels:      map[string]string{"common-key": "common-value"},
						Annotations: map[string]string{"common-annotation": "val"},
					},
					Worker: datav1alpha1.JuiceFSCompTemplateSpec{
						PodMetadata: datav1alpha1.PodMetadata{
							Labels:      map[string]string{"common-key": "worker-value"},
							Annotations: map[string]string{"common-annotation": "worker-val"},
						},
					},
				},
			},
			Value: &JuiceFS{},
			wantValue: &JuiceFS{
				Worker: Worker{
					Labels:      map[string]string{"common-key": "worker-value"},
					Annotations: map[string]string{"common-annotation": "worker-val"},
				},
				Fuse: Fuse{
					Labels:      map[string]string{"common-key": "common-value"},
					Annotations: map[string]string{"common-annotation": "val"},
				},
			},
		},
	}

	for _, tt := range testCases {
		err := engine.transformPodMetadata(tt.Runtime, tt.Value)
		if err != nil {
			t.Fatalf("test name: %s. Expect err = nil, but got err = %v", tt.Name, err)
		}

		if !reflect.DeepEqual(tt.Value, tt.wantValue) {
			t.Fatalf("test name: %s. Expect value %v, but got %v", tt.Name, tt.wantValue, tt.Value)
		}
	}
}
