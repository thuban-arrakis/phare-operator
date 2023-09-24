/*
Copyright 2023.

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

package v1beta1

import (
  corev1 "k8s.io/api/core/v1"
  metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PhareSpec defines the desired state of Phare.
type PhareSpec struct {
  Microservice MicroserviceSpec  `json:"microservice"`
  Service      ServiceSpec       `json:"service,omitempty"`
  Config       map[string]string `json:"config,omitempty"`
}

// MicroserviceSpec contains the specifications related to the microservice.
type MicroserviceSpec struct {
  Name            string    `json:"name"`
  Namespace       string    `json:"namespace"`
  Kind            string    `json:"kind"`
  ReplicaCount    int32     `json:"replicaCount"`
  Image           ImageSpec `json:"image"`
  ImagePullPolicy string    `json:"imagePullPolicy"`
}

// ImageSpec holds information about the microservice's container image.
type ImageSpec struct {
  Repository string `json:"repository"`
  Tag        string `json:"tag"`
}

// PharePhase represents the phases of Phare processing.
type PharePhase string

// These are valid phases of Phare.
const (
  // PharePhaseReconciling means the Phare is being reconciled.
  PharePhaseReconciling PharePhase = "Reconciling"

  // PharePhaseActive means the Phare is active and running.
  PharePhaseActive PharePhase = "Active"

  // PharePhaseFailed means the Phare failed to reconcile correctly.
  PharePhaseFailed PharePhase = "Failed"
)

// PhareStatus defines the observed state of Phare.
type PhareStatus struct {
  // INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
  // Important: Run "make" to regenerate code after modifying this file
  // Phase represents the current phase of Phare processing.
  Phase PharePhase `json:"phase,omitempty"`

  // Message provides additional information about the current phase.
  Message string `json:"message,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Phare is the Schema for the phares API.
type Phare struct {
  metav1.TypeMeta   `json:",inline"`
  metav1.ObjectMeta `json:"metadata,omitempty"`

  Spec   PhareSpec   `json:"spec,omitempty"`
  Status PhareStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PhareList contains a list of Phare.
type PhareList struct {
  metav1.TypeMeta `json:",inline"`
  metav1.ListMeta `json:"metadata,omitempty"`
  Items           []Phare `json:"items"`
}

type ServiceSpec struct {
  Type corev1.ServiceType `json:"type,omitempty"`

  Ports []corev1.ServicePort `json:"ports,omitempty"`

  Annotations map[string]string `json:"annotations,omitempty"`

  Labels map[string]string `json:"labels,omitempty"`
}

func init() {
  SchemeBuilder.Register(&Phare{}, &PhareList{})
}
