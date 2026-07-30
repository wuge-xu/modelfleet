/*
Copyright 2026.

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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// RuntimeType identifies the inference runtime used by a ModelService.
type RuntimeType string

const (
	// RuntimeTypeTransformers runs a lightweight Transformers-based runtime.
	RuntimeTypeTransformers RuntimeType = "transformers"

	// RuntimeTypeKVCacheServe uses KVCache-Serve as the inference runtime.
	RuntimeTypeKVCacheServe RuntimeType = "kvcache-serve"

	// RuntimeTypeVLLM uses vLLM as the inference runtime.
	RuntimeTypeVLLM RuntimeType = "vllm"
)

// ModelServicePhase represents the high-level lifecycle phase of a ModelService.
type ModelServicePhase string

const (
	// ModelServicePhasePending means reconciliation has not completed yet.
	ModelServicePhasePending ModelServicePhase = "Pending"

	// ModelServicePhaseDeploying means Kubernetes resources are being created or updated.
	ModelServicePhaseDeploying ModelServicePhase = "Deploying"

	// ModelServicePhaseReady means the model service is available.
	ModelServicePhaseReady ModelServicePhase = "Ready"

	// ModelServicePhaseDegraded means the model service cannot reach the desired state.
	ModelServicePhaseDegraded ModelServicePhase = "Degraded"
)

// ModelSpec identifies the model artifact that should be served.
type ModelSpec struct {
	// Name is the logical model name exposed by ModelFleet.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Version identifies the model artifact version.
	// +kubebuilder:validation:MinLength=1
	Version string `json:"version"`

	// URI points to the model artifact or model repository.
	// Examples include hf://sshleifer/tiny-gpt2 and pvc://models/tiny-gpt.
	// +kubebuilder:validation:MinLength=1
	URI string `json:"uri"`
}

// RuntimeSpec configures the inference runtime container.
type RuntimeSpec struct {
	// Type selects the runtime adapter used by ModelFleet.
	// +kubebuilder:validation:Enum=transformers;kvcache-serve;vllm
	Type RuntimeType `json:"type"`

	// Image is the container image used to run the inference service.
	// +kubebuilder:validation:MinLength=1
	Image string `json:"image"`

	// Args contains optional additional container arguments.
	// +optional
	Args []string `json:"args,omitempty"`
}

// ModelServiceSpec defines the desired state of ModelService.
type ModelServiceSpec struct {
	// Model identifies the model artifact to serve.
	Model ModelSpec `json:"model"`

	// Runtime configures the inference runtime.
	Runtime RuntimeSpec `json:"runtime"`

	// Replicas is the desired number of inference runtime Pods.
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// Port is the TCP port exposed by the inference runtime.
	// +kubebuilder:default=8000
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +optional
	Port int32 `json:"port,omitempty"`

	// Resources defines CPU and memory requests and limits.
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

// ModelServiceStatus defines the observed state of ModelService.
type ModelServiceStatus struct {
	// ObservedGeneration is the latest metadata generation processed by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Phase is the high-level lifecycle phase of the model service.
	// +kubebuilder:validation:Enum=Pending;Deploying;Ready;Degraded
	// +optional
	Phase ModelServicePhase `json:"phase,omitempty"`

	// DeploymentName is the managed inference Deployment.
	// +optional
	DeploymentName string `json:"deploymentName,omitempty"`

	// ServiceName is the managed Kubernetes Service.
	// +optional
	ServiceName string `json:"serviceName,omitempty"`

	// ReadyReplicas is the number of ready inference Pods.
	// +optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// AvailableReplicas is the number of available inference Pods.
	// +optional
	AvailableReplicas int32 `json:"availableReplicas,omitempty"`

	// Conditions represent detailed aspects of the current state.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=msvc
// +kubebuilder:printcolumn:name="Model",type=string,JSONPath=".spec.model.name"
// +kubebuilder:printcolumn:name="Version",type=string,JSONPath=".spec.model.version"
// +kubebuilder:printcolumn:name="Runtime",type=string,JSONPath=".spec.runtime.type"
// +kubebuilder:printcolumn:name="Ready",type=integer,JSONPath=".status.readyReplicas"
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"

// ModelService is the Schema for the modelservices API.
type ModelService struct {
	metav1.TypeMeta `json:",inline"`

	// Metadata is standard Kubernetes object metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// Spec defines the desired state of ModelService.
	// +required
	Spec ModelServiceSpec `json:"spec"`

	// Status defines the observed state of ModelService.
	// +optional
	Status ModelServiceStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// ModelServiceList contains a list of ModelService resources.
type ModelServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []ModelService `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &ModelService{}, &ModelServiceList{})
		return nil
	})
}
