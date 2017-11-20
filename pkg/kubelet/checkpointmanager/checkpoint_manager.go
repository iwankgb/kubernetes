/*
Copyright 2016 The Kubernetes Authors.

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
	"fmt"

	utilstore "k8s.io/kubernetes/pkg/kubelet/util/store"
	utilfs "k8s.io/kubernetes/pkg/util/filesystem"
)

// Checkpoint provides the process checkpoint data
type Checkpoint interface {
	MarshalCheckpoint() ([]byte, error)
	UnmarshalCheckpoint(blob []byte) error
	GetChecksum() uint64
	UpdateChecksum()
}

// CheckpointManager provides the interface to manage checkpoint
type CheckpointManager interface {
	// CreateCheckpoint persists checkpoint in CheckpointStore.
	CreateCheckpoint(checkpointKey string, checkpoint Checkpoint) error
	// GetCheckpoint retrieves checkpoint from CheckpointStore.
	GetCheckpoint(checkpointKey string, checkpoint Checkpoint) error
	// WARNING: RemoveCheckpoint will not return error if checkpoint does not exist.
	RemoveCheckpoint(checkpointKey string) error
	// ListCheckpoint returns the list of existing checkpoints.
	ListCheckpoints() ([]string, error)
}

// CheckpointManagerImpl is an implementation of CheckpointManager. It persists checkpoint in CheckpointStore
type CheckpointManagerImpl struct {
	store utilstore.Store
}

func NewCheckpointManager(checkpointDir string) (CheckpointManager, error) {
	fstore, err := utilstore.NewFileStore(checkpointDir, utilfs.DefaultFs{})
	if err != nil {
		return nil, err
	}
	return &CheckpointManagerImpl{store: fstore}, nil
}

// CreateCheckpoint persists checkpoint in CheckpointStore.
func (manager *CheckpointManagerImpl) CreateCheckpoint(checkpointKey string, checkpoint Checkpoint) error {
	checkpoint.UpdateChecksum()
	blob, err := checkpoint.MarshalCheckpoint()
	if err != nil {
		return err
	}
	return manager.store.Write(checkpointKey, blob)
}

// GetCheckpoint retrieves checkpoint from CheckpointStore.
func (manager *CheckpointManagerImpl) GetCheckpoint(checkpointKey string, checkpoint Checkpoint) error {
	blob, err := manager.store.Read(checkpointKey)
	if err != nil {
		return err
	}
	err = checkpoint.UnmarshalCheckpoint(blob)
	return err
}

// WARNING: RemoveCheckpoint will not return error if checkpoint does not exist.
func (manager *CheckpointManagerImpl) RemoveCheckpoint(checkpointKey string) error {
	return manager.store.Delete(checkpointKey)
}

// ListCheckpoint returns the list of existing checkpoints.
func (manager *CheckpointManagerImpl) ListCheckpoints() ([]string, error) {
	keys, err := manager.store.List()
	if err != nil {
		return []string{}, fmt.Errorf("failed to list checkpoint store: %v", err)
	}
	return keys, nil
}
