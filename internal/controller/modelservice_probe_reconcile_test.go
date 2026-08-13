package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	servingv1alpha1 "github.com/wuge-xu/modelfleet/api/v1alpha1"
)

var _ = Describe("ModelService probe reconciliation", func() {
	const (
		resourceName      = "probe-resource"
		resourceNamespace = "default"
	)

	ctx := context.Background()

	modelServiceKey := types.NamespacedName{
		Name:      resourceName,
		Namespace: resourceNamespace,
	}

	deploymentKey := types.NamespacedName{
		Name:      resourceName + "-runtime",
		Namespace: resourceNamespace,
	}

	serviceKey := types.NamespacedName{
		Name:      resourceName + "-service",
		Namespace: resourceNamespace,
	}

	BeforeEach(func() {
		modelService := &servingv1alpha1.ModelService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resourceName,
				Namespace: resourceNamespace,
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
				},
				Port: 9000,
			},
		}

		Expect(
			k8sClient.Create(ctx, modelService),
		).To(Succeed())
	})

	AfterEach(func() {
		service := &corev1.Service{}

		err := k8sClient.Get(
			ctx,
			serviceKey,
			service,
		)

		if err == nil {
			Expect(
				k8sClient.Delete(ctx, service),
			).To(Succeed())
		} else {
			Expect(errors.IsNotFound(err)).To(BeTrue())
		}

		deployment := &appsv1.Deployment{}

		err = k8sClient.Get(
			ctx,
			deploymentKey,
			deployment,
		)

		if err == nil {
			Expect(
				k8sClient.Delete(ctx, deployment),
			).To(Succeed())
		} else {
			Expect(errors.IsNotFound(err)).To(BeTrue())
		}

		modelService := &servingv1alpha1.ModelService{}

		err = k8sClient.Get(
			ctx,
			modelServiceKey,
			modelService,
		)

		if err == nil {
			Expect(
				k8sClient.Delete(ctx, modelService),
			).To(Succeed())
		} else {
			Expect(errors.IsNotFound(err)).To(BeTrue())
		}
	})

	It("creates and self-heals runtime probes", func() {
		controllerReconciler := &ModelServiceReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}

		request := reconcile.Request{
			NamespacedName: modelServiceKey,
		}

		By("creating the Deployment with all probes")

		_, err := controllerReconciler.Reconcile(
			ctx,
			request,
		)
		Expect(err).NotTo(HaveOccurred())

		deployment := &appsv1.Deployment{}

		Eventually(func() error {
			return k8sClient.Get(
				ctx,
				deploymentKey,
				deployment,
			)
		}, "5s", "200ms").Should(Succeed())

		runtimeContainer :=
			&deployment.Spec.Template.Spec.Containers[0]

		expectRuntimeProbes(runtimeContainer)

		By("manually removing and corrupting probe configuration")

		runtimeContainer.StartupProbe = nil

		runtimeContainer.ReadinessProbe = &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/wrong-ready",
					Port: intstr.FromInt32(7777),
				},
			},
			PeriodSeconds:    99,
			TimeoutSeconds:   99,
			FailureThreshold: 99,
		}

		runtimeContainer.LivenessProbe = &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/wrong-health",
					Port: intstr.FromInt32(7777),
				},
			},
			PeriodSeconds:    99,
			TimeoutSeconds:   99,
			FailureThreshold: 99,
		}

		Expect(
			k8sClient.Update(
				ctx,
				deployment,
			),
		).To(Succeed())

		By("reconciling the ModelService again")

		_, err = controllerReconciler.Reconcile(
			ctx,
			request,
		)
		Expect(err).NotTo(HaveOccurred())

		healedDeployment := &appsv1.Deployment{}

		Expect(
			k8sClient.Get(
				ctx,
				deploymentKey,
				healedDeployment,
			),
		).To(Succeed())

		healedRuntime :=
			&healedDeployment.Spec.Template.Spec.Containers[0]

		expectRuntimeProbes(healedRuntime)
	})
})

func expectRuntimeProbes(
	runtimeContainer *corev1.Container,
) {
	Expect(runtimeContainer.StartupProbe).
		NotTo(BeNil())

	Expect(runtimeContainer.StartupProbe.HTTPGet).
		NotTo(BeNil())

	Expect(runtimeContainer.StartupProbe.HTTPGet.Path).
		To(Equal(startupProbePath))

	Expect(runtimeContainer.StartupProbe.HTTPGet.Port).
		To(Equal(intstr.FromString(runtimeHTTPPortName)))

	Expect(runtimeContainer.StartupProbe.PeriodSeconds).
		To(Equal(int32(5)))

	Expect(runtimeContainer.StartupProbe.TimeoutSeconds).
		To(Equal(int32(2)))

	Expect(runtimeContainer.StartupProbe.FailureThreshold).
		To(Equal(int32(60)))

	Expect(runtimeContainer.ReadinessProbe).
		NotTo(BeNil())

	Expect(runtimeContainer.ReadinessProbe.HTTPGet).
		NotTo(BeNil())

	Expect(runtimeContainer.ReadinessProbe.HTTPGet.Path).
		To(Equal(readinessProbePath))

	Expect(runtimeContainer.ReadinessProbe.HTTPGet.Port).
		To(Equal(intstr.FromString(runtimeHTTPPortName)))

	Expect(runtimeContainer.ReadinessProbe.PeriodSeconds).
		To(Equal(int32(5)))

	Expect(runtimeContainer.ReadinessProbe.TimeoutSeconds).
		To(Equal(int32(2)))

	Expect(runtimeContainer.ReadinessProbe.FailureThreshold).
		To(Equal(int32(3)))

	Expect(runtimeContainer.LivenessProbe).
		NotTo(BeNil())

	Expect(runtimeContainer.LivenessProbe.HTTPGet).
		NotTo(BeNil())

	Expect(runtimeContainer.LivenessProbe.HTTPGet.Path).
		To(Equal(livenessProbePath))

	Expect(runtimeContainer.LivenessProbe.HTTPGet.Port).
		To(Equal(intstr.FromString(runtimeHTTPPortName)))

	Expect(runtimeContainer.LivenessProbe.PeriodSeconds).
		To(Equal(int32(10)))

	Expect(runtimeContainer.LivenessProbe.TimeoutSeconds).
		To(Equal(int32(2)))

	Expect(runtimeContainer.LivenessProbe.FailureThreshold).
		To(Equal(int32(3)))
}
