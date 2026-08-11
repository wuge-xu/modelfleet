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

package controller

import (
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	servingv1alpha1 "github.com/wuge-xu/modelfleet/api/v1alpha1"
)

func TestBuildModelDeployment(t *testing.T) {
	scheme := runtime.NewScheme()

	if err := servingv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add ModelService scheme: %v", err)
	}

	replicas := int32(2)

	modelService := &servingv1alpha1.ModelService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tiny-gpt",
			Namespace: "default",
			UID:       types.UID("modelservice-uid"),
		},
		Spec: servingv1alpha1.ModelServiceSpec{
			Model: servingv1alpha1.ModelSpec{
				Name:    "tiny-gpt",
				Version: "v1",
				URI:     "hf://sshleifer/tiny-gpt2",
			},
			Runtime: servingv1alpha1.RuntimeSpec{
				Type:  servingv1alpha1.RuntimeTypeTransformers,
				Image: "ghcr.io/wuge-xu/modelfleet-tiny-runtime:v0.1.0",
				Args:  []string{"--model-uri=hf://sshleifer/tiny-gpt2"},
			},
			Replicas: &replicas,
			Port:     8080,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("250m"),
					corev1.ResourceMemory: resource.MustParse("512Mi"),
				},
			},
		},
	}

	deployment, err := buildModelDeployment(modelService, scheme)
	if err != nil {
		t.Fatalf("build Deployment: %v", err)
	}

	if deployment.Name != "tiny-gpt-runtime" {
		t.Fatalf("unexpected Deployment name: %s", deployment.Name)
	}

	if deployment.Namespace != "default" {
		t.Fatalf("unexpected Deployment namespace: %s", deployment.Namespace)
	}

	if deployment.Spec.Replicas == nil || *deployment.Spec.Replicas != 2 {
		t.Fatalf("unexpected replicas: %v", deployment.Spec.Replicas)
	}

	if len(deployment.Spec.Template.Spec.Containers) != 1 {
		t.Fatalf("expected one runtime container")
	}

	container := deployment.Spec.Template.Spec.Containers[0]

	if container.Name != runtimeContainerName {
		t.Fatalf("unexpected container name: %s", container.Name)
	}

	if container.Image != modelService.Spec.Runtime.Image {
		t.Fatalf("unexpected image: %s", container.Image)
	}

	if !reflect.DeepEqual(container.Args, modelService.Spec.Runtime.Args) {
		t.Fatalf("unexpected args: %#v", container.Args)
	}

	if len(container.Ports) != 1 || container.Ports[0].ContainerPort != 8080 {
		t.Fatalf("unexpected container ports: %#v", container.Ports)
	}

	if container.Resources.Requests.Cpu().String() != "250m" {
		t.Fatalf(
			"unexpected CPU request: %s",
			container.Resources.Requests.Cpu().String(),
		)
	}

	if container.Resources.Requests.Memory().String() != "512Mi" {
		t.Fatalf(
			"unexpected memory request: %s",
			container.Resources.Requests.Memory().String(),
		)
	}

	for key, value := range deployment.Spec.Selector.MatchLabels {
		if deployment.Spec.Template.Labels[key] != value {
			t.Fatalf(
				"selector label %s=%s is missing from Pod labels",
				key,
				value,
			)
		}
	}

	if deployment.Spec.Template.Labels["serving.modelfleet.io/runtime"] != "transformers" {
		t.Fatalf("runtime Pod label was not set")
	}

	if deployment.Annotations["serving.modelfleet.io/model-version"] != "v1" {
		t.Fatalf("model version annotation was not set")
	}

	if deployment.Spec.Template.Annotations["serving.modelfleet.io/model-uri"] != modelService.Spec.Model.URI {
		t.Fatalf("model URI annotation was not set")
	}

	ownerReferences := deployment.GetOwnerReferences()

	if len(ownerReferences) != 1 {
		t.Fatalf(
			"expected one owner reference, got %d",
			len(ownerReferences),
		)
	}

	ownerReference := ownerReferences[0]

	if ownerReference.UID != modelService.UID {
		t.Fatalf("unexpected owner UID: %s", ownerReference.UID)
	}

	if ownerReference.Controller == nil || !*ownerReference.Controller {
		t.Fatalf("ModelService owner reference is not marked as controller")
	}
}

func TestBuildModelDeploymentUsesDefaults(t *testing.T) {
	scheme := runtime.NewScheme()

	if err := servingv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add ModelService scheme: %v", err)
	}

	modelService := &servingv1alpha1.ModelService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "defaulted-model",
			Namespace: "default",
			UID:       types.UID("defaulted-model-uid"),
		},
		Spec: servingv1alpha1.ModelServiceSpec{
			Model: servingv1alpha1.ModelSpec{
				Name:    "defaulted-model",
				Version: "v1",
				URI:     "hf://example/defaulted-model",
			},
			Runtime: servingv1alpha1.RuntimeSpec{
				Type:  servingv1alpha1.RuntimeTypeTransformers,
				Image: "example.invalid/runtime:v1",
			},
		},
	}

	deployment, err := buildModelDeployment(modelService, scheme)
	if err != nil {
		t.Fatalf("build Deployment: %v", err)
	}

	if deployment.Spec.Replicas == nil ||
		*deployment.Spec.Replicas != defaultModelReplicas {
		t.Fatalf(
			"expected default replicas %d",
			defaultModelReplicas,
		)
	}

	containerPort :=
		deployment.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort

	if containerPort != defaultModelPort {
		t.Fatalf(
			"expected default port %d, got %d",
			defaultModelPort,
			containerPort,
		)
	}
}
