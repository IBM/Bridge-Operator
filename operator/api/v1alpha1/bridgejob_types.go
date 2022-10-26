/*
Copyright 2022.

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
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BridgeJobSpec defines the desired state of BridgeJob
type BridgeJobSpec struct {
	// This field is a way to integrate multiple watcher pod. Depending on the the pod name we can communicate with a different external system
	// Currently implemented are include:
	//		LSF - HPC with LSF (https://www.ibm.com/docs/en/slsfh/10.2.0?topic=overview)
	//		SLURM - HPC with SLURM (https://slurm.schedmd.com/documentation.html)
	//		Quantum - quantum integration through IBM Cloud (https://www.ibm.com/quantum-computing/services/)
	//		Ray - ray cluster integration (https://www.ray.io/)
	// +kubebuilder:default:="ibm.com/bridge-operator-lsf-pod:0.1"
	Image string `json:"image,omitempty" description:"Defines a base image used for running Pod"`

	// Use "IfNotPresent" for normal functioning and "Always" when you are testing a pod and plan to iterate on the pod implementations
	// +kubebuilder:default:=IfNotPresent
	ImagePullPolicy apiv1.PullPolicy `json:"imagepullpolicy,omitempty"  description:"Defines image pull policy, default IfNotPresent"`

	// Access to the external resource
	//+kubebuilder:validation:Required
	ResourceURL string `json:"resourceURL" description:"External resource URL"`

	// Secret containing credential for resource access
	//+kubebuilder:validation:Required
	ResourceSecret string `json:"resourcesecret" description:"Secret name with credentials to External resource (HPC cluster); has to be in same namespace"`

	// Update interval for the watcher pod
	// +kubebuilder:default:=20
	UpdateInterval int `json:"updateinterval,omitempty" description:"Status polling interval (in secs). Default is 2 min"`

	// A flag to kill an external job
	JobKill bool `json:"kill,omitempty" description:"Kill job flag, if set pod and job on external resource are killed"`

	// struct of data related to job files
	// +kubebuilder:validation:Required
	JobData JobData `json:"jobdata"`

	// Common job resources for external job (JSON string)
	JobProperties string `json:"jobproperties,omitempty"`

	// struct for S3 access information. If Secret defined, assume we want to use S3
	S3Storage S3 `json:"s3storage,omitempty"`

	// struct for uploading results.
	S3Upload Upload `json:"s3upload,omitempty"`
}

// Job data information
type JobData struct {
	// Job script can get different forms depending on the external system
	// Batch script for HPC - LSF and Slurm
	// Python for quantum and Ray
	// We currently support several ways to specify script:
	//		specify location of script on the remote system - string with the location
	//		inline script content here - the full content of the script as a string
	//		specify script location in S3 in the form of bucket:object - here we assume that overall S3 information,
	//						including URL and security is specified in S3 storage structure
	// Location is specified by Script location
	// +kubebuilder:validation:Required
	JobScript string `json:"jobscript" description:"Depending on the script location, a full path to script in user's home to run or content of the script or its S3 location"`
	// In addition to the script itself, some remote systems require script metadata, for example:
	// In the case of quantum, script metadata includes definition of input/output and intermediate data
	// In the case of Ray script metadata include the list of python libraries that need to be added for execution
	// Metadata is specified in JSON with remote system specific format
	// We currently support several ways to specify script metadata:
	//		inline script metadata content here - the full content of the script metadata as a json string
	//		specify script metadata location in S3 in the form of bucket:object - here we assume that overall S3 information,
	//						including URL and security is specified in S3 storage structure
	// Location is specified by ScriptExtra location
	ScriptMetadata string `json:"scriptmetadata,omitempty"  description:"Depending on the script location, content of the script metadata or its S3 location"`
	// Another component of script is execution parameters
	// parameters specified in JSON with remote system specific format
	// We currently support several ways to specify script parameters:
	//		inline job parameters content here - the full content of the script parameters as a json string
	//		specify job parameters location in S3 in the form of comma separated bucket:object - here we assume that overall S3 information,
	//						including URL and security is specified in S3 storage structure
	// Location is specified by ScriptExtra location
	JobParameters string `json:"jobparameters,omitempty"  description:"Depending on the script location, content of the script parameters or its S3 location"`
	// Script location - Location of script
	// Possible values are:
	//				"remote"
	//				"inline"
	//				"s3"
	// +kubebuilder:default:="remote"
	ScriptLocation string `json:"scriptlocation,omitempty" description:"Script location (default is remote)"`
	// Script extra (metadata/parameters) location - Location of script metadata/parameters
	// Possible values are:
	//				"inline"
	//				"s3"
	// +kubebuilder:default:="inline"
	ScriptExtraLocation string `json:"scriptextralocation,omitempty" description:"Script extras location (default is inline)"`
	// List of additional data files to be uploaded to remote resource
	// A list of S3 locations in the form of comma separated bucket:object pairs - here we assume that overall S3 information,
	//						including URL and security is specified in S3 storage structure
	AdditionalData string `json:"additionaldata,omitempty" description:"A list of additional files to upload to resource"`
}

// S3 connection information
type S3 struct {
	// +kubebuilder:default:=""
	S3Secret string `json:"s3secret,omitempty" description:"If not empty, expects Secret name with S3 credentials"`
	// +kubebuilder:default:=""
	Endpoint string `json:"endpoint,omitempty" description:"If s3secret not empty, expects S3 endpoint"`
	// +kubebuilder:default:=true
	Secure bool `json:"secure,omitempty" description:"If s3secret not empty, expects if S3 endpoint is https (default true)"`
	// +kubebuilder:default:=""
}

// Files upload information
type Upload struct {
	Bucket string `json:"bucket" description:"S3 bucket for uploading results"`
	// Files are uploaded to the specified bucket to the object /jobname/filename
	// Files uploaded by default are: output, errors, and script
	// +kubebuilder:default:=""
	Files string `json:"files,omitempty" description:"String of comma separated additional files to be uploaded to S3 after job ends (.out and .err are always uploaded)"`
}

// BridgeJobStatus defines the observed state of BridgeJob
type BridgeJobStatus struct {
	// Current job status
	JobStatus string `json:"jobstatus,omitempty" description:"Current job status"`

	// Represents time when the job was submitted to External resource (HPC cluster).
	StartTime string `json:"starttime,omitempty"`

	// Represents time when the job in External resource (HPC cluster) was completed.
	CompletionTime string `json:"completiontime,omitempty"`

	// Message filled when job is finished in any state
	// Should contain place where output files are located
	Message string `json:"message,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// BridgeJob is the Schema for the bridgejobs API
type BridgeJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BridgeJobSpec   `json:"spec,omitempty"`
	Status BridgeJobStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BridgeJobList contains a list of BridgeJob
type BridgeJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BridgeJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BridgeJob{}, &BridgeJobList{})
}
