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
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	servingv1alpha1 "github.com/wuge-xu/modelfleet/api/v1alpha1"
)

// ModelServiceReconciler reconciles a ModelService object.
type ModelServiceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=serving.modelfleet.io,resources=modelservices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=serving.modelfleet.io,resources=modelservices/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=serving.modelfleet.io,resources=modelservices/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

// Reconcile moves the actual Kubernetes state toward the desired ModelService state.
func (r *ModelServiceReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)

	modelService := &servingv1alpha1.ModelService{}

	if err := r.Get(ctx, req.NamespacedName, modelService); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, fmt.Errorf(
			"get ModelService %s: %w",
			req.NamespacedName,
			err,
		)
	}

	desiredDeployment, err := buildModelDeployment(
		modelService,
		r.Scheme,
	)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf(
			"build Deployment for ModelService %s: %w",
			req.NamespacedName,
			err,
		)
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      desiredDeployment.Name,
			Namespace: desiredDeployment.Namespace,
		},
	}

	operation, err := controllerutil.CreateOrUpdate(
		ctx,
		r.Client,
		deployment,
		func() error {
			return applyDesiredDeployment(
				modelService,
				desiredDeployment,
				deployment,
				r.Scheme,
			)
		},
	)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf(
			"reconcile Deployment %s/%s: %w",
			deployment.Namespace,
			deployment.Name,
			err,
		)
	}

	logger.Info(
		"Reconciled model Deployment",
		"modelService", req.NamespacedName,
		"deployment", client.ObjectKeyFromObject(deployment),
		"operation", operation,
	)

	return ctrl.Result{}, nil
}

func applyDesiredDeployment(
	modelService *servingv1alpha1.ModelService,
	desired *appsv1.Deployment,
	actual *appsv1.Deployment,
	scheme *runtime.Scheme,
) error {
	if err := controllerutil.SetControllerReference(
		modelService,
		actual,
		scheme,
	); err != nil {
		return fmt.Errorf(
			"set ModelService controller reference: %w",
			err,
		)
	}

	actual.Labels = copyStringMap(desired.Labels)
	actual.Annotations = copyStringMap(desired.Annotations)
	actual.Spec.Replicas = desired.Spec.Replicas

	// Deployment selector is immutable after creation.
	if actual.Spec.Selector == nil {
		actual.Spec.Selector = desired.Spec.Selector.DeepCopy()
	}

	actual.Spec.Template.Labels =
		copyStringMap(desired.Spec.Template.Labels)

	actual.Spec.Template.Annotations =
		copyStringMap(desired.Spec.Template.Annotations)

	desiredRuntime := desired.Spec.Template.Spec.Containers[0]
	actualRuntime := findRuntimeContainer(actual)

	if actualRuntime == nil {
		actual.Spec.Template.Spec.Containers = append(
			actual.Spec.Template.Spec.Containers,
			*desiredRuntime.DeepCopy(),
		)

		return nil
	}

	actualRuntime.Image = desiredRuntime.Image
	actualRuntime.ImagePullPolicy = desiredRuntime.ImagePullPolicy
	actualRuntime.Args = append(
		[]string(nil),
		desiredRuntime.Args...,
	)
	actualRuntime.Ports = append(
		[]corev1.ContainerPort(nil),
		desiredRuntime.Ports...,
	)
	actualRuntime.Resources =
		*desiredRuntime.Resources.DeepCopy()

	return nil
}

func findRuntimeContainer(
	deployment *appsv1.Deployment,
) *corev1.Container {
	for index := range deployment.Spec.Template.Spec.Containers {
		container :=
			&deployment.Spec.Template.Spec.Containers[index]

		if container.Name == runtimeContainerName {
			return container
		}
	}

	return nil
}

func copyStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}

	result := make(map[string]string, len(source))

	for key, value := range source {
		result[key] = value
	}

	return result
}

// SetupWithManager sets up the controller with the Manager.
func (r *ModelServiceReconciler) SetupWithManager(
	mgr ctrl.Manager,
) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&servingv1alpha1.ModelService{}).
		Owns(&appsv1.Deployment{}).
		Named("modelservice").
		Complete(r)
}
