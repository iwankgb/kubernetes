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
	CheckSum          uint64
}

func NewDevicePluginCheckpoint() checkpointmanager.Checkpoint {
	return &checkpointData{RegisteredDevices: make(map[string][]string)}
}

func (cp *checkpointData) MarshalCheckpoint() ([]byte, error) {
	return json.Marshal(*cp)
}

func (cp *checkpointData) UnmarshalCheckpoint(blob []byte) error {
	err := json.Unmarshal(blob, cp)
	cksum := cp.CheckSum
	if cksum != cp.GetChecksum() {
		return errors.CorruptCheckpointError
	}
	return err
}

func (cp *checkpointData) GetChecksum() uint64 {
	orig := cp.CheckSum
	cp.CheckSum = 0
	hash := fnv.New32a()
	hashutil.DeepHashObject(hash, *cp)
	cp.CheckSum = orig
	return uint64(hash.Sum32())
}

func (cp *checkpointData) VerifyChecksum() error {
	if cp.CheckSum != cp.GetChecksum() {
		return errors.CorruptCheckpointError
	}
	return nil
}

func (cp *checkpointData) UpdateChecksum() {
	cp.CheckSum = cp.GetChecksum()
}
