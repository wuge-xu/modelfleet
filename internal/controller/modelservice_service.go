package controller

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	servingv1alpha1 "github.com/wuge-xu/modelfleet/api/v1alpha1"
)

const (
	modelServiceSuffix = "-service"
	servicePortName    = "http"
)

func buildModelService(
	modelService *servingv1alpha1.ModelService,
	scheme *runtime.Scheme,
) (*corev1.Service, error) {
	port := modelService.Spec.Port
	if port == 0 {
		port = 8000
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      modelService.Name + modelServiceSuffix,
			Namespace: modelService.Namespace,
			Labels:    modelDeploymentLabels(modelService),
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: copyStringMap(
				modelSelectorLabels(modelService),
			),
			Ports: []corev1.ServicePort{
				{
					Name:       servicePortName,
					Protocol:   corev1.ProtocolTCP,
					Port:       port,
					TargetPort: intstr.FromInt32(port),
				},
			},
		},
	}

	if err := controllerutil.SetControllerReference(
		modelService,
		service,
		scheme,
	); err != nil {
		return nil, fmt.Errorf(
			"set ModelService controller reference on Service: %w",
			err,
		)
	}

	return service, nil
}
