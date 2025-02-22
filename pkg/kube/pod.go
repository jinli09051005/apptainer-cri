// Copyright (c) 2018-2019 Sylabs, Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kube

import (
	"context"
	"fmt"
	"sync"

	"github.com/apptainer/apptainer/pkg/ociruntime"
	"github.com/golang/glog"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sylabs/singularity-cri/pkg/namespace"
	"github.com/sylabs/singularity-cri/pkg/network"
	"github.com/sylabs/singularity-cri/pkg/rand"
	"github.com/sylabs/singularity-cri/pkg/singularity/runtime"
	k8s "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
)

const (
	// PodIDLen reflects number of symbols in pod unique ID.
	PodIDLen = 64
)

// Pod represents kubernetes pod. It encapsulates all pod-specific
// logic and should be used by runtime for correct interaction.
type Pod struct {
	id string
	*k8s.PodSandboxConfig
	baseDir string

	isStopped bool
	isRemoved bool

	runtimeState runtime.State
	ociState     *ociruntime.State
	namespaces   []specs.LinuxNamespace

	mu         sync.Mutex
	containers []*Container

	cli        *runtime.CLIClient
	syncChan   <-chan runtime.State
	syncCancel context.CancelFunc

	network *network.PodNetwork
}

// NewPod constructs Pod instance. Pod is thread safe to use.
func NewPod(config *k8s.PodSandboxConfig) *Pod {
	podID := rand.GenerateID(PodIDLen)
	return &Pod{
		PodSandboxConfig: config,
		id:               podID,
		cli:              runtime.NewCLIClient(),
	}
}

// ID returns unique pod ID.
func (p *Pod) ID() string {
	return p.id
}

// State returns current pod state.
func (p *Pod) State() k8s.PodSandboxState {
	if p.runtimeState == runtime.StateRunning {
		return k8s.PodSandboxState_SANDBOX_READY
	}
	return k8s.PodSandboxState_SANDBOX_NOTREADY
}

// CreatedAt returns pod creation time in Unix nano.
func (p *Pod) CreatedAt() int64 {
	if p.ociState.CreatedAt == nil {
		return 0
	}
	return *p.ociState.CreatedAt
}

// Run prepares and runs pod based on initial config passed to NewPod.
// All files created (namespaces, sync socket, etc) are located in baseDir.
func (p *Pod) Run(baseDir string) error {
	var err error
	defer func() {
		if err != nil {
			if err := p.terminate(true); err != nil {
				glog.Errorf("Could not kill pod after failed run: %v", err)
			}
			if err := p.cli.Delete(p.id); err != nil {
				glog.Errorf("Could not remove pod: %v", err)
			}
			if err := p.cleanupFiles(true); err != nil {
				glog.Errorf("Could not cleanup pod after failed run: %v", err)
			}
		}
	}()

	p.baseDir = baseDir
	if err = p.validateConfig(); err != nil {
		return fmt.Errorf("invalid pod config: %v", err)
	}
	if err = p.prepareFiles(); err != nil {
		return fmt.Errorf("could not create pod directories: %v", err)
	}
	if err = p.unshareNamespaces(); err != nil {
		return fmt.Errorf("could not unshare namespaces: %v", err)
	}
	if err = p.spawnOCIPod(); err != nil {
		return fmt.Errorf("could not spawn pod: %v", err)
	}
	if err = p.UpdateState(); err != nil {
		return fmt.Errorf("could not update pod state: %v", err)
	}
	return nil
}

// Stop stops pod and all its containers, reclaims any resources.
func (p *Pod) Stop() error {
	if p.isStopped {
		return nil
	}

	for _, c := range p.containers {
		err := c.Stop(0)
		if err != nil {
			return fmt.Errorf("could not stop container %s: %v", c.id, err)
		}
	}

	err := p.terminate(false)
	if err != nil {
		return fmt.Errorf("could not stop pod process: %v", err)
	}
	if err := p.UpdateState(); err != nil {
		return fmt.Errorf("could not update container state: %v", err)
	}
	p.isStopped = true
	return err
}

// Remove removes pod and all its containers, making sure nothing
// of it left on the host filesystem. When no Stop is called before
// Remove forcibly kills all containers and pod itself.
func (p *Pod) Remove() error {
	if p.isRemoved {
		return nil
	}

	for _, c := range p.containers {
		err := c.Remove()
		if err != nil {
			return fmt.Errorf("could not remove container %s: %v", c.id, err)
		}
	}

	if err := p.terminate(true); err != nil {
		return fmt.Errorf("could not kill pod process: %v", err)
	}
	if err := p.cli.Delete(p.id); err != nil && err != runtime.ErrNotFound {
		return fmt.Errorf("could not remove pod: %v", err)
	}
	if err := p.cleanupFiles(false); err != nil {
		glog.Errorf("Pod cleanup failed: %v", err)
	}
	p.isRemoved = true
	return nil
}

// MatchesFilter tests Pod against passed filter and returns true if it matches.
func (p *Pod) MatchesFilter(filter *k8s.PodSandboxFilter) bool {
	if filter == nil {
		return true
	}

	if filter.Id != "" && filter.Id != p.id {
		return false
	}

	if filter.State != nil && filter.State.State != p.State() {
		return false
	}

	for k, v := range filter.LabelSelector {
		label, ok := p.Labels[k]
		if !ok {
			return false
		}
		if v != label {
			return false
		}
	}
	return true
}

// Containers return list or container IDs that are in this pod.
func (p *Pod) Containers() []string {
	var containers []string
	for _, c := range p.containers {
		containers = append(containers, c.id)
	}
	return containers
}

func (p *Pod) addContainer(cont *Container) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, c := range p.containers {
		if c.id == cont.id {
			return
		}
	}
	p.containers = append(p.containers, cont)
}

func (p *Pod) removeContainer(cont *Container) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i, c := range p.containers {
		if c.id == cont.id {
			p.containers = append(p.containers[:i], p.containers[i+1:]...)
			return
		}
	}
}

func (p *Pod) unshareNamespaces() error {
	p.namespaces = append(p.namespaces, specs.LinuxNamespace{
		Type: specs.UTSNamespace,
		Path: p.bindNamespacePath(specs.UTSNamespace),
	})
	security := p.GetLinux().GetSecurityContext()
	if security.GetNamespaceOptions().GetNetwork() == k8s.NamespaceMode_POD {
		p.namespaces = append(p.namespaces, specs.LinuxNamespace{
			Type: specs.NetworkNamespace,
			Path: p.bindNamespacePath(specs.NetworkNamespace),
		})
	}
	if security.GetNamespaceOptions().GetIpc() == k8s.NamespaceMode_POD {
		p.namespaces = append(p.namespaces, specs.LinuxNamespace{
			Type: specs.IPCNamespace,
			Path: p.bindNamespacePath(specs.IPCNamespace),
		})
	}
	if err := namespace.UnshareAll(p.namespaces); err != nil {
		return fmt.Errorf("unsahre all failed: %v", err)
	}
	return nil
}
