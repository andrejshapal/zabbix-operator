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
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	zabbixv1 "github.com/andrejshapal/zabbix-operator/api/v1"
)

var _ = Describe("ZabbixInstance Controller", func() {
	Context("When reconciling a resource", func() {
		const (
			resourceName      = "test-resource"
			resourceNamespace = "default"
		)

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: resourceNamespace,
		}
		zabbixinstance := &zabbixv1.ZabbixInstance{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind ZabbixInstance")
			err := k8sClient.Get(ctx, typeNamespacedName, zabbixinstance)
			if err != nil && errors.IsNotFound(err) {
				resource := &zabbixv1.ZabbixInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: resourceNamespace,
					},
					// TODO(user): Specify other spec details if needed.
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &zabbixv1.ZabbixInstance{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance ZabbixInstance")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &ZabbixInstanceReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			expectedImages := map[string]string{
				"server": "docker.io/zabbix/server-mysql:alpine-latest",
				"web":    "docker.io/zabbix/web-nginx-mysql:alpine-latest",
				"agent":  "docker.io/zabbix/agent2:alpine-latest",
			}
			for deploymentType, expectedImage := range expectedImages {
				deployment := &appsv1.Deployment{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name:      resourceName + "-" + deploymentType,
					Namespace: resourceNamespace,
				}, deployment)).To(Succeed())
				Expect(deployment.Spec.Template.Spec.Containers[0].Image).To(Equal(expectedImage))
				Expect(deployment.OwnerReferences).To(HaveLen(1))
				Expect(deployment.OwnerReferences[0].Controller).NotTo(BeNil())
				Expect(*deployment.OwnerReferences[0].Controller).To(BeTrue())
			}
		})
	})
})
