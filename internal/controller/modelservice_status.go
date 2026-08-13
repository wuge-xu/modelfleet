package controller

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	servingv1alpha1 "github.com/wuge-xu/modelfleet/api/v1alpha1"
)

const (
	conditionTypeReady       = "Ready"
	conditionTypeProgressing = "Progressing"
	conditionTypeDegraded    = "Degraded"
)

func buildModelServiceStatus(
	modelService *servingv1alpha1.ModelService,
	deployment *appsv1.Deployment,
) servingv1alpha1.ModelServiceStatus {
	status := servingv1alpha1.ModelServiceStatus{
		ObservedGeneration: modelService.Generation,
		Phase:              servingv1alpha1.ModelServicePhaseDeploying,
		DeploymentName:     modelDeploymentName(modelService),
		ServiceName:        modelService.Name + modelServiceSuffix,
		Conditions: append(
			[]metav1.Condition(nil),
			modelService.Status.Conditions...,
		),
	}

	if deployment != nil {
		status.ReadyReplicas = deployment.Status.ReadyReplicas
		status.AvailableReplicas = deployment.Status.AvailableReplicas
	}

	desiredReplicas := *desiredModelReplicas(modelService)

	degraded := deploymentProgressDeadlineExceeded(deployment)

	ready := deployment != nil &&
		deployment.Status.ReadyReplicas >= desiredReplicas &&
		deployment.Status.AvailableReplicas >= desiredReplicas

	switch {
	case degraded:
		status.Phase = servingv1alpha1.ModelServicePhaseDegraded

		setModelServiceCondition(
			&status,
			modelService.Generation,
			conditionTypeReady,
			metav1.ConditionFalse,
			"ProgressDeadlineExceeded",
			"Model Deployment failed to become ready before its progress deadline.",
		)

		setModelServiceCondition(
			&status,
			modelService.Generation,
			conditionTypeProgressing,
			metav1.ConditionFalse,
			"ProgressDeadlineExceeded",
			"Model Deployment is no longer progressing successfully.",
		)

		setModelServiceCondition(
			&status,
			modelService.Generation,
			conditionTypeDegraded,
			metav1.ConditionTrue,
			"ProgressDeadlineExceeded",
			"Model Deployment exceeded its progress deadline.",
		)

	case ready:
		status.Phase = servingv1alpha1.ModelServicePhaseReady

		setModelServiceCondition(
			&status,
			modelService.Generation,
			conditionTypeReady,
			metav1.ConditionTrue,
			"ReplicasReady",
			"All desired model replicas are ready and available.",
		)

		setModelServiceCondition(
			&status,
			modelService.Generation,
			conditionTypeProgressing,
			metav1.ConditionFalse,
			"ReconcileComplete",
			"Model runtime has reached the desired ready state.",
		)

		setModelServiceCondition(
			&status,
			modelService.Generation,
			conditionTypeDegraded,
			metav1.ConditionFalse,
			"Healthy",
			"No degraded model runtime condition is present.",
		)

	default:
		status.Phase = servingv1alpha1.ModelServicePhaseDeploying

		setModelServiceCondition(
			&status,
			modelService.Generation,
			conditionTypeReady,
			metav1.ConditionFalse,
			"ReplicasNotReady",
			"Model runtime has not reached the desired ready replica count.",
		)

		setModelServiceCondition(
			&status,
			modelService.Generation,
			conditionTypeProgressing,
			metav1.ConditionTrue,
			"Reconciling",
			"Model runtime is progressing toward the desired state.",
		)

		setModelServiceCondition(
			&status,
			modelService.Generation,
			conditionTypeDegraded,
			metav1.ConditionFalse,
			"Progressing",
			"Model runtime is still progressing and is not degraded.",
		)
	}

	return status
}

func setModelServiceCondition(
	status *servingv1alpha1.ModelServiceStatus,
	observedGeneration int64,
	conditionType string,
	conditionStatus metav1.ConditionStatus,
	reason string,
	message string,
) {
	apimeta.SetStatusCondition(
		&status.Conditions,
		metav1.Condition{
			Type:               conditionType,
			Status:             conditionStatus,
			Reason:             reason,
			Message:            message,
			ObservedGeneration: observedGeneration,
			LastTransitionTime: metav1.Now(),
		},
	)
}

func deploymentProgressDeadlineExceeded(
	deployment *appsv1.Deployment,
) bool {
	if deployment == nil {
		return false
	}

	for _, condition := range deployment.Status.Conditions {
		if condition.Type == appsv1.DeploymentProgressing &&
			condition.Status == corev1.ConditionFalse &&
			condition.Reason == "ProgressDeadlineExceeded" {
			return true
		}
	}

	return false
}
