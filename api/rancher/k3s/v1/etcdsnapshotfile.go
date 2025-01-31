/*
Copyright © 2023 - 2024 SUSE LLC

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

// ETCDSnapshotFile is the struct representing a k3s ETCDSnapshotFile.
// +kubebuilder:object:root=true
type ETCDSnapshotFile struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ETCDSnapshotSpec   `json:"spec,omitempty"`
	Status ETCDSnapshotStatus `json:"status,omitempty"`
}

// ETCDSnapshotSpec is the struct spec representing a k3s ETCDSnapshotFile.
type ETCDSnapshotSpec struct {
	SnapshotName string            `json:"snapshotName"`
	NodeName     string            `json:"nodeName"`
	Location     string            `json:"location"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	S3           *ETCDSnapshotS3   `json:"s3,omitempty"`
}

// ETCDSnapshotS3 is the struct representing a k3s ETCDSnapshotFile S3.
type ETCDSnapshotS3 struct {
	Endpoint      string `json:"endpoint,omitempty"`
	EndpointCA    string `json:"endpointCA,omitempty"`
	SkipSSLVerify bool   `json:"skipSSLVerify,omitempty"`
	Bucket        string `json:"bucket,omitempty"`
	Region        string `json:"region,omitempty"`
	Insecure      bool   `json:"insecure,omitempty"`
}

// ETCDSnapshotStatus is the status of the k3s ETCDSnapshotFile.
type ETCDSnapshotStatus struct {
	// CreationTime is the timestamp when the snapshot was taken by etcd.
	CreationTime *metav1.Time `json:"creationTime,omitempty"`
	// ReadyToUse indicates that the snapshot is available to be restored.
	ReadyToUse *bool `json:"readyToUse,omitempty"`
	// Error is the last observed error during snapshot creation, if any.
	// If the snapshot is retried, this field will be cleared on success.
	Error *ETCDSnapshotError `json:"error,omitempty"`
}

// ETCDSnapshotError describes an error encountered during snapshot creation.
type ETCDSnapshotError struct {
	// Time is the timestamp when the error was encountered.
	Time *metav1.Time `json:"time,omitempty"`
	// Message is a string detailing the encountered error during snapshot creation if specified.
	// NOTE: message may be logged, and it should not contain sensitive information.
	Message *string `json:"message,omitempty"`
}

// ETCDSnapshotFileList contains a list of the k3s ETCDSnapshotFiles.
// +kubebuilder:object:root=true
type ETCDSnapshotFileList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ETCDSnapshotFile `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ETCDSnapshotFile{}, &ETCDSnapshotFileList{})
}
