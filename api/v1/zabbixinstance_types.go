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

package v1

import (
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ZabbixInstanceSpec defines the desired state of ZabbixInstance
type ZabbixInstanceSpec struct {
	// +kubebuilder:default:={"imageRegistry":"docker.io","imageRepository":"zabbix","zabbixImageTag":"alpine-latest"}
	ZabbixGlobal *ZabbixGlobal `json:"global,omitempty"`
	// +kubebuilder:default:={"image":{"name":"zabbix-server-mysql"}}
	ZabbixServer *ZabbixServer `json:"server,omitempty"`
	// +kubebuilder:default:={"image":{"name":"zabbix-web-nginx-mysql"}}
	ZabbixWebServer *ZabbixWebServer `json:"webServer,omitempty"`
	// +kubebuilder:default:={"image":{"name":"zabbix-agent2"}}
	ZabbixAgent *ZabbixAgent `json:"agent,omitempty"`
}

type ZabbixGlobal struct {
	CommonLabels map[string]string `json:"commonLabels,omitempty"`
	// +kubebuilder:default:="alpine-latest"
	// +kubebuilder:validation:MinLength=1
	ZabbixImageTag string `json:"zabbixImageTag,omitempty"`
	// +kubebuilder:default:="docker.io"
	// +kubebuilder:validation:MinLength=1
	ImageRegistry string `json:"imageRegistry,omitempty"`
	// +kubebuilder:default:="zabbix"
	// +kubebuilder:validation:MinLength=1
	ImageRepository string `json:"imageRepository,omitempty"`
}

type ZabbixServer struct {
	ZabbixBase `json:",inline"`
}

type ZabbixWebServer struct {
	ZabbixBase    `json:",inline"`
	ZabbixIngress *ZabbixIngress `json:"ingress,omitempty"`
}

type ZabbixAgent struct {
	ZabbixBase `json:",inline"`
}

type ZabbixBase struct {
	// +kubebuilder:default:={}
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
	// +kubebuilder:default:={}
	ZabbixImage *ZabbixImage `json:"image,omitempty"`

	Service *ZabbixService `json:"service,omitempty"`

	ExtraEnv            []corev1.EnvVar      `json:"extraEnv,omitempty"`
	ExtraVolumeMounts   []corev1.VolumeMount `json:"extraVolumeMounts,omitempty"`
	ExtraContainers     []corev1.Container   `json:"extraContainers,omitempty"`
	ExtraInitContainers []corev1.Container   `json:"extraInitContainers,omitempty"`
	ExtraVolumes        []corev1.Volume      `json:"extraVolumes,omitempty"`

	PodSecurityContext *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`

	LivenessProbe  *corev1.Probe `json:"livenessProbe,omitempty"`
	ReadinessProbe *corev1.Probe `json:"readinessProbe,omitempty"`
	StartupProbe   *corev1.Probe `json:"startupProbe,omitempty"`

	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
}

type ZabbixImage struct {
	Registry   string `json:"registry,omitempty"`
	Repository string `json:"repository,omitempty"`
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name,omitempty"`
	Tag  string `json:"tag,omitempty"`
	// +kubebuilder:default:="IfNotPresent"
	PullPolicy  corev1.PullPolicy             `json:"pullPolicy,omitempty"`
	PullSecrets []corev1.LocalObjectReference `json:"pullSecrets,omitempty"`
}

type ZabbixService struct {
	Type                     corev1.ServiceType `json:"type,omitempty"`
	ClusterIP                string             `json:"clusterIP,omitempty"`
	ExternalIPs              []string           `json:"externalIPs,omitempty"`
	LoadBalancerIP           string             `json:"loadBalancerIP,omitempty"`
	LoadBalancerSourceRanges []string           `json:"loadBalancerSourceRanges,omitempty"`
	LoadBalancerClass        *string            `json:"loadBalancerClass,omitempty"`
	Port                     int32              `json:"port,omitempty"`
	NodePort                 int32              `json:"nodePort,omitempty"`
	Annotations              map[string]string  `json:"annotations,omitempty"`
}

type ZabbixIngress struct {
	Enabled          bool                      `json:"enabled,omitempty"`
	Annotations      map[string]string         `json:"annotations,omitempty"`
	Hosts            []ZabbixIngressHost       `json:"hosts,omitempty"`
	TLS              []networkingv1.IngressTLS `json:"tls,omitempty"`
	IngressClassName *string                   `json:"ingressClassName,omitempty"`
}

type ZabbixIngressHost struct {
	Host  string                         `json:"host,omitempty"`
	Paths []networkingv1.HTTPIngressPath `json:"paths,omitempty"`
}

// ZabbixInstanceStatus defines the observed state of ZabbixInstance.
type ZabbixInstanceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// conditions represent the current state of the ZabbixInstance resource.
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
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced

// ZabbixInstance is the Schema for the zabbixinstances API
type ZabbixInstance struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// spec defines the desired state of ZabbixInstance
	// +required
	Spec ZabbixInstanceSpec `json:"spec"`

	// status defines the observed state of ZabbixInstance
	// +optional
	Status ZabbixInstanceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ZabbixInstanceList contains a list of ZabbixInstance
type ZabbixInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ZabbixInstance `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &ZabbixInstance{}, &ZabbixInstanceList{})
		return nil
	})
}
