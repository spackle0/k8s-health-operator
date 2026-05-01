package controller

import (
	"fmt"

	monitoringv1alpha1 "github.com/spackle0/k8s-health-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

// Per-rule decision-making

// Crash Loop
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

// OOM Kill
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
