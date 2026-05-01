package controller

import (
	"fmt"
	"time"

	monitoringv1alpha1 "github.com/spackle0/k8s-health-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

// Per-rule decision-making

// CrashLoop
func evaluateCrashLoop(pod corev1.Pod, cs corev1.ContainerStatus, threshold int) (monitoringv1alpha1.Finding, bool) {
	if int(cs.RestartCount) < threshold {
		return monitoringv1alpha1.Finding{}, false
	}
	return monitoringv1alpha1.Finding{
		PodRef:   fmt.Sprintf("%s/%s", pod.Namespace, pod.Name),
		RuleType: monitoringv1alpha1.RuleCrashLoop,
		Message:  fmt.Sprintf("Container %s in pod %s restarted %d times", cs.Name, pod.Name, cs.RestartCount),
	}, true
}

// OOMKill
func evaluateOOMKill(pod corev1.Pod, cs corev1.ContainerStatus) (monitoringv1alpha1.Finding, bool) {
	term := cs.LastTerminationState.Terminated
	if term == nil || term.Reason != "OOMKilled" {
		return monitoringv1alpha1.Finding{}, false
	}
	return monitoringv1alpha1.Finding{
		PodRef:   fmt.Sprintf("%s/%s", pod.Namespace, pod.Name),
		RuleType: monitoringv1alpha1.RuleOOMKill,
		Message:  fmt.Sprintf("Container %s in pod %s was OOMKilled", cs.Name, pod.Name),
	}, true
}

// Pending
func evaluatePending(pod corev1.Pod, threshold time.Duration, now time.Time) (monitoringv1alpha1.Finding, bool) {
	if pod.Status.Phase != corev1.PodPending {
		return monitoringv1alpha1.Finding{}, false
	}
	age := now.Sub(pod.CreationTimestamp.Time)
	if age < threshold {
		return monitoringv1alpha1.Finding{}, false
	}
	return monitoringv1alpha1.Finding{
		PodRef:   fmt.Sprintf("%s/%s", pod.Namespace, pod.Name),
		RuleType: monitoringv1alpha1.RulePending,
		Message:  fmt.Sprintf("Pod %s has been pending for %s", pod.Name, age.Round(time.Second)),
	}, true
}
