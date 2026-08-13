package controller

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	servingv1alpha1 "github.com/wuge-xu/modelfleet/api/v1alpha1"
)

func TestBuildModelServiceStatusDeploying(t *testing.T) {
	replicas := int32(2)

	modelService := &servingv1alpha1.ModelService{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "demo",
			Namespace:  "default",
			Generation: 3,
		},
		Spec: servingv1alpha1.ModelServiceSpec{
			Replicas: &replicas,
		},
	}

	deployment := &appsv1.Deployment{
		Status: appsv1.DeploymentStatus{
			ReadyReplicas:     1,
			AvailableReplicas: 1,
		},
	}

	status := buildModelServiceStatus(
		modelService,
		deployment,
	)

	if status.Phase != servingv1alpha1.ModelServicePhaseDeploying {
		t.Fatalf(
			"expected Deploying phase, got %s",
			status.Phase,
		)
	}

	if status.ObservedGeneration != 3 {
		t.Fatalf(
			"expected observedGeneration 3, got %d",
			status.ObservedGeneration,
		)
	}

	assertCondition(
		t,
		status.Conditions,
		conditionTypeReady,
		metav1.ConditionFalse,
		"ReplicasNotReady",
	)

	assertCondition(
		t,
		status.Conditions,
		conditionTypeProgressing,
		metav1.ConditionTrue,
		"Reconciling",
	)
}

func TestBuildModelServiceStatusReady(t *testing.T) {
	replicas := int32(2)

	modelService := &servingv1alpha1.ModelService{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "demo",
			Namespace:  "default",
			Generation: 4,
		},
		Spec: servingv1alpha1.ModelServiceSpec{
			Replicas: &replicas,
		},
	}

	deployment := &appsv1.Deployment{
		Status: appsv1.DeploymentStatus{
			ReadyReplicas:     2,
			AvailableReplicas: 2,
		},
	}

	status := buildModelServiceStatus(
		modelService,
		deployment,
	)

	if status.Phase != servingv1alpha1.ModelServicePhaseReady {
		t.Fatalf(
			"expected Ready phase, got %s",
			status.Phase,
		)
	}

	if status.ReadyReplicas != 2 {
		t.Fatalf(
			"expected 2 ready replicas, got %d",
			status.ReadyReplicas,
		)
	}

	assertCondition(
		t,
		status.Conditions,
		conditionTypeReady,
		metav1.ConditionTrue,
		"ReplicasReady",
	)

	assertCondition(
		t,
		status.Conditions,
		conditionTypeDegraded,
		metav1.ConditionFalse,
		"Healthy",
	)
}

func TestBuildModelServiceStatusDegraded(t *testing.T) {
	replicas := int32(2)

	modelService := &servingv1alpha1.ModelService{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "demo",
			Namespace:  "default",
			Generation: 5,
		},
		Spec: servingv1alpha1.ModelServiceSpec{
			Replicas: &replicas,
		},
	}

	deployment := &appsv1.Deployment{
		Status: appsv1.DeploymentStatus{
			ReadyReplicas:     0,
			AvailableReplicas: 0,
			Conditions: []appsv1.DeploymentCondition{
				{
					Type:   appsv1.DeploymentProgressing,
					Status: corev1.ConditionFalse,
					Reason: "ProgressDeadlineExceeded",
				},
			},
		},
	}

	status := buildModelServiceStatus(
		modelService,
		deployment,
	)

	if status.Phase != servingv1alpha1.ModelServicePhaseDegraded {
		t.Fatalf(
			"expected Degraded phase, got %s",
			status.Phase,
		)
	}

	assertCondition(
		t,
		status.Conditions,
		conditionTypeDegraded,
		metav1.ConditionTrue,
		"ProgressDeadlineExceeded",
	)
}

func assertCondition(
	t *testing.T,
	conditions []metav1.Condition,
	conditionType string,
	expectedStatus metav1.ConditionStatus,
	expectedReason string,
) {
	t.Helper()

	for _, condition := range conditions {
		if condition.Type != conditionType {
			continue
		}

		if condition.Status != expectedStatus {
			t.Fatalf(
				"condition %s expected status %s, got %s",
				conditionType,
				expectedStatus,
				condition.Status,
			)
		}

		if condition.Reason != expectedReason {
			t.Fatalf(
				"condition %s expected reason %s, got %s",
				conditionType,
				expectedReason,
				condition.Reason,
			)
		}

		return
	}

	t.Fatalf(
		"condition %s not found",
		conditionType,
	)
}
