/*
Copyright 2017 The Kubernetes Authors.

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

package deviceplugin

import (
	"encoding/json"
	"hash/fnv"

	"k8s.io/kubernetes/pkg/kubelet/checkpointmanager"
	"k8s.io/kubernetes/pkg/kubelet/checkpointmanager/errors"
	hashutil "k8s.io/kubernetes/pkg/util/hash"
)

type podDevicesCheckpointEntry struct {
	PodUID        string
	ContainerName string
	ResourceName  string
	DeviceIDs     []string
	AllocResp     []byte
}

// checkpointData struct is used to store pod to device allocation information
// in a checkpoint file.
// TODO: add version control when we need to change checkpoint format.
type checkpointData struct {
	podDeviceEntries  []podDevicesCheckpointEntry
	RegisteredDevices map[string][]string
	Checksum          uint64
}

// NewDevicePluginCheckpoint returns an instance of Checkpoint
func NewDevicePluginCheckpoint() checkpointmanager.Checkpoint {
	return &checkpointData{RegisteredDevices: make(map[string][]string)}
}

// MarshalCheckpoint returns marshalled data
func (cp *checkpointData) MarshalCheckpoint() ([]byte, error) {
	return json.Marshal(*cp)
}

// UnmarshalCheckpoint returns unmarshalled data
func (cp *checkpointData) UnmarshalCheckpoint(blob []byte) error {
	err := json.Unmarshal(blob, cp)
	cksum := cp.Checksum
	if cksum != cp.GetChecksum() {
		return errors.ErrCorruptCheckpoint
	}
	return err
}

// GetChecksum returns calculated checksum of checkpoint data
func (cp *checkpointData) GetChecksum() uint64 {
	orig := cp.Checksum
	cp.Checksum = 0
	hash := fnv.New32a()
	hashutil.DeepHashObject(hash, *cp)
	cp.Checksum = orig
	return uint64(hash.Sum32())
}

// VerifyChecksum verifies that passed checksum is same as calculated checksum
func (cp *checkpointData) VerifyChecksum() error {
	if cp.Checksum != cp.GetChecksum() {
		return errors.ErrCorruptCheckpoint
	}
	return nil
}

// UpdateChecksum updates checksum in the checkpoint data
func (cp *checkpointData) UpdateChecksum() {
	cp.Checksum = cp.GetChecksum()
}

//IsChecksumValid validates stored checksum against newly calculated one
func (cp *checkpointData) IsChecksumValid() bool {
	return cp.Checksum == cp.GetChecksum()
}
