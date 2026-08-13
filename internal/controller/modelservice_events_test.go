package controller

import (
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	events "k8s.io/client-go/tools/events"

	servingv1alpha1 "github.com/wuge-xu/modelfleet/api/v1alpha1"
)

func TestRecordStatusTransitionEvents(t *testing.T) {
	modelService := &servingv1alpha1.ModelService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "event-model",
			Namespace: "default",
			UID:       "event-model-uid",
		},
	}

	tests := []struct {
		name          string
		previousPhase servingv1alpha1.ModelServicePhase
		currentPhase  servingv1alpha1.ModelServicePhase
		expected      string
	}{
		{
			name:          "initial deploying",
			previousPhase: "",
			currentPhase: servingv1alpha1.
				ModelServicePhaseDeploying,
			expected: "Normal ModelServiceDeploying",
		},
		{
			name: "ready",
			previousPhase: servingv1alpha1.
				ModelServicePhaseDeploying,
			currentPhase: servingv1alpha1.
				ModelServicePhaseReady,
			expected: "Normal ModelServiceReady",
		},
		{
			name: "degraded",
			previousPhase: servingv1alpha1.
				ModelServicePhaseReady,
			currentPhase: servingv1alpha1.
				ModelServicePhaseDegraded,
			expected: "Warning ModelServiceDegraded",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := events.NewFakeRecorder(1)

			controllerReconciler :=
				&ModelServiceReconciler{
					Recorder: recorder,
				}

			controllerReconciler.
				recordStatusTransitionEvent(
					modelService,
					test.previousPhase,
					test.currentPhase,
				)

			event := <-recorder.Events

			if !strings.Contains(
				event,
				test.expected,
			) {
				t.Fatalf(
					"expected event containing %q, got %q",
					test.expected,
					event,
				)
			}
		})
	}
}

func TestRecordStatusTransitionEventSkipsUnchangedPhase(
	t *testing.T,
) {
	recorder := events.NewFakeRecorder(1)

	controllerReconciler :=
		&ModelServiceReconciler{
			Recorder: recorder,
		}

	modelService := &servingv1alpha1.ModelService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "event-model",
			Namespace: "default",
			UID:       "event-model-uid",
		},
	}

	controllerReconciler.
		recordStatusTransitionEvent(
			modelService,
			servingv1alpha1.ModelServicePhaseReady,
			servingv1alpha1.ModelServicePhaseReady,
		)

	select {
	case event := <-recorder.Events:
		t.Fatalf(
			"expected no event for unchanged phase, got %q",
			event,
		)

	default:
	}
}
