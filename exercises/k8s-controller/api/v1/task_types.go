/*
Copyright 2025.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TaskSpec defines the desired state of Task.
type TaskSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Command is the command to be executed
	// +kubebuilder:validation:Required
	Command string `json:"command"`

	// Args are the arguments to pass to the command
	// +optional
	Args []string `json:"args,omitempty"`

	// Schedule is a cron expression for recurring tasks
	// +optional
	// +kubebuilder:validation:Pattern=`^(@(annually|yearly|monthly|weekly|daily|hourly|reboot))|(@every (\d+(ns|us|µs|ms|s|m|h))+)|((((\d+,)+\d+|(\d+(\/|-|\#)\d+)|\d+L?|\*(\/\d+)?|L(-\d+)?|\?|[A-Z]{3}(-\d{4})?) ?){5,7})$`
	Schedule string `json:"schedule,omitempty"`
}

// TaskStatus defines the observed state of Task.
type TaskStatus struct {
	// LastExecutionTime is the last time the command was executed
	// +optional
	LastExecutionTime *metav1.Time `json:"lastExecutionTime,omitempty"`

	// LastExecutionOutput contains the output of the last command execution
	// +optional
	LastExecutionOutput string `json:"lastExecutionOutput,omitempty"`

	// LastError contains the error message if the last execution failed
	// +optional
	LastError string `json:"lastError,omitempty"`

	// ExecutionCount is the number of times the command has been executed
	// +optional
	ExecutionCount int32 `json:"executionCount,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Command",type="string",JSONPath=".spec.command"
// +kubebuilder:printcolumn:name="Schedule",type="string",JSONPath=".spec.schedule"
// +kubebuilder:printcolumn:name="LastExecuted",type="date",JSONPath=".status.lastExecutionTime"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.lastError"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Task is the Schema for the tasks API.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Task struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TaskSpec   `json:"spec,omitempty"`
	Status TaskStatus `json:"status,omitempty"`
}

// TaskList contains a list of Task
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type TaskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Task `json:"items"`
}

// +kubebuilder:object:root=true

func init() {
	SchemeBuilder.Register(&Task{}, &TaskList{})
}
