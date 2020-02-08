/*
Copyright 2018 Openstorage.org

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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1alpha1 "github.com/libopenstorage/stork/pkg/apis/stork/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeVolumeSnapshotRestores implements VolumeSnapshotRestoreInterface
type FakeVolumeSnapshotRestores struct {
	Fake *FakeStorkV1alpha1
	ns   string
}

var volumesnapshotrestoresResource = schema.GroupVersionResource{Group: "stork.libopenstorage.org", Version: "v1alpha1", Resource: "volumesnapshotrestores"}

var volumesnapshotrestoresKind = schema.GroupVersionKind{Group: "stork.libopenstorage.org", Version: "v1alpha1", Kind: "VolumeSnapshotRestore"}

// Get takes name of the volumeSnapshotRestore, and returns the corresponding volumeSnapshotRestore object, and an error if there is any.
func (c *FakeVolumeSnapshotRestores) Get(name string, options v1.GetOptions) (result *v1alpha1.VolumeSnapshotRestore, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(volumesnapshotrestoresResource, c.ns, name), &v1alpha1.VolumeSnapshotRestore{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.VolumeSnapshotRestore), err
}

// List takes label and field selectors, and returns the list of VolumeSnapshotRestores that match those selectors.
func (c *FakeVolumeSnapshotRestores) List(opts v1.ListOptions) (result *v1alpha1.VolumeSnapshotRestoreList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(volumesnapshotrestoresResource, volumesnapshotrestoresKind, c.ns, opts), &v1alpha1.VolumeSnapshotRestoreList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.VolumeSnapshotRestoreList{ListMeta: obj.(*v1alpha1.VolumeSnapshotRestoreList).ListMeta}
	for _, item := range obj.(*v1alpha1.VolumeSnapshotRestoreList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested volumeSnapshotRestores.
func (c *FakeVolumeSnapshotRestores) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(volumesnapshotrestoresResource, c.ns, opts))

}

// Create takes the representation of a volumeSnapshotRestore and creates it.  Returns the server's representation of the volumeSnapshotRestore, and an error, if there is any.
func (c *FakeVolumeSnapshotRestores) Create(volumeSnapshotRestore *v1alpha1.VolumeSnapshotRestore) (result *v1alpha1.VolumeSnapshotRestore, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(volumesnapshotrestoresResource, c.ns, volumeSnapshotRestore), &v1alpha1.VolumeSnapshotRestore{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.VolumeSnapshotRestore), err
}

// Update takes the representation of a volumeSnapshotRestore and updates it. Returns the server's representation of the volumeSnapshotRestore, and an error, if there is any.
func (c *FakeVolumeSnapshotRestores) Update(volumeSnapshotRestore *v1alpha1.VolumeSnapshotRestore) (result *v1alpha1.VolumeSnapshotRestore, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(volumesnapshotrestoresResource, c.ns, volumeSnapshotRestore), &v1alpha1.VolumeSnapshotRestore{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.VolumeSnapshotRestore), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeVolumeSnapshotRestores) UpdateStatus(volumeSnapshotRestore *v1alpha1.VolumeSnapshotRestore) (*v1alpha1.VolumeSnapshotRestore, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(volumesnapshotrestoresResource, "status", c.ns, volumeSnapshotRestore), &v1alpha1.VolumeSnapshotRestore{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.VolumeSnapshotRestore), err
}

// Delete takes name of the volumeSnapshotRestore and deletes it. Returns an error if one occurs.
func (c *FakeVolumeSnapshotRestores) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(volumesnapshotrestoresResource, c.ns, name), &v1alpha1.VolumeSnapshotRestore{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeVolumeSnapshotRestores) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(volumesnapshotrestoresResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.VolumeSnapshotRestoreList{})
	return err
}

// Patch applies the patch and returns the patched volumeSnapshotRestore.
func (c *FakeVolumeSnapshotRestores) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.VolumeSnapshotRestore, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(volumesnapshotrestoresResource, c.ns, name, pt, data, subresources...), &v1alpha1.VolumeSnapshotRestore{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.VolumeSnapshotRestore), err
}
