package controller

import (
	"testing"
	"time"

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
func TestEvaluatePending(t *testing.T) {
	created := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name      string
		phase     corev1.PodPhase
		age       time.Duration
		threshold time.Duration
		wantOK    bool
	}{
		{
			name:      "pending past threshold",
			phase:     corev1.PodPending,
			age:       7 * time.Minute,
			threshold: 5 * time.Minute,
			wantOK:    true,
		},
		{
			name:      "pending below threshold",
			phase:     corev1.PodPending,
			age:       2 * time.Minute,
			threshold: 5 * time.Minute,
			wantOK:    false,
		},
		{
			name:      "pending at threshold",
			phase:     corev1.PodPending,
			age:       5 * time.Minute,
			threshold: 5 * time.Minute,
			wantOK:    true,
		},
		{
			name:      "running",
			phase:     corev1.PodRunning,
			age:       7 * time.Minute,
			threshold: 5 * time.Minute,
			wantOK:    false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pod := corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "demo",
					Namespace:         "default",
					CreationTimestamp: metav1.Time{Time: created},
				},
				Status: corev1.PodStatus{Phase: tc.phase},
			}
			now := created.Add(tc.age)

			got, ok := evaluatePending(pod, tc.threshold, now)
			if ok != tc.wantOK {
				t.Errorf("ok = %v, want %v", ok, tc.wantOK)
			}
			if ok {
				if got.RuleType != monitoringv1alpha1.RulePending {
					t.Errorf("RuleType = %q, want %q", got.RuleType, monitoringv1alpha1.RulePending)
				}
				if got.PodRef != "default/demo" {
					t.Errorf("PodRef = %q, want %q", got.PodRef, "default/demo")
				}
			}
		})
	}

}
