package controller

import (
	"testing"

	monitoringv1alpha1 "github.com/spackle0/k8s-health-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestEvaluateCrashLoop(t *testing.T) {
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
	}

	cases := []struct {
		name         string
		restartCount int32
		threshold    int
		wantOK       bool
	}{
		{name: "below threshold", restartCount: 2, threshold: 3, wantOK: false},
		{name: "at threshold", restartCount: 3, threshold: 3, wantOK: true},
		{name: "above threshold", restartCount: 5, threshold: 3, wantOK: true},
		{name: "zero restarts", restartCount: 0, threshold: 3, wantOK: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cs := corev1.ContainerStatus{Name: "app", RestartCount: tc.restartCount}
			got, ok := evaluateCrashLoop(pod, cs, tc.threshold)
			if ok != tc.wantOK {
				t.Errorf("ok = %v, want %v", ok, tc.wantOK)
			}
			if ok {
				if got.RuleType != monitoringv1alpha1.RuleCrashLoop {
					t.Errorf("RuleType = %q, want %q", got.RuleType, monitoringv1alpha1.RuleCrashLoop)
				}
				if got.PodRef != "default/demo" {
					t.Errorf("PodRef = %q, want %q", got.PodRef, "default/demo")
				}
			}
		})
	}
}

func TestEvaluateOOMKill(t *testing.T) {
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
	}

	cases := []struct {
		name   string
		state  corev1.ContainerState
		wantOK bool
	}{
		{
			name:   "no prior termination",
			state:  corev1.ContainerState{Terminated: nil},
			wantOK: false,
		},
		{
			name: "terminated with OOMKilled",
			state: corev1.ContainerState{
				Terminated: &corev1.ContainerStateTerminated{Reason: "OOMKilled"},
			},
			wantOK: true,
		},
		{
			name: "terminated with other reason",
			state: corev1.ContainerState{
				Terminated: &corev1.ContainerStateTerminated{Reason: "Error"},
			},
			wantOK: false,
		},
		{
			name: "terminated with empty reason",
			state: corev1.ContainerState{
				Terminated: &corev1.ContainerStateTerminated{Reason: ""},
			},
			wantOK: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cs := corev1.ContainerStatus{Name: "app", LastTerminationState: tc.state}
			got, ok := evaluateOOMKill(pod, cs)
			if ok != tc.wantOK {
				t.Errorf("ok = %v, want %v", ok, tc.wantOK)
			}
			if ok {
				if got.RuleType != monitoringv1alpha1.RuleOOMKill {
					t.Errorf("RuleType = %q, want %q", got.RuleType, monitoringv1alpha1.RuleOOMKill)
				}
				if got.PodRef != "default/demo" {
					t.Errorf("PodRef = %q, want %q", got.PodRef, "default/demo")
				}
			}
		})
	}
}
