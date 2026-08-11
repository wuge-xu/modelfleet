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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	servingv1alpha1 "github.com/wuge-xu/modelfleet/api/v1alpha1"
)

var _ = Describe("ModelService Deployment recovery", func() {
	const (
		resourceName      = "rebuild-resource"
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

	BeforeEach(func() {
		modelService := &servingv1alpha1.ModelService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resourceName,
				Namespace: resourceNamespace,
			},
			Spec: servingv1alpha1.ModelServiceSpec{
				Model: servingv1alpha1.ModelSpec{
					Name:    "rebuild-model",
					Version: "v1",
					URI:     "hf://example/rebuild-model",
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
		deployment := &appsv1.Deployment{}

		err := k8sClient.Get(
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

	It("recreates the Deployment after deletion", func() {
		controllerReconciler := &ModelServiceReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}

		request := reconcile.Request{
			NamespacedName: modelServiceKey,
		}

		By("creating the initial Deployment")

		_, err := controllerReconciler.Reconcile(
			ctx,
			request,
		)
		Expect(err).NotTo(HaveOccurred())

		deployment := &appsv1.Deployment{}

		Expect(
			k8sClient.Get(
				ctx,
				deploymentKey,
				deployment,
			),
		).To(Succeed())

		firstUID := deployment.UID

		By("deleting the managed Deployment")

		Expect(
			k8sClient.Delete(ctx, deployment),
		).To(Succeed())

		Eventually(func() bool {
			current := &appsv1.Deployment{}

			err := k8sClient.Get(
				ctx,
				deploymentKey,
				current,
			)

			return errors.IsNotFound(err)
		}, "5s", "200ms").Should(BeTrue())

		By("reconciling the ModelService again")

		_, err = controllerReconciler.Reconcile(
			ctx,
			request,
		)
		Expect(err).NotTo(HaveOccurred())

		rebuiltDeployment := &appsv1.Deployment{}

		Eventually(func() error {
			return k8sClient.Get(
				ctx,
				deploymentKey,
				rebuiltDeployment,
			)
		}, "5s", "200ms").Should(Succeed())

		Expect(rebuiltDeployment.UID).
			NotTo(Equal(firstUID))

		Expect(
			rebuiltDeployment.Spec.Template.Spec.Containers,
		).To(HaveLen(1))

		Expect(
			rebuiltDeployment.Spec.Template.Spec.Containers[0].Image,
		).To(Equal("example.invalid/runtime:v1"))
	})
})
