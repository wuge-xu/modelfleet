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
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	servingv1alpha1 "github.com/wuge-xu/modelfleet/api/v1alpha1"
)

var _ = Describe("ModelService Controller", func() {
	Context("When reconciling a ModelService", func() {
		const (
			resourceName      = "test-resource"
			resourceNamespace = "default"

			initialImage = "ghcr.io/wuge-xu/modelfleet-tiny-runtime:v0.1.0"
			updatedImage = "ghcr.io/wuge-xu/modelfleet-vllm-runtime:v0.2.0"
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
			resource := &servingv1alpha1.ModelService{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: resourceNamespace,
				},
				Spec: servingv1alpha1.ModelServiceSpec{
					Model: servingv1alpha1.ModelSpec{
						Name:    "test-model",
						Version: "v1",
						URI:     "hf://sshleifer/tiny-gpt2",
					},
					Runtime: servingv1alpha1.RuntimeSpec{
						Type:  servingv1alpha1.RuntimeTypeTransformers,
						Image: initialImage,
					},
				},
			}

			Expect(
				k8sClient.Create(ctx, resource),
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

			modelService :=
				&servingv1alpha1.ModelService{}

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

		It("creates, updates and self-heals the managed Deployment", func() {
			controllerReconciler := &ModelServiceReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			request := reconcile.Request{
				NamespacedName: modelServiceKey,
			}

			By("creating Deployment on first reconcile")

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

			Expect(deployment.Spec.Replicas).
				NotTo(BeNil())

			Expect(*deployment.Spec.Replicas).
				To(Equal(int32(1)))

			runtimeContainer :=
				deployment.Spec.Template.Spec.Containers[0]

			Expect(runtimeContainer.Image).
				To(Equal(initialImage))

			firstResourceVersion :=
				deployment.ResourceVersion

			By("remaining idempotent on repeated reconcile")

			_, err = controllerReconciler.Reconcile(
				ctx,
				request,
			)
			Expect(err).NotTo(HaveOccurred())

			secondDeployment := &appsv1.Deployment{}

			Expect(
				k8sClient.Get(
					ctx,
					deploymentKey,
					secondDeployment,
				),
			).To(Succeed())

			Expect(
				secondDeployment.ResourceVersion,
			).To(Equal(firstResourceVersion))

			By("updating Deployment after ModelService spec changes")

			modelService :=
				&servingv1alpha1.ModelService{}

			Expect(
				k8sClient.Get(
					ctx,
					modelServiceKey,
					modelService,
				),
			).To(Succeed())

			replicas := int32(3)

			modelService.Spec.Replicas = &replicas
			modelService.Spec.Port = 9000
			modelService.Spec.Model.Version = "v2"
			modelService.Spec.Model.URI =
				"hf://example/test-model-v2"

			modelService.Spec.Runtime.Type =
				servingv1alpha1.RuntimeTypeVLLM

			modelService.Spec.Runtime.Image =
				updatedImage

			modelService.Spec.Runtime.Args = []string{
				"--tensor-parallel-size=1",
			}

			Expect(
				k8sClient.Update(
					ctx,
					modelService,
				),
			).To(Succeed())

			_, err = controllerReconciler.Reconcile(
				ctx,
				request,
			)
			Expect(err).NotTo(HaveOccurred())

			updatedDeployment := &appsv1.Deployment{}

			Expect(
				k8sClient.Get(
					ctx,
					deploymentKey,
					updatedDeployment,
				),
			).To(Succeed())

			Expect(updatedDeployment.Spec.Replicas).
				NotTo(BeNil())

			Expect(*updatedDeployment.Spec.Replicas).
				To(Equal(int32(3)))

			updatedRuntime :=
				updatedDeployment.Spec.Template.Spec.Containers[0]

			Expect(updatedRuntime.Image).
				To(Equal(updatedImage))

			Expect(updatedRuntime.Args).To(Equal(
				[]string{
					"--model",
					"example/test-model-v2",
					"--port",
					"9000",
					"--tensor-parallel-size=1",
				},
			))

			Expect(updatedRuntime.Ports).To(HaveLen(1))

			Expect(
				updatedRuntime.Ports[0].ContainerPort,
			).To(Equal(int32(9000)))

			Expect(
				updatedDeployment.Spec.Template.Labels["serving.modelfleet.io/runtime"],
			).To(Equal("vllm"))

			Expect(
				updatedDeployment.Annotations["serving.modelfleet.io/model-version"],
			).To(Equal("v2"))

			_, selectorContainsRuntime :=
				updatedDeployment.Spec.Selector.MatchLabels["serving.modelfleet.io/runtime"]

			Expect(selectorContainsRuntime).
				To(BeFalse())

			By("restoring manually drifted Deployment fields")

			driftedDeployment := &appsv1.Deployment{}

			Expect(
				k8sClient.Get(
					ctx,
					deploymentKey,
					driftedDeployment,
				),
			).To(Succeed())

			driftedReplicas := int32(9)

			driftedDeployment.Spec.Replicas =
				&driftedReplicas

			driftedDeployment.
				Spec.
				Template.
				Spec.
				Containers[0].
				Image = "example.invalid/rogue-runtime:latest"

			driftedDeployment.
				Spec.
				Template.
				Spec.
				Containers[0].
				Args = []string{"--rogue"}

			driftedDeployment.Spec.Template.Labels["serving.modelfleet.io/runtime"] = "rogue"

			driftedDeployment.Annotations["serving.modelfleet.io/model-version"] = "rogue"

			Expect(
				k8sClient.Update(
					ctx,
					driftedDeployment,
				),
			).To(Succeed())

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

			Expect(healedDeployment.Spec.Replicas).
				NotTo(BeNil())

			Expect(*healedDeployment.Spec.Replicas).
				To(Equal(int32(3)))

			healedRuntime :=
				healedDeployment.Spec.Template.Spec.Containers[0]

			Expect(healedRuntime.Image).
				To(Equal(updatedImage))

			Expect(healedRuntime.Args).To(Equal(
				[]string{
					"--model",
					"example/test-model-v2",
					"--port",
					"9000",
					"--tensor-parallel-size=1",
				},
			))

			Expect(
				healedDeployment.Spec.Template.Labels["serving.modelfleet.io/runtime"],
			).To(Equal("vllm"))

			Expect(
				healedDeployment.Annotations["serving.modelfleet.io/model-version"],
			).To(Equal("v2"))
		})
	})
})
