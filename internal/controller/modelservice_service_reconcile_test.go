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

var _ = Describe("ModelService Service reconciliation", func() {
	const (
		resourceName      = "service-resource"
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
					Name:    "service-model",
					Version: "v1",
					URI:     "hf://example/service-model",
				},
				Runtime: servingv1alpha1.RuntimeSpec{
					Type:  servingv1alpha1.RuntimeTypeTransformers,
					Image: "example.invalid/runtime:v1",
				},
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

	It("creates, updates, self-heals and rebuilds the Service", func() {
		controllerReconciler := &ModelServiceReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}

		request := reconcile.Request{
			NamespacedName: modelServiceKey,
		}

		By("creating the Service on first reconcile")

		_, err := controllerReconciler.Reconcile(
			ctx,
			request,
		)
		Expect(err).NotTo(HaveOccurred())

		service := &corev1.Service{}

		Eventually(func() error {
			return k8sClient.Get(
				ctx,
				serviceKey,
				service,
			)
		}, "5s", "200ms").Should(Succeed())

		Expect(service.Spec.Type).
			To(Equal(corev1.ServiceTypeClusterIP))

		Expect(service.Spec.Ports).To(HaveLen(1))
		Expect(service.Spec.Ports[0].Port).
			To(Equal(int32(8000)))
		Expect(service.Spec.Ports[0].TargetPort).
			To(Equal(intstr.FromInt32(8000)))

		Expect(
			service.Spec.Selector["serving.modelfleet.io/modelservice"],
		).To(Equal(resourceName))

		_, selectorContainsRuntime :=
			service.Spec.Selector["serving.modelfleet.io/runtime"]

		Expect(selectorContainsRuntime).To(BeFalse())

		Expect(service.OwnerReferences).To(HaveLen(1))
		Expect(service.OwnerReferences[0].Name).
			To(Equal(resourceName))

		firstResourceVersion := service.ResourceVersion
		firstClusterIP := service.Spec.ClusterIP

		By("remaining idempotent on repeated reconcile")

		_, err = controllerReconciler.Reconcile(
			ctx,
			request,
		)
		Expect(err).NotTo(HaveOccurred())

		secondService := &corev1.Service{}

		Expect(
			k8sClient.Get(
				ctx,
				serviceKey,
				secondService,
			),
		).To(Succeed())

		Expect(secondService.ResourceVersion).
			To(Equal(firstResourceVersion))

		Expect(secondService.Spec.ClusterIP).
			To(Equal(firstClusterIP))

		By("updating the Service after ModelService port changes")

		modelService := &servingv1alpha1.ModelService{}

		Expect(
			k8sClient.Get(
				ctx,
				modelServiceKey,
				modelService,
			),
		).To(Succeed())

		modelService.Spec.Port = 9000
		modelService.Spec.Runtime.Type =
			servingv1alpha1.RuntimeTypeVLLM
		modelService.Spec.Runtime.Image =
			"example.invalid/runtime:v2"

		Expect(
			k8sClient.Update(ctx, modelService),
		).To(Succeed())

		_, err = controllerReconciler.Reconcile(
			ctx,
			request,
		)
		Expect(err).NotTo(HaveOccurred())

		updatedService := &corev1.Service{}

		Expect(
			k8sClient.Get(
				ctx,
				serviceKey,
				updatedService,
			),
		).To(Succeed())

		Expect(updatedService.Spec.Ports).To(HaveLen(1))
		Expect(updatedService.Spec.Ports[0].Port).
			To(Equal(int32(9000)))
		Expect(updatedService.Spec.Ports[0].TargetPort).
			To(Equal(intstr.FromInt32(9000)))

		Expect(updatedService.Spec.ClusterIP).
			To(Equal(firstClusterIP))

		Expect(
			updatedService.Labels["serving.modelfleet.io/runtime"],
		).To(Equal("vllm"))

		_, selectorContainsRuntime =
			updatedService.Spec.Selector["serving.modelfleet.io/runtime"]

		Expect(selectorContainsRuntime).To(BeFalse())

		By("restoring manually drifted Service fields")

		driftedService := &corev1.Service{}

		Expect(
			k8sClient.Get(
				ctx,
				serviceKey,
				driftedService,
			),
		).To(Succeed())

		driftedService.Spec.Selector["serving.modelfleet.io/modelservice"] = "rogue"

		driftedService.Spec.Ports[0].Port = 7777
		driftedService.Spec.Ports[0].TargetPort =
			intstr.FromInt32(7777)

		driftedService.Labels["serving.modelfleet.io/runtime"] = "rogue"

		Expect(
			k8sClient.Update(
				ctx,
				driftedService,
			),
		).To(Succeed())

		_, err = controllerReconciler.Reconcile(
			ctx,
			request,
		)
		Expect(err).NotTo(HaveOccurred())

		healedService := &corev1.Service{}

		Expect(
			k8sClient.Get(
				ctx,
				serviceKey,
				healedService,
			),
		).To(Succeed())

		Expect(
			healedService.Spec.Selector["serving.modelfleet.io/modelservice"],
		).To(Equal(resourceName))

		Expect(healedService.Spec.Ports[0].Port).
			To(Equal(int32(9000)))

		Expect(healedService.Spec.Ports[0].TargetPort).
			To(Equal(intstr.FromInt32(9000)))

		Expect(
			healedService.Labels["serving.modelfleet.io/runtime"],
		).To(Equal("vllm"))

		Expect(healedService.Spec.ClusterIP).
			To(Equal(firstClusterIP))

		By("rebuilding the Service after deletion")

		firstUID := healedService.UID

		Expect(
			k8sClient.Delete(
				ctx,
				healedService,
			),
		).To(Succeed())

		Eventually(func() bool {
			current := &corev1.Service{}

			err := k8sClient.Get(
				ctx,
				serviceKey,
				current,
			)

			return errors.IsNotFound(err)
		}, "5s", "200ms").Should(BeTrue())

		_, err = controllerReconciler.Reconcile(
			ctx,
			request,
		)
		Expect(err).NotTo(HaveOccurred())

		rebuiltService := &corev1.Service{}

		Eventually(func() error {
			return k8sClient.Get(
				ctx,
				serviceKey,
				rebuiltService,
			)
		}, "5s", "200ms").Should(Succeed())

		Expect(rebuiltService.UID).
			NotTo(Equal(firstUID))

		Expect(rebuiltService.Spec.Ports[0].Port).
			To(Equal(int32(9000)))

		Expect(
			rebuiltService.Spec.Selector["serving.modelfleet.io/modelservice"],
		).To(Equal(resourceName))
	})
})
