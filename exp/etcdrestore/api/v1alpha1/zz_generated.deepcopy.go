//go:build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EtcdMachineSnapshot) DeepCopyInto(out *EtcdMachineSnapshot) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EtcdMachineSnapshot.
func (in *EtcdMachineSnapshot) DeepCopy() *EtcdMachineSnapshot {
	if in == nil {
		return nil
	}
	out := new(EtcdMachineSnapshot)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *EtcdMachineSnapshot) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EtcdMachineSnapshotList) DeepCopyInto(out *EtcdMachineSnapshotList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]EtcdMachineSnapshot, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EtcdMachineSnapshotList.
func (in *EtcdMachineSnapshotList) DeepCopy() *EtcdMachineSnapshotList {
	if in == nil {
		return nil
	}
	out := new(EtcdMachineSnapshotList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *EtcdMachineSnapshotList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EtcdMachineSnapshotSpec) DeepCopyInto(out *EtcdMachineSnapshotSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EtcdMachineSnapshotSpec.
func (in *EtcdMachineSnapshotSpec) DeepCopy() *EtcdMachineSnapshotSpec {
	if in == nil {
		return nil
	}
	out := new(EtcdMachineSnapshotSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EtcdMachineSnapshotStatus) DeepCopyInto(out *EtcdMachineSnapshotStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EtcdMachineSnapshotStatus.
func (in *EtcdMachineSnapshotStatus) DeepCopy() *EtcdMachineSnapshotStatus {
	if in == nil {
		return nil
	}
	out := new(EtcdMachineSnapshotStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EtcdSnapshotRestore) DeepCopyInto(out *EtcdSnapshotRestore) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EtcdSnapshotRestore.
func (in *EtcdSnapshotRestore) DeepCopy() *EtcdSnapshotRestore {
	if in == nil {
		return nil
	}
	out := new(EtcdSnapshotRestore)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *EtcdSnapshotRestore) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EtcdSnapshotRestoreList) DeepCopyInto(out *EtcdSnapshotRestoreList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]EtcdSnapshotRestore, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EtcdSnapshotRestoreList.
func (in *EtcdSnapshotRestoreList) DeepCopy() *EtcdSnapshotRestoreList {
	if in == nil {
		return nil
	}
	out := new(EtcdSnapshotRestoreList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *EtcdSnapshotRestoreList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EtcdSnapshotRestoreSpec) DeepCopyInto(out *EtcdSnapshotRestoreSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EtcdSnapshotRestoreSpec.
func (in *EtcdSnapshotRestoreSpec) DeepCopy() *EtcdSnapshotRestoreSpec {
	if in == nil {
		return nil
	}
	out := new(EtcdSnapshotRestoreSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EtcdSnapshotRestoreStatus) DeepCopyInto(out *EtcdSnapshotRestoreStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EtcdSnapshotRestoreStatus.
func (in *EtcdSnapshotRestoreStatus) DeepCopy() *EtcdSnapshotRestoreStatus {
	if in == nil {
		return nil
	}
	out := new(EtcdSnapshotRestoreStatus)
	in.DeepCopyInto(out)
	return out
}
