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

package checkpointmanager

import (
	"encoding/json"
	"hash/fnv"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/kubernetes/pkg/kubelet/checkpointmanager/errors"
	utilstore "k8s.io/kubernetes/pkg/kubelet/checkpointmanager/testing"
	hashutil "k8s.io/kubernetes/pkg/util/hash"
)

type Protocol string

// PortMapping is the port mapping configurations of a sandbox.
type PortMapping struct {
	// Protocol of the port mapping.
	Protocol *Protocol
	// Port number within the container.
	ContainerPort *int32
	// Port number on the host.
	HostPort *int32
}

// CheckpointData contains all types of data that can be stored in the checkpoint.
type CheckpointData struct {
	Version      string
	Name         string
	PortMappings []*PortMapping
	HostNetwork  bool
	Checksum     uint64
}

func newFakeCheckpoint(name string, portMappings []*PortMapping, hostnetwork bool) Checkpoint {
	return &CheckpointData{
		Version:      "v1",
		Name:         name,
		PortMappings: portMappings,
		HostNetwork:  hostnetwork,
	}
}

func (cp *CheckpointData) MarshalCheckpoint() ([]byte, error) {
	return json.Marshal(*cp)
}

func (cp *CheckpointData) UnmarshalCheckpoint(blob []byte) error {
	err := json.Unmarshal(blob, cp)
	cksum := cp.Checksum
	if cksum != cp.GetChecksum() {
		return errors.ErrCorruptCheckpoint
	}
	return err
}

func (cp *CheckpointData) GetChecksum() uint64 {
	orig := cp.Checksum
	cp.Checksum = 0
	hash := fnv.New32a()
	hashutil.DeepHashObject(hash, *cp)
	cp.Checksum = orig
	return uint64(hash.Sum32())
}

func (cp *CheckpointData) VerifyChecksum() error {
	if cp.Checksum != cp.GetChecksum() {
		return errors.ErrCorruptCheckpoint
	}
	return nil
}
func (cp *CheckpointData) UpdateChecksum() {
	cp.Checksum = cp.GetChecksum()
}

func NewTestCheckpointManager() CheckpointManager {
	return &Impl{store: utilstore.NewMemStore()}
}

func TestCheckpointManager(t *testing.T) {
	var err error
	manager := NewTestCheckpointManager()
	port80 := int32(80)
	port443 := int32(443)
	proto := Protocol("tcp")

	portMappings := []*PortMapping{
		{
			&proto,
			&port80,
			&port80,
		},
		{
			&proto,
			&port443,
			&port443,
		},
	}
	checkpoint1 := newFakeCheckpoint("check1", portMappings, true)

	checkpoints := []struct {
		checkpointKey     string
		checkpoint        Checkpoint
		expectHostNetwork bool
	}{
		{
			"key1",
			checkpoint1,
			true,
		},
		{
			"key2",
			newFakeCheckpoint("check2", nil, false),
			false,
		},
	}

	for _, tc := range checkpoints {
		// Test CreateCheckpoints
		err = manager.CreateCheckpoint(tc.checkpointKey, tc.checkpoint)
		assert.NoError(t, err)

		// Test GetCheckpoints
		checkpointOut := newFakeCheckpoint("", nil, false)
		err := manager.GetCheckpoint(tc.checkpointKey, checkpointOut)
		assert.NoError(t, err)
		assert.Equal(t, checkpointOut.(*CheckpointData), tc.checkpoint.(*CheckpointData))
		assert.Equal(t, checkpointOut.(*CheckpointData).PortMappings, tc.checkpoint.(*CheckpointData).PortMappings)
		assert.Equal(t, checkpointOut.(*CheckpointData).HostNetwork, tc.expectHostNetwork)
	}
	// Test ListCheckpoints
	keys, err := manager.ListCheckpoints()
	assert.NoError(t, err)
	sort.Strings(keys)
	assert.Equal(t, keys, []string{"key1", "key2"})

	// Test RemoveCheckpoints
	err = manager.RemoveCheckpoint("key1")
	assert.NoError(t, err)
	// Test Remove Nonexisted Checkpoints
	err = manager.RemoveCheckpoint("key1")
	assert.NoError(t, err)

	// Test ListCheckpoints
	keys, err = manager.ListCheckpoints()
	assert.NoError(t, err)
	assert.Equal(t, keys, []string{"key2"})

	// Test Get NonExisted Checkpoint
	checkpointNE := newFakeCheckpoint("NE", nil, false)
	err = manager.GetCheckpoint("key1", checkpointNE)
	assert.Error(t, err)
}
