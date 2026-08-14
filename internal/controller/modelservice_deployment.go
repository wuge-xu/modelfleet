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
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	servingv1alpha1 "github.com/wuge-xu/modelfleet/api/v1alpha1"
)

const (
	defaultModelReplicas int32 = 1
	defaultModelPort     int32 = 8000

	runtimeContainerName = "runtime"
	runtimeHTTPPortName  = "http"
)

func buildModelDeployment(
	modelService *servingv1alpha1.ModelService,
	scheme *runtime.Scheme,
) (*appsv1.Deployment, error) {
	adapter, err := resolveRuntimeAdapter(
		modelService.Spec.Runtime.Type,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"resolve runtime adapter: %w",
			err,
		)
	}

	probePaths := adapter.ProbePaths()

	labels := modelDeploymentLabels(modelService)
	selectorLabels := modelSelectorLabels(modelService)

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      modelDeploymentName(modelService),
			Namespace: modelService.Namespace,
			Labels:    labels,
			Annotations: map[string]string{
				"serving.modelfleet.io/model-version": modelService.Spec.Model.Version,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: desiredModelReplicas(modelService),
			Selector: &metav1.LabelSelector{
				MatchLabels: selectorLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
					Annotations: map[string]string{
						"serving.modelfleet.io/model-uri":     modelService.Spec.Model.URI,
						"serving.modelfleet.io/model-version": modelService.Spec.Model.Version,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            runtimeContainerName,
							Image:           modelService.Spec.Runtime.Image,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Args:            adapter.BuildArgs(modelService),
							Ports: []corev1.ContainerPort{
								{
									Name:          runtimeHTTPPortName,
									ContainerPort: desiredModelPort(modelService),
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Resources: *modelService.Spec.Resources.DeepCopy(),

							StartupProbe: buildHTTPProbe(
								probePaths.Startup,
								5,
								2,
								60,
							),

							ReadinessProbe: buildHTTPProbe(
								probePaths.Readiness,
								5,
								2,
								3,
							),

							LivenessProbe: buildHTTPProbe(
								probePaths.Liveness,
								10,
								2,
								3,
							),
						},
					},
				},
			},
		},
	}

	if err := controllerutil.SetControllerReference(
		modelService,
		deployment,
		scheme,
	); err != nil {
		return nil, fmt.Errorf(
			"set ModelService controller reference: %w",
			err,
		)
	}

	return deployment, nil
}

func buildHTTPProbe(
	path string,
	periodSeconds int32,
	timeoutSeconds int32,
	failureThreshold int32,
) *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   path,
				Port:   intstr.FromString(runtimeHTTPPortName),
				Scheme: corev1.URISchemeHTTP,
			},
		},
		PeriodSeconds:    periodSeconds,
		TimeoutSeconds:   timeoutSeconds,
		SuccessThreshold: 1,
		FailureThreshold: failureThreshold,
	}
}

func modelDeploymentName(
	modelService *servingv1alpha1.ModelService,
) string {
	return modelService.Name + "-runtime"
}

func modelSelectorLabels(
	modelService *servingv1alpha1.ModelService,
) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":             "model-runtime",
		"app.kubernetes.io/instance":         modelService.Name,
		"serving.modelfleet.io/modelservice": modelService.Name,
	}
}

func modelDeploymentLabels(
	modelService *servingv1alpha1.ModelService,
) map[string]string {
	labels := modelSelectorLabels(modelService)

	labels["app.kubernetes.io/managed-by"] = "modelfleet"
	labels["serving.modelfleet.io/runtime"] =
		string(modelService.Spec.Runtime.Type)

	return labels
}

func desiredModelReplicas(
	modelService *servingv1alpha1.ModelService,
) *int32 {
	replicas := defaultModelReplicas

	if modelService.Spec.Replicas != nil {
		replicas = *modelService.Spec.Replicas
	}

	return &replicas
}

func desiredModelPort(
	modelService *servingv1alpha1.ModelService,
) int32 {
	if modelService.Spec.Port == 0 {
		return defaultModelPort
	}

	return modelService.Spec.Port
}
