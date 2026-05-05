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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	monitoringv1alpha1 "github.com/spackle0/k8s-health-operator/api/v1alpha1"
)

var _ = Describe("HealthPolicy Controller", func() {
	Context("When reconciling with a CrashLoopDetection rule", func() {
		const (
			policyName = "test-crashloop-policy"
			podName    = "test-crashy"
			namespace  = "default"
		)

		BeforeEach(func() {
			By("creating a HealthPolicy with a CrashLoopDetection rule")
			policy := &monitoringv1alpha1.HealthPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      policyName,
					Namespace: namespace,
				},
				Spec: monitoringv1alpha1.HealthPolicySpec{
					Namespaces:        []string{namespace},
					ReportingInterval: metav1.Duration{Duration: 30 * time.Second},
					Rules: []monitoringv1alpha1.RuleSpec{
						{
							Type:      monitoringv1alpha1.RuleCrashLoop,
							CrashLoop: &monitoringv1alpha1.CrashLoopConfig{Threshold: 3},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, policy)).To(Succeed())
		})

		AfterEach(func() {
			By("cleaning up policy and pod")
			policy := &monitoringv1alpha1.HealthPolicy{
				ObjectMeta: metav1.ObjectMeta{Name: policyName, Namespace: namespace},
			}
			_ = k8sClient.Delete(ctx, policy)

			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: podName, Namespace: namespace},
			}
			_ = k8sClient.Delete(ctx, pod)
		})

		It("should record a CrashLoopDetection finding for a pod above threshold", func() {
			// 2. Create a pod and bump its container's RestartCount above 3
			//    via the two-step Create + Status().Update pattern.

			// 3. Construct the reconciler and call Reconcile once with the
			//    NamespacedName of the policy.
			reconciler := &HealthPolicyReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			By("creating a pod with RestartCount above the threshold")
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      podName,
					Namespace: namespace,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "app", Image: "busybox"},
					},
				},
			}
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())

			// Status is a subresource - must be updated separately, after Create.
			pod.Status.ContainerStatuses = []corev1.ContainerStatus{
				{Name: "app", RestartCount: 7},
			}

			Expect(k8sClient.Status().Update(ctx, pod)).To(Succeed())
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: policyName, Namespace: namespace},
			})
			Expect(err).NotTo(HaveOccurred())

			// 4. Fetch the policy back and assert the finding.
			updated := &monitoringv1alpha1.HealthPolicy{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: policyName, Namespace: namespace}, updated)).To(Succeed())

			Expect(updated.Status.Findings).To(HaveLen(1))
			Expect(updated.Status.Findings[0].RuleType).To(Equal(monitoringv1alpha1.RuleCrashLoop))
			Expect(updated.Status.Findings[0].PodRef).To(Equal(namespace + "/" + podName))
		})

		It("should not record a finding for a pod below threshold", func() {
			By("creating a pod with RestartCount below the threshold")
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      podName,
					Namespace: namespace,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "app", Image: "busybox"},
					},
				},
			}
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())

			pod.Status.ContainerStatuses = []corev1.ContainerStatus{
				{Name: "app", RestartCount: 1},
			}
			Expect(k8sClient.Status().Update(ctx, pod)).To(Succeed())
			reconciler := &HealthPolicyReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: policyName, Namespace: namespace},
			})
			Expect(err).NotTo(HaveOccurred())

			updated := &monitoringv1alpha1.HealthPolicy{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: policyName, Namespace: namespace}, updated)).To(Succeed())

			Expect(updated.Status.Findings).To(BeEmpty())
		})
	})
})
