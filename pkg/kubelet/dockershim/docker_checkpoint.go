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

package dockershim

import (
	"encoding/json"
	"hash/fnv"

	"k8s.io/kubernetes/pkg/kubelet/checkpointmanager"
	"k8s.io/kubernetes/pkg/kubelet/checkpointmanager/errors"
	hashutil "k8s.io/kubernetes/pkg/util/hash"
)

const (
	// default directory to store pod sandbox checkpoint files
	sandboxCheckpointDir = "sandbox"
	protocolTCP          = Protocol("tcp")
	protocolUDP          = Protocol("udp")
	schemaVersion        = "v1"
)

type Protocol string

// PortMapping is the port mapping configurations of a sandbox.
type PortMapping struct {
	// Protocol of the port mapping.
	Protocol *Protocol `json:"protocol,omitempty"`
	// Port number within the container.
	ContainerPort *int32 `json:"container_port,omitempty"`
	// Port number on the host.
	HostPort *int32 `json:"host_port,omitempty"`
}

// CheckpointData contains all types of data that can be stored in the checkpoint.
type CheckpointData struct {
	PortMappings []*PortMapping `json:"port_mappings,omitempty"`
	HostNetwork  bool           `json:"host_network,omitempty"`
}

// PodSandboxCheckpoint is the checkpoint structure for a sandbox
type PodSandboxCheckpoint struct {
	// Version of the pod sandbox checkpoint schema.
	Version string `json:"version"`
	// Pod name of the sandbox. Same as the pod name in the PodSpec.
	Name string `json:"name"`
	// Pod namespace of the sandbox. Same as the pod namespace in the PodSpec.
	Namespace string `json:"namespace"`
	// Data to checkpoint for pod sandbox.
	Data *CheckpointData `json:"data,omitempty"`
	// Checksum is calculated with fnv hash of the checkpoint object with checksum field set to be zero
	Checksum uint64 `json:"checksum"`
}

func NewPodSandboxCheckpoint(namespace, name string) checkpointmanager.Checkpoint {
	return &PodSandboxCheckpoint{
		Version:   schemaVersion,
		Namespace: namespace,
		Name:      name,
		Data:      &CheckpointData{},
	}
}

func (cp *PodSandboxCheckpoint) MarshalCheckpoint() ([]byte, error) {
	return json.Marshal(*cp)
}

func (cp *PodSandboxCheckpoint) UnmarshalCheckpoint(blob []byte) error {
	err := json.Unmarshal(blob, cp)
	return err
}

func (cp *PodSandboxCheckpoint) GetChecksum() uint64 {
	orig := cp.Checksum
	cp.Checksum = 0
	hash := fnv.New32a()
	hashutil.DeepHashObject(hash, *cp)
	cp.Checksum = orig
	return uint64(hash.Sum32())
}

func (cp *PodSandboxCheckpoint) VerifyChecksum() error {
	if cp.Checksum != cp.GetChecksum() {
		return errors.ErrCorruptCheckpoint
	}
	return nil
}

func (cp *PodSandboxCheckpoint) UpdateChecksum() {
	cp.Checksum = cp.GetChecksum()
}

//IsChecksumValid validates stored checksum against newly calculated one
func (cp *PodSandboxCheckpoint) IsChecksumValid() bool {
	return cp.Checksum == cp.GetChecksum()
}
