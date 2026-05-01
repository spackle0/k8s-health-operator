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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	monitoringv1alpha1 "github.com/spackle0/k8s-health-operator/api/v1alpha1"
)

// HealthPolicyReconciler reconciles a HealthPolicy object
type HealthPolicyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=monitoring.hugh.local,resources=healthpolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=monitoring.hugh.local,resources=healthpolicies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=monitoring.hugh.local,resources=healthpolicies/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the HealthPolicy object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.23.3/pkg/reconcile
func (r *HealthPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Reconcile flow:
	// 1. Read inputs
	//    - r.Get the policy             (already there)
	//    - r.List pods in each ns       (new: this is the chunk you're adding)
	//
	// 2. Compute / mutate in memory
	//    - Build findings from the pod list (later, not yet)
	//    - SetStatusCondition           (already there)
	//
	// 3. Persist the result
	//    - Status().Update              (already there, stays at the bottom)
	log := logf.FromContext(ctx)

	var policy monitoringv1alpha1.HealthPolicy

	log.Info("Reconciling HealthPolicy")
	if err := r.Get(ctx, req.NamespacedName, &policy); err != nil {
		// This is a briefer way to check if the resource is not found.
		// The IgnoreNotFound function will return nil if the error is not a NotFound error.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Info("Policy spec loaded",
		"namespaces", policy.Spec.Namespaces,
		"crashLoopThreshold", policy.Spec.CrashLoopThreshold,
	)

	// This is so that we can preserve the time of the first occurrence of a finding
	type findingKey struct {
		podRef   string
		ruleType monitoringv1alpha1.RuleType
	}

	// Let's populate the loop map of the findings from the existing one
	priorByKey := make(map[findingKey]monitoringv1alpha1.Finding, len(policy.Status.Findings))
	for _, f := range policy.Status.Findings {
		priorByKey[findingKey{podRef: f.PodRef, ruleType: f.RuleType}] = f
	}

	var findings []monitoringv1alpha1.Finding
	now := metav1.Now()

	// This is a closure that adds a finding
	addFinding := func(f monitoringv1alpha1.Finding) {
		f.LastObservedTime = now
		f.FirstObservedTime = now
		if prior, ok := priorByKey[findingKey{podRef: f.PodRef, ruleType: f.RuleType}]; ok {
			f.FirstObservedTime = prior.FirstObservedTime
		}
		findings = append(findings, f)
		log.Info("Finding recorded",
			"ruleType", f.RuleType,
			"podRef", f.PodRef,
			"message", f.Message,
		)
	}

	// Loop through chosen namespaces
	for _, ns := range policy.Spec.Namespaces {
		var podList corev1.PodList
		// client.InNamespace is not a function call returning data. It returns
		// a value of type client.ListOption (which is itself an interface).
		if err := r.List(ctx, &podList, client.InNamespace(ns)); err != nil {
			return ctrl.Result{}, err
		}
		log.Info("Pods listed", "podNamespace", ns, "count", len(podList.Items))

		// Check all the pods in the Namespace for different error conditions
		for _, pod := range podList.Items {

			// Per-pod evaluators
			if f, ok := evaluatePending(pod, policy.Spec.PendingPodThreshold.Duration, now.Time); ok {
				addFinding(f)
			}

			// Per container evaluators
			for _, cs := range pod.Status.ContainerStatuses {
				if f, ok := evaluateCrashLoop(pod, cs, policy.Spec.CrashLoopThreshold); ok {
					addFinding(f)
				}
				if f, ok := evaluateOOMKill(pod, cs); ok {
					addFinding(f)
				}
			}
		}
	}

	policy.Status.Findings = findings

	// SetStatusCondition mutates the conditions slice in memory; it does
	// not call the API server. The persisted change happens in
	// Status().Update below.
	meta.SetStatusCondition(&policy.Status.Conditions, metav1.Condition{
		Type:    "Available",
		Status:  metav1.ConditionTrue,
		Reason:  "Reconciled",
		Message: "Policy reconciled successfully",
	})

	// Use r.Status().Update (not r.Update) because spec and status are
	// separate subresources with different RBAC. The controller has
	// permission to write status; it intentionally does not touch spec.
	if err := r.Status().Update(ctx, &policy); err != nil {
		return ctrl.Result{}, err
	}

	// Don't add RequeueAfter to any of the error returns above. Here's why:
	//
	// - Returning a non-nil error already tells controller-runtime to requeue with
	// exponential backoff (starting at ~5ms, capping at ~16min). So errors get
	// retried automatically.
	// - The not-found return (client.IgnoreNotFound returns nil) means the policy
	// was deleted. There's nothing to requeue, there's no object to come back to.
	// - Adding RequeueAfter to error returns would conflict with the backoff logic.
	return ctrl.Result{RequeueAfter: policy.Spec.ReportingInterval.Duration}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *HealthPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&monitoringv1alpha1.HealthPolicy{}).
		Named("healthpolicy").
		Complete(r)
}
