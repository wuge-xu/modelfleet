package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	servingv1alpha1 "github.com/wuge-xu/modelfleet/api/v1alpha1"
)

var _ = Describe("ModelService status reconciliation", func() {
	const (
		resourceName      = "status-resource"
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
					Name:    "status-model",
					Version: "v1",
					URI:     "hf://example/status-model",
				},
				Runtime: servingv1alpha1.RuntimeSpec{
					Type:  servingv1alpha1.RuntimeTypeVLLM,
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

	It("transitions through Deploying, Ready and Degraded idempotently", func() {
		controllerReconciler := &ModelServiceReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}

		request := reconcile.Request{
			NamespacedName: modelServiceKey,
		}

		By("writing Deploying status on initial reconcile")

		_, err := controllerReconciler.Reconcile(
			ctx,
			request,
		)
		Expect(err).NotTo(HaveOccurred())

		deployingModelService :=
			&servingv1alpha1.ModelService{}

		Expect(
			k8sClient.Get(
				ctx,
				modelServiceKey,
				deployingModelService,
			),
		).To(Succeed())

		Expect(deployingModelService.Status.Phase).
			To(Equal(
				servingv1alpha1.ModelServicePhaseDeploying,
			))

		Expect(
			deployingModelService.Status.ObservedGeneration,
		).To(Equal(deployingModelService.Generation))

		Expect(
			deployingModelService.Status.DeploymentName,
		).To(Equal(resourceName + "-runtime"))

		Expect(
			deployingModelService.Status.ServiceName,
		).To(Equal(resourceName + "-service"))

		readyCondition := apimeta.FindStatusCondition(
			deployingModelService.Status.Conditions,
			conditionTypeReady,
		)

		Expect(readyCondition).NotTo(BeNil())
		Expect(readyCondition.Status).
			To(Equal(metav1.ConditionFalse))
		Expect(readyCondition.Reason).
			To(Equal("ReplicasNotReady"))

		progressingCondition :=
			apimeta.FindStatusCondition(
				deployingModelService.Status.Conditions,
				conditionTypeProgressing,
			)

		Expect(progressingCondition).NotTo(BeNil())
		Expect(progressingCondition.Status).
			To(Equal(metav1.ConditionTrue))

		By("simulating a ready Deployment")

		deployment := &appsv1.Deployment{}

		Expect(
			k8sClient.Get(
				ctx,
				deploymentKey,
				deployment,
			),
		).To(Succeed())

		deployment.Status.Replicas = 1
		deployment.Status.UpdatedReplicas = 1
		deployment.Status.ReadyReplicas = 1
		deployment.Status.AvailableReplicas = 1

		Expect(
			k8sClient.Status().Update(
				ctx,
				deployment,
			),
		).To(Succeed())

		_, err = controllerReconciler.Reconcile(
			ctx,
			request,
		)
		Expect(err).NotTo(HaveOccurred())

		readyModelService :=
			&servingv1alpha1.ModelService{}

		Expect(
			k8sClient.Get(
				ctx,
				modelServiceKey,
				readyModelService,
			),
		).To(Succeed())

		Expect(readyModelService.Status.Phase).
			To(Equal(
				servingv1alpha1.ModelServicePhaseReady,
			))

		Expect(readyModelService.Status.ReadyReplicas).
			To(Equal(int32(1)))

		Expect(readyModelService.Status.AvailableReplicas).
			To(Equal(int32(1)))

		readyCondition =
			apimeta.FindStatusCondition(
				readyModelService.Status.Conditions,
				conditionTypeReady,
			)

		Expect(readyCondition).NotTo(BeNil())
		Expect(readyCondition.Status).
			To(Equal(metav1.ConditionTrue))
		Expect(readyCondition.Reason).
			To(Equal("ReplicasReady"))

		degradedCondition :=
			apimeta.FindStatusCondition(
				readyModelService.Status.Conditions,
				conditionTypeDegraded,
			)

		Expect(degradedCondition).NotTo(BeNil())
		Expect(degradedCondition.Status).
			To(Equal(metav1.ConditionFalse))

		readyResourceVersion :=
			readyModelService.ResourceVersion

		readyTransitionTime :=
			readyCondition.LastTransitionTime

		By("remaining idempotent when status has not changed")

		_, err = controllerReconciler.Reconcile(
			ctx,
			request,
		)
		Expect(err).NotTo(HaveOccurred())

		idempotentModelService :=
			&servingv1alpha1.ModelService{}

		Expect(
			k8sClient.Get(
				ctx,
				modelServiceKey,
				idempotentModelService,
			),
		).To(Succeed())

		Expect(
			idempotentModelService.ResourceVersion,
		).To(Equal(readyResourceVersion))

		idempotentReadyCondition :=
			apimeta.FindStatusCondition(
				idempotentModelService.Status.Conditions,
				conditionTypeReady,
			)

		Expect(idempotentReadyCondition).NotTo(BeNil())

		Expect(
			idempotentReadyCondition.LastTransitionTime,
		).To(Equal(readyTransitionTime))

		By("simulating a Deployment progress deadline failure")

		degradedDeployment := &appsv1.Deployment{}

		Expect(
			k8sClient.Get(
				ctx,
				deploymentKey,
				degradedDeployment,
			),
		).To(Succeed())

		degradedDeployment.Status.Replicas = 1
		degradedDeployment.Status.UpdatedReplicas = 0
		degradedDeployment.Status.ReadyReplicas = 0
		degradedDeployment.Status.AvailableReplicas = 0
		degradedDeployment.Status.UnavailableReplicas = 1

		degradedDeployment.Status.Conditions =
			[]appsv1.DeploymentCondition{
				{
					Type:   appsv1.DeploymentProgressing,
					Status: corev1.ConditionFalse,
					Reason: "ProgressDeadlineExceeded",
				},
			}

		Expect(
			k8sClient.Status().Update(
				ctx,
				degradedDeployment,
			),
		).To(Succeed())

		_, err = controllerReconciler.Reconcile(
			ctx,
			request,
		)
		Expect(err).NotTo(HaveOccurred())

		degradedModelService :=
			&servingv1alpha1.ModelService{}

		Expect(
			k8sClient.Get(
				ctx,
				modelServiceKey,
				degradedModelService,
			),
		).To(Succeed())

		Expect(degradedModelService.Status.Phase).
			To(Equal(
				servingv1alpha1.ModelServicePhaseDegraded,
			))

		degradedCondition =
			apimeta.FindStatusCondition(
				degradedModelService.Status.Conditions,
				conditionTypeDegraded,
			)

		Expect(degradedCondition).NotTo(BeNil())
		Expect(degradedCondition.Status).
			To(Equal(metav1.ConditionTrue))
		Expect(degradedCondition.Reason).
			To(Equal("ProgressDeadlineExceeded"))

		readyCondition =
			apimeta.FindStatusCondition(
				degradedModelService.Status.Conditions,
				conditionTypeReady,
			)

		Expect(readyCondition).NotTo(BeNil())
		Expect(readyCondition.Status).
			To(Equal(metav1.ConditionFalse))
		Expect(readyCondition.Reason).
			To(Equal("ProgressDeadlineExceeded"))
	})
})
