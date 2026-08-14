package controller

import (
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	servingv1alpha1 "github.com/wuge-xu/modelfleet/api/v1alpha1"
)

func TestBuildModelDeploymentProbes(t *testing.T) {
	scheme := runtime.NewScheme()

	if err := servingv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add ModelService scheme: %v", err)
	}

	modelService := &servingv1alpha1.ModelService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "probe-model",
			Namespace: "default",
			UID:       "test-uid",
		},
		Spec: servingv1alpha1.ModelServiceSpec{
			Model: servingv1alpha1.ModelSpec{
				Name:    "probe-model",
				Version: "v1",
				URI:     "hf://example/probe-model",
			},
			Runtime: servingv1alpha1.RuntimeSpec{
				Type:  servingv1alpha1.RuntimeTypeVLLM,
				Image: "example.invalid/runtime:v1",
				Args: []string{
					"--tensor-parallel-size=1",
				},
			},
			Port: 9000,
		},
	}

	deployment, err := buildModelDeployment(
		modelService,
		scheme,
	)
	if err != nil {
		t.Fatalf("build Deployment: %v", err)
	}

	runtimeContainer :=
		deployment.Spec.Template.Spec.Containers[0]

	expectedArgs := []string{
		"--model",
		"example/probe-model",
		"--port",
		"9000",
		"--tensor-parallel-size=1",
	}

	if !reflect.DeepEqual(
		runtimeContainer.Args,
		expectedArgs,
	) {
		t.Fatalf(
			"expected args %v, got %v",
			expectedArgs,
			runtimeContainer.Args,
		)
	}

	assertHTTPProbe(
		t,
		"startup",
		runtimeContainer.StartupProbe,
		"/health",
		5,
		2,
		60,
	)

	assertHTTPProbe(
		t,
		"readiness",
		runtimeContainer.ReadinessProbe,
		"/health",
		5,
		2,
		3,
	)

	assertHTTPProbe(
		t,
		"liveness",
		runtimeContainer.LivenessProbe,
		"/health",
		10,
		2,
		3,
	)
}

func assertHTTPProbe(
	t *testing.T,
	name string,
	probe *corev1.Probe,
	expectedPath string,
	expectedPeriod int32,
	expectedTimeout int32,
	expectedFailureThreshold int32,
) {
	t.Helper()

	if probe == nil {
		t.Fatalf("%s probe must not be nil", name)
	}

	if probe.HTTPGet == nil {
		t.Fatalf("%s probe must use HTTP GET", name)
	}

	if probe.HTTPGet.Path != expectedPath {
		t.Fatalf(
			"%s probe expected path %q, got %q",
			name,
			expectedPath,
			probe.HTTPGet.Path,
		)
	}

	if probe.HTTPGet.Port.StrVal != runtimeHTTPPortName {
		t.Fatalf(
			"%s probe expected named port %q, got %q",
			name,
			runtimeHTTPPortName,
			probe.HTTPGet.Port.StrVal,
		)
	}

	if probe.SuccessThreshold != 1 {
		t.Fatalf(
			"%s probe expected success threshold 1, got %d",
			name,
			probe.SuccessThreshold,
		)
	}

	if probe.PeriodSeconds != expectedPeriod {
		t.Fatalf(
			"%s probe expected period %d, got %d",
			name,
			expectedPeriod,
			probe.PeriodSeconds,
		)
	}

	if probe.TimeoutSeconds != expectedTimeout {
		t.Fatalf(
			"%s probe expected timeout %d, got %d",
			name,
			expectedTimeout,
			probe.TimeoutSeconds,
		)
	}

	if probe.FailureThreshold != expectedFailureThreshold {
		t.Fatalf(
			"%s probe expected failure threshold %d, got %d",
			name,
			expectedFailureThreshold,
			probe.FailureThreshold,
		)
	}
}
