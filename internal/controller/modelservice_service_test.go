package controller

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	servingv1alpha1 "github.com/wuge-xu/modelfleet/api/v1alpha1"
)

func TestBuildModelService(t *testing.T) {
	scheme := runtime.NewScheme()

	if err := servingv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add ModelService scheme: %v", err)
	}

	modelService := &servingv1alpha1.ModelService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "default",
			UID:       "test-uid",
		},
		Spec: servingv1alpha1.ModelServiceSpec{
			Runtime: servingv1alpha1.RuntimeSpec{
				Type: servingv1alpha1.RuntimeTypeVLLM,
			},
			Port: 9000,
		},
	}

	service, err := buildModelService(
		modelService,
		scheme,
	)
	if err != nil {
		t.Fatalf("build Service: %v", err)
	}

	if service.Name != "demo-service" {
		t.Fatalf(
			"expected service name demo-service, got %s",
			service.Name,
		)
	}

	if service.Namespace != "default" {
		t.Fatalf(
			"expected namespace default, got %s",
			service.Namespace,
		)
	}

	if service.Spec.Type != corev1.ServiceTypeClusterIP {
		t.Fatalf(
			"expected ClusterIP service, got %s",
			service.Spec.Type,
		)
	}

	if len(service.Spec.Ports) != 1 {
		t.Fatalf(
			"expected 1 service port, got %d",
			len(service.Spec.Ports),
		)
	}

	port := service.Spec.Ports[0]

	if port.Port != 9000 {
		t.Fatalf(
			"expected service port 9000, got %d",
			port.Port,
		)
	}

	if port.TargetPort.IntVal != 9000 {
		t.Fatalf(
			"expected target port 9000, got %d",
			port.TargetPort.IntVal,
		)
	}

	if _, exists :=
		service.Spec.Selector["serving.modelfleet.io/runtime"]; exists {
		t.Fatal(
			"runtime must not be part of stable Service selector",
		)
	}

	if service.Spec.Selector["serving.modelfleet.io/modelservice"] != "demo" {
		t.Fatalf(
			"expected modelservice selector demo",
		)
	}

	if len(service.OwnerReferences) != 1 {
		t.Fatalf(
			"expected one OwnerReference, got %d",
			len(service.OwnerReferences),
		)
	}

	if service.OwnerReferences[0].Name != "demo" {
		t.Fatalf(
			"expected owner demo, got %s",
			service.OwnerReferences[0].Name,
		)
	}
}

func TestBuildModelServiceUsesDefaultPort(t *testing.T) {
	scheme := runtime.NewScheme()

	if err := servingv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add ModelService scheme: %v", err)
	}

	modelService := &servingv1alpha1.ModelService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default-port",
			Namespace: "default",
			UID:       "test-uid",
		},
		Spec: servingv1alpha1.ModelServiceSpec{
			Runtime: servingv1alpha1.RuntimeSpec{
				Type: servingv1alpha1.RuntimeTypeTransformers,
			},
		},
	}

	service, err := buildModelService(
		modelService,
		scheme,
	)
	if err != nil {
		t.Fatalf("build Service: %v", err)
	}

	port := service.Spec.Ports[0]

	if port.Port != 8000 {
		t.Fatalf(
			"expected default service port 8000, got %d",
			port.Port,
		)
	}

	if port.TargetPort.IntVal != 8000 {
		t.Fatalf(
			"expected default target port 8000, got %d",
			port.TargetPort.IntVal,
		)
	}
}
