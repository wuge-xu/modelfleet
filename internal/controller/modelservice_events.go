package controller

import (
	corev1 "k8s.io/api/core/v1"

	servingv1alpha1 "github.com/wuge-xu/modelfleet/api/v1alpha1"
)

const (
	eventActionStatusTransition = "StatusTransition"

	eventReasonDeploying = "ModelServiceDeploying"
	eventReasonReady     = "ModelServiceReady"
	eventReasonDegraded  = "ModelServiceDegraded"
)

func (r *ModelServiceReconciler) recordStatusTransitionEvent(
	modelService *servingv1alpha1.ModelService,
	previousPhase servingv1alpha1.ModelServicePhase,
	currentPhase servingv1alpha1.ModelServicePhase,
) {
	if r.Recorder == nil {
		return
	}

	if previousPhase == currentPhase {
		return
	}

	previousPhaseName := string(previousPhase)

	if previousPhaseName == "" {
		previousPhaseName =
			string(servingv1alpha1.ModelServicePhasePending)
	}

	switch currentPhase {
	case servingv1alpha1.ModelServicePhaseDeploying:
		r.Recorder.Eventf(
			modelService,
			nil,
			corev1.EventTypeNormal,
			eventReasonDeploying,
			eventActionStatusTransition,
			"ModelService transitioned from %s to Deploying.",
			previousPhaseName,
		)

	case servingv1alpha1.ModelServicePhaseReady:
		r.Recorder.Eventf(
			modelService,
			nil,
			corev1.EventTypeNormal,
			eventReasonReady,
			eventActionStatusTransition,
			"ModelService transitioned from %s to Ready and can serve traffic.",
			previousPhaseName,
		)

	case servingv1alpha1.ModelServicePhaseDegraded:
		r.Recorder.Eventf(
			modelService,
			nil,
			corev1.EventTypeWarning,
			eventReasonDegraded,
			eventActionStatusTransition,
			"ModelService transitioned from %s to Degraded because the model Deployment exceeded its progress deadline.",
			previousPhaseName,
		)
	}
}
