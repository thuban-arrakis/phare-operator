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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

// PhareSpec defines the desired state of Phare.
type PhareSpec struct {
	MicroService MicroServiceSpec `json:"microservice"`
	Service      *v1.ServiceSpec  `json:"service,omitempty"`
	ToolChain    *ToolChainSpec   `json:"toolchain,omitempty"`
}

// MicroserviceSpec contains the specifications related to the microservice.
type MicroServiceSpec struct {
	// Provides deterministic kind of the microservice.
	// +kubebuilder:validation:Enum=Deployment;StatefulSet
	Kind                 string                     `json:"kind"`
	ReplicaCount         int32                      `json:"replicaCount,omitempty"`
	Image                ImageSpec                  `json:"image"`
	Ports                []v1.ContainerPort         `json:"ports,omitempty"`
	ImagePullPolicy      v1.PullPolicy              `json:"imagePullPolicy,omitempty"`
	Env                  []v1.EnvVar                `json:"env,omitempty"`
	EnvFrom              []v1.EnvFromSource         `json:"envFrom,omitempty"`
	Affinity             *v1.Affinity               `json:"affinity,omitempty"`
	Tolerations          []v1.Toleration            `json:"tolerations,omitempty"`
	Volumes              []v1.Volume                `json:"volumes,omitempty"`
	VolumeMounts         []v1.VolumeMount           `json:"volumeMounts,omitempty"`
	VolumeClaimTemplates []v1.PersistentVolumeClaim `json:"volumeClaimTemplates,omitempty"`
	InitContainers       []v1.Container             `json:"initContainers,omitempty"`
	ExtraContainers      []v1.Container             `json:"extraContainers,omitempty"`
	ResourceRequirements v1.ResourceRequirements    `json:"resourceRequirements,omitempty"` // TODO: Rename to Resources.
	Command              []string                   `json:"command,omitempty"`
	Args                 []string                   `json:"args,omitempty"`
	PodLabels            map[string]string          `json:"podLabels,omitempty"`
	PodAnnotations       map[string]string          `json:"podAnnotations,omitempty"`
	LivenessProbe        *v1.Probe                  `json:"livenessProbe,omitempty"`
	ReadinessProbe       *v1.Probe                  `json:"readinessProbe,omitempty"`
	StartupProbe         *v1.Probe                  `json:"startupProbe,omitempty"`
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

func init() {
	SchemeBuilder.Register(&Phare{}, &PhareList{})
}

type ToolChainSpec struct {
	Config            ConfigSpec             `json:"config,omitempty"`
	HTTPRoute         *HTTPRouteSpec         `json:"httpRoute,omitempty"`
	HealthCheckPolicy *HealthCheckPolicySpec `json:"healthCheckPolicy,omitempty"`
	GCPBackendPolicy  *GCPBackendPolicySpec  `json:"gcpBackendPolicy,omitempty"`
}

type ConfigSpec map[string]string

type HTTPRouteSpec struct {
	Hostnames []gatewayv1beta1.Hostname        `json:"hostnames,omitempty"`
	ParentRef []gatewayv1beta1.ParentReference `json:"parentRefs,omitempty"` // Ensure this is named correctly
	// +kubebuilder:validation:MaxItems=10
	Rules []gatewayv1beta1.HTTPRouteRule `json:"rules,omitempty"`
}
type HealthCheckPolicySpec struct {
	Default   DefaultCheck `json:"default"`
	TargetRef TargetRef    `json:"targetRef"`
}

type DefaultCheck struct {
	CheckIntervalSec   string            `json:"checkIntervalSec"`
	TimeoutSec         string            `json:"timeoutSec"`
	HealthyThreshold   string            `json:"healthyThreshold"`
	UnhealthyThreshold string            `json:"unhealthyThreshold"`
	LogConfig          LogConfig         `json:"logConfig"`
	Config             HealthCheckConfig `json:"config"`
}

type LogConfig struct {
	Enabled string `json:"enabled"`
}

type HealthCheckConfig struct {
	Type             string      `json:"type"`
	HTTPHealthCheck  HealthCheck `json:"httpHealthCheck"`
	HTTPSHealthCheck HealthCheck `json:"httpsHealthCheck"`
	GRPCCheck        GRPCCheck   `json:"grpcHealthCheck"`
	HTTP2Check       HealthCheck `json:"http2HealthCheck"`
}

type HealthCheck struct {
	PortSpecification string `json:"portSpecification"`
	Port              string `json:"port"`
	PortName          string `json:"portName"`
	Host              string `json:"host"`
	RequestPath       string `json:"requestPath"`
	Response          string `json:"response"`
	ProxyHeader       string `json:"proxyHeader"`
}

type GRPCCheck struct {
	GRPCServiceName   string `json:"grpcServiceName"`
	PortSpecification string `json:"portSpecification"`
	Port              string `json:"port"`
	PortName          string `json:"portName"`
}

type TargetRef struct {
	Group string `json:"group"`
	Kind  string `json:"kind"`
	Name  string `json:"name"`
}

type GCPBackendPolicySpec struct {
	Default   GCPBackendPolicyDefaultSpec   `json:"default,omitempty"`
	TargetRef GCPBackendPolicyTargetRefSpec `json:"targetRef,omitempty"`
}

type GCPBackendPolicyDefaultSpec struct {
	Logging    GCPBackendPolicyLoggingSpec `json:"logging,omitempty"`
	TimeoutSec int                         `json:"timeoutSec,omitempty"`
}

type GCPBackendPolicyLoggingSpec struct {
	Enabled    bool `json:"enabled"`
	SampleRate int  `json:"sampleRate"`
}

type GCPBackendPolicyTargetRefSpec struct {
	Group string `json:"group"`
	Kind  string `json:"kind"`
	Name  string `json:"name"`
}
