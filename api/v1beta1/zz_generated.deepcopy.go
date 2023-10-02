//go:build !ignore_autogenerated
// +build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1beta1

import (
	"k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	apisv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ConfigSpec) DeepCopyInto(out *ConfigSpec) {
	*out = *in
	if in.Data != nil {
		in, out := &in.Data, &out.Data
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConfigSpec.
func (in *ConfigSpec) DeepCopy() *ConfigSpec {
	if in == nil {
		return nil
	}
	out := new(ConfigSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DefaultCheck) DeepCopyInto(out *DefaultCheck) {
	*out = *in
	out.LogConfig = in.LogConfig
	out.Config = in.Config
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DefaultCheck.
func (in *DefaultCheck) DeepCopy() *DefaultCheck {
	if in == nil {
		return nil
	}
	out := new(DefaultCheck)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GCPBackendPolicyDefaultSpec) DeepCopyInto(out *GCPBackendPolicyDefaultSpec) {
	*out = *in
	out.Logging = in.Logging
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GCPBackendPolicyDefaultSpec.
func (in *GCPBackendPolicyDefaultSpec) DeepCopy() *GCPBackendPolicyDefaultSpec {
	if in == nil {
		return nil
	}
	out := new(GCPBackendPolicyDefaultSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GCPBackendPolicyLoggingSpec) DeepCopyInto(out *GCPBackendPolicyLoggingSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GCPBackendPolicyLoggingSpec.
func (in *GCPBackendPolicyLoggingSpec) DeepCopy() *GCPBackendPolicyLoggingSpec {
	if in == nil {
		return nil
	}
	out := new(GCPBackendPolicyLoggingSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GCPBackendPolicySpec) DeepCopyInto(out *GCPBackendPolicySpec) {
	*out = *in
	out.Default = in.Default
	out.TargetRef = in.TargetRef
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GCPBackendPolicySpec.
func (in *GCPBackendPolicySpec) DeepCopy() *GCPBackendPolicySpec {
	if in == nil {
		return nil
	}
	out := new(GCPBackendPolicySpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GCPBackendPolicyTargetRefSpec) DeepCopyInto(out *GCPBackendPolicyTargetRefSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GCPBackendPolicyTargetRefSpec.
func (in *GCPBackendPolicyTargetRefSpec) DeepCopy() *GCPBackendPolicyTargetRefSpec {
	if in == nil {
		return nil
	}
	out := new(GCPBackendPolicyTargetRefSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GRPCCheck) DeepCopyInto(out *GRPCCheck) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GRPCCheck.
func (in *GRPCCheck) DeepCopy() *GRPCCheck {
	if in == nil {
		return nil
	}
	out := new(GRPCCheck)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HTTPRouteSpec) DeepCopyInto(out *HTTPRouteSpec) {
	*out = *in
	if in.Hostnames != nil {
		in, out := &in.Hostnames, &out.Hostnames
		*out = make([]apisv1beta1.Hostname, len(*in))
		copy(*out, *in)
	}
	if in.ParentRef != nil {
		in, out := &in.ParentRef, &out.ParentRef
		*out = make([]apisv1beta1.ParentReference, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Rules != nil {
		in, out := &in.Rules, &out.Rules
		*out = make([]apisv1beta1.HTTPRouteRule, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HTTPRouteSpec.
func (in *HTTPRouteSpec) DeepCopy() *HTTPRouteSpec {
	if in == nil {
		return nil
	}
	out := new(HTTPRouteSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HealthCheck) DeepCopyInto(out *HealthCheck) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HealthCheck.
func (in *HealthCheck) DeepCopy() *HealthCheck {
	if in == nil {
		return nil
	}
	out := new(HealthCheck)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HealthCheckConfig) DeepCopyInto(out *HealthCheckConfig) {
	*out = *in
	out.HTTPHealthCheck = in.HTTPHealthCheck
	out.HTTPSHealthCheck = in.HTTPSHealthCheck
	out.GRPCCheck = in.GRPCCheck
	out.HTTP2Check = in.HTTP2Check
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HealthCheckConfig.
func (in *HealthCheckConfig) DeepCopy() *HealthCheckConfig {
	if in == nil {
		return nil
	}
	out := new(HealthCheckConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HealthCheckPolicySpec) DeepCopyInto(out *HealthCheckPolicySpec) {
	*out = *in
	out.Default = in.Default
	out.TargetRef = in.TargetRef
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HealthCheckPolicySpec.
func (in *HealthCheckPolicySpec) DeepCopy() *HealthCheckPolicySpec {
	if in == nil {
		return nil
	}
	out := new(HealthCheckPolicySpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ImageSpec) DeepCopyInto(out *ImageSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ImageSpec.
func (in *ImageSpec) DeepCopy() *ImageSpec {
	if in == nil {
		return nil
	}
	out := new(ImageSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LogConfig) DeepCopyInto(out *LogConfig) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LogConfig.
func (in *LogConfig) DeepCopy() *LogConfig {
	if in == nil {
		return nil
	}
	out := new(LogConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MicroServiceSpec) DeepCopyInto(out *MicroServiceSpec) {
	*out = *in
	out.Image = in.Image
	if in.Env != nil {
		in, out := &in.Env, &out.Env
		*out = make([]v1.EnvVar, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Affinity != nil {
		in, out := &in.Affinity, &out.Affinity
		*out = new(v1.Affinity)
		(*in).DeepCopyInto(*out)
	}
	if in.Tolerations != nil {
		in, out := &in.Tolerations, &out.Tolerations
		*out = make([]v1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Volumes != nil {
		in, out := &in.Volumes, &out.Volumes
		*out = make([]v1.Volume, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Sidecars != nil {
		in, out := &in.Sidecars, &out.Sidecars
		*out = make([]Sidecar, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.InitContainers != nil {
		in, out := &in.InitContainers, &out.InitContainers
		*out = make([]v1.Container, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	in.deleteOptions.DeepCopyInto(&out.deleteOptions)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MicroServiceSpec.
func (in *MicroServiceSpec) DeepCopy() *MicroServiceSpec {
	if in == nil {
		return nil
	}
	out := new(MicroServiceSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Phare) DeepCopyInto(out *Phare) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Phare.
func (in *Phare) DeepCopy() *Phare {
	if in == nil {
		return nil
	}
	out := new(Phare)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Phare) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PhareList) DeepCopyInto(out *PhareList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Phare, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PhareList.
func (in *PhareList) DeepCopy() *PhareList {
	if in == nil {
		return nil
	}
	out := new(PhareList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *PhareList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PhareSpec) DeepCopyInto(out *PhareSpec) {
	*out = *in
	in.MicroService.DeepCopyInto(&out.MicroService)
	if in.Service != nil {
		in, out := &in.Service, &out.Service
		*out = new(ServiceSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.Config != nil {
		in, out := &in.Config, &out.Config
		*out = new(ConfigSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.ToolChain != nil {
		in, out := &in.ToolChain, &out.ToolChain
		*out = new(ToolChainSpec)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PhareSpec.
func (in *PhareSpec) DeepCopy() *PhareSpec {
	if in == nil {
		return nil
	}
	out := new(PhareSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PhareStatus) DeepCopyInto(out *PhareStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PhareStatus.
func (in *PhareStatus) DeepCopy() *PhareStatus {
	if in == nil {
		return nil
	}
	out := new(PhareStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResourceDescription) DeepCopyInto(out *ResourceDescription) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResourceDescription.
func (in *ResourceDescription) DeepCopy() *ResourceDescription {
	if in == nil {
		return nil
	}
	out := new(ResourceDescription)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Resources) DeepCopyInto(out *Resources) {
	*out = *in
	out.ResourceRequests = in.ResourceRequests
	out.ResourceLimits = in.ResourceLimits
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Resources.
func (in *Resources) DeepCopy() *Resources {
	if in == nil {
		return nil
	}
	out := new(Resources)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ServiceSpec) DeepCopyInto(out *ServiceSpec) {
	*out = *in
	if in.Ports != nil {
		in, out := &in.Ports, &out.Ports
		*out = make([]v1.ServicePort, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Annotations != nil {
		in, out := &in.Annotations, &out.Annotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ServiceSpec.
func (in *ServiceSpec) DeepCopy() *ServiceSpec {
	if in == nil {
		return nil
	}
	out := new(ServiceSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Sidecar) DeepCopyInto(out *Sidecar) {
	*out = *in
	if in.Resources != nil {
		in, out := &in.Resources, &out.Resources
		*out = new(Resources)
		**out = **in
	}
	if in.Ports != nil {
		in, out := &in.Ports, &out.Ports
		*out = make([]v1.ContainerPort, len(*in))
		copy(*out, *in)
	}
	if in.Env != nil {
		in, out := &in.Env, &out.Env
		*out = make([]v1.EnvVar, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Sidecar.
func (in *Sidecar) DeepCopy() *Sidecar {
	if in == nil {
		return nil
	}
	out := new(Sidecar)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TargetRef) DeepCopyInto(out *TargetRef) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TargetRef.
func (in *TargetRef) DeepCopy() *TargetRef {
	if in == nil {
		return nil
	}
	out := new(TargetRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ToolChainSpec) DeepCopyInto(out *ToolChainSpec) {
	*out = *in
	if in.HTTPRoute != nil {
		in, out := &in.HTTPRoute, &out.HTTPRoute
		*out = new(HTTPRouteSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.HealthCheckPolicy != nil {
		in, out := &in.HealthCheckPolicy, &out.HealthCheckPolicy
		*out = new(HealthCheckPolicySpec)
		**out = **in
	}
	if in.GCPBackendPolicy != nil {
		in, out := &in.GCPBackendPolicy, &out.GCPBackendPolicy
		*out = new(GCPBackendPolicySpec)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ToolChainSpec.
func (in *ToolChainSpec) DeepCopy() *ToolChainSpec {
	if in == nil {
		return nil
	}
	out := new(ToolChainSpec)
	in.DeepCopyInto(out)
	return out
}
