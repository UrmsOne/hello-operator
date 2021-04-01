/*


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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ComponentType contains the different types of components of the service
type ComponentType string

const (
	DeploymentComponent ComponentType = "deployment"
	ServiceComponent    ComponentType = "service"
)

// ComponentStatusSpec describes the state of the component
type ComponentStatusSpec struct {
	LatestReadyTime string `json:"latest_ready_time,omitempty"`
}

// HelloServiceSpec defines the desired state of HelloService
type HelloServiceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Image is an example field of HelloService. Edit HelloService_types.go to remove/update
	Image    string `json:"image,omitempty"`
	Port     int32  `json:"port,omitempty"`
	Replicas int32  `json:"replicas,omitempty"`
}

// HelloServiceStatus defines the observed state of HelloService
type HelloServiceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	//Conditions []metav1.ConditionStatus
	Components map[ComponentType]ComponentStatusSpec `json:"components,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// HelloService is the Schema for the helloservices API
type HelloService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HelloServiceSpec   `json:"spec,omitempty"`
	Status HelloServiceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// HelloServiceList contains a list of HelloService
type HelloServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HelloService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HelloService{}, &HelloServiceList{})
}
