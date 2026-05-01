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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RuleType identifies a detection rule.
// +kubebuilder:validation:Enum=CrashLoopDetection;OOMKillDetection;PendingPodDetection
type RuleType string

const (
	RuleCrashLoop RuleType = "CrashLoopDetection"
	RuleOOMKill   RuleType = "OOMKillDetection"
	RulePending   RuleType = "PendingPodDetection"
)

type Finding struct {
	PodRef            string      `json:"podRef"` // namespace/podname
	FirstObservedTime metav1.Time `json:"firstObservedTime"`
	LastObservedTime  metav1.Time `json:"lastObservedTime"`
	RuleType          RuleType    `json:"ruleType"`
	Message           string      `json:"message"`
}

// HealthPolicySpec defines the desired state of HealthPolicy
type HealthPolicySpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html

	Namespaces []string `json:"namespaces,omitempty"`

	// Max restarts before it comes up as a finding
	// +kubebuilder:default=3
	// +kubebuilder:validation:Minimum=1
	CrashLoopThreshold int `json:"crashLoopThreshold,omitempty"`

	// How often to wait in the queue before the next check
	// +kubebuilder:default="30s"
	ReportingInterval metav1.Duration `json:"reportingInterval,omitempty"`

	// Max time for a pod to be in Pending state before it becomes a finding
	// +kubebuilder:default="5m"
	PendingPodThreshold metav1.Duration `json:"pendingPodThreshold,omitempty"`
}

// HealthPolicyStatus defines the observed state of HealthPolicy.
type HealthPolicyStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// conditions represent the current state of the HealthPolicy resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// A list of findings keyed on podRef and ruleType
	// +listType=map
	// +listMapKey=podRef
	// +listMapKey=ruleType
	Findings []Finding `json:"findings,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// HealthPolicy is the Schema for the healthpolicies API
type HealthPolicy struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of HealthPolicy
	// +required
	Spec HealthPolicySpec `json:"spec"`

	// status defines the observed state of HealthPolicy
	// +optional
	Status HealthPolicyStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// HealthPolicyList contains a list of HealthPolicy
type HealthPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []HealthPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HealthPolicy{}, &HealthPolicyList{})
}
