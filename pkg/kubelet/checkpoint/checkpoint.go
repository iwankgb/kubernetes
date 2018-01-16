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

package checkpoint

import (
	"encoding/json"
	"fmt"
	"hash/fnv"

	"github.com/golang/glog"

	"k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/kubelet/checkpointmanager"
	"k8s.io/kubernetes/pkg/kubelet/checkpointmanager/errors"
	hashutil "k8s.io/kubernetes/pkg/util/hash"
)

const (
	// Delimiter used on checkpoints written to disk
	delimiter = "_"
	podPrefix = "Pod"
)

// Data to be stored as checkpoint
type Data struct {
	Pod      *v1.Pod
	Checksum uint64
}

// NewPodCheckpoint returns new pod checkpoint
func NewPodCheckpoint(pod *v1.Pod) checkpointmanager.Checkpoint {
	return &Data{Pod: pod}
}

// MarshalCheckpoint returns marshalled data
func (cp *Data) MarshalCheckpoint() ([]byte, error) {
	return json.Marshal(*cp)
}

// UnmarshalCheckpoint returns unmarshalled data
func (cp *Data) UnmarshalCheckpoint(blob []byte) error {
	err := json.Unmarshal(blob, cp)
	cksum := cp.Checksum
	if cksum != cp.GetChecksum() {
		return errors.ErrCorruptCheckpoint
	}
	return err
}

// GetChecksum returns calculated checksum of checkpoint data
func (cp *Data) GetChecksum() uint64 {
	orig := cp.Checksum
	cp.Checksum = 0
	hash := fnv.New32a()
	hashutil.DeepHashObject(hash, *cp)
	cp.Checksum = orig
	return uint64(hash.Sum32())
}

// VerifyChecksum verifies that passed checksum is same as calculated checksum
func (cp *Data) VerifyChecksum() error {
	if cp.Checksum != cp.GetChecksum() {
		return errors.ErrCorruptCheckpoint
	}
	return nil
}

// UpdateChecksum updates checksum in the checkpoint data
func (cp *Data) UpdateChecksum() {
	cp.Checksum = cp.GetChecksum()
}

// checkAnnotations will validate the checkpoint annotations exist on the Pod
func checkAnnotations(pod *v1.Pod) bool {
	if podAnnotations := pod.GetAnnotations(); podAnnotations != nil {
		if podAnnotations[core.BootstrapCheckpointAnnotationKey] == "true" {
			return true
		}
	}
	return false
}

//getPodKey returns the full qualified path for the pod checkpoint
func getPodKey(pod *v1.Pod) string {
	return fmt.Sprintf("Pod%v%v.yaml", delimiter, pod.GetUID())
}

// LoadPods Loads All Checkpoints from disk
func LoadPods(cpm checkpointmanager.CheckpointManager) ([]*v1.Pod, error) {
	pods := make([]*v1.Pod, 0)

	var err error
	checkpointKeys := []string{}
	checkpointKeys, err = cpm.ListCheckpoints()
	if err != nil {
		glog.Errorf("Failed to list checkpoints: %v", err)
	}

	for _, key := range checkpointKeys {
		checkpoint := NewPodCheckpoint(nil)
		err := cpm.GetCheckpoint(key, checkpoint)
		if err != nil {
			glog.Errorf("Failed to retrieve checkpoint for pod %q: %v", key, err)
			continue
		}
		pods = append(pods, checkpoint.(*Data).Pod)
	}
	return pods, nil
}

// WritePod a checkpoint to a file on disk if annotation is present
func WritePod(cpm checkpointmanager.CheckpointManager, pod *v1.Pod) error {
	var err error
	if checkAnnotations(pod) {
		data := NewPodCheckpoint(pod)
		err = cpm.CreateCheckpoint(getPodKey(pod), data)
	} else {
		// This is to handle an edge where a pod update could remove
		// an annotation and the checkpoint should then be removed.
		err = cpm.RemoveCheckpoint(getPodKey(pod))
	}
	return err
}

// DeletePod deletes a checkpoint from disk if present
func DeletePod(cpm checkpointmanager.CheckpointManager, pod *v1.Pod) error {
	return cpm.RemoveCheckpoint(getPodKey(pod))
}
