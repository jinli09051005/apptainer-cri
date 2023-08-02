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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	ocibundle "github.com/apptainer/apptainer/pkg/ocibundle/sif"
	"github.com/golang/glog"
)

const (
	contSocketPath    = "sync.sock"
	contBundlePath    = "bundle/"
	contRootfsPath    = "rootfs/"
	contOCIConfigPath = "config.json"
)

// ociConfigPath returns path to container's config.json file.
func (c *Container) ociConfigPath() string {
	return filepath.Join(c.baseDir, contBundlePath, contOCIConfigPath)
}

// rootfsPath returns path to container's rootfs directory.
func (c *Container) rootfsPath() string {
	return filepath.Join(c.baseDir, contBundlePath, contRootfsPath)
}

// socketPath returns path to container's sync socket.
func (c *Container) socketPath() string {
	return filepath.Join(c.baseDir, contSocketPath)
}

// bundlePath returns path to container's filesystem bundle directory.
func (c *Container) bundlePath() string {
	return filepath.Join(c.baseDir, contBundlePath)
}

// addLogDirectory creates a dedicated directory for container logs under pod's
// log directory. If pod log directory is not specified, no container logs will be collected
// even if container log path is not empty.
func (c *Container) addLogDirectory() error {
	logDir := c.pod.GetLogDirectory()
	logPath := c.GetLogPath()
	if logDir == "" || logPath == "" {
		return nil
	}

	logPath = filepath.Join(logDir, logPath)
	logDir = filepath.Dir(logPath)
	glog.V(5).Infof("Creating log directory %s", logDir)
	err := os.MkdirAll(logDir, 0755)
	if err != nil {
		return fmt.Errorf("could not create %s: %v", logDir, err)
	}
	c.logPath = logPath
	return nil
}

func (c *Container) addOCIBundle() error {
	glog.V(5).Infof("Creating SIF bundle at %s", c.bundlePath())
	d, err := ocibundle.FromSif(c.imgInfo.Path, c.bundlePath(), true)
	if err != nil {
		return fmt.Errorf("could not create SIF bundle driver: %v", err)
	}
	if err := d.Create(nil); err != nil {
		return fmt.Errorf("could not create SIF bundle: %v", err)
	}

	glog.V(5).Infof("Generating OCI config for container %s", c.id)
	ociSpec, err := translateContainer(c, c.pod)
	if err != nil {
		return fmt.Errorf("could not generate oci spec for container: %v", err)
	}
	config, err := os.OpenFile(c.ociConfigPath(), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("could not create OCI config file: %v", err)
	}
	defer config.Close()
	err = json.NewEncoder(config).Encode(ociSpec)
	if err != nil {
		return fmt.Errorf("could not encode OCI config into json: %v", err)
	}
	return nil
}

func (c *Container) cleanupFiles(silent bool) error {
	glog.V(5).Infof("Removing bundle at %s", c.bundlePath())
	d, err := ocibundle.FromSif("", c.bundlePath(), true)
	if err != nil {
		if !silent {
			return fmt.Errorf("could not create SIF bundle driver: %v", err)
		}
		glog.Errorf("Could not create SIF bundle driver: %v", err)
	}
	if err := d.Delete(); err != nil {
		if !silent {
			return fmt.Errorf("could not delete SIF bundle: %v", err)
		}
		glog.Errorf("Could not delete SIF bundle: %v", err)
	}
	glog.V(5).Infof("Removing container base directory %s", c.baseDir)
	err = os.RemoveAll(c.baseDir)
	if err != nil {
		if !silent {
			return fmt.Errorf("could not cleanup container: %v", err)
		}
		glog.Errorf("Could not cleanup container: %v", err)
	}
	// do not clean any container logs as
	// they will be removed during pod cleanup
	// see https://github.com/sylabs/singularity-cri/issues/314
	return nil
}

func (c *Container) collectTrash() error {
	if c.trashDir == "" {
		return nil
	}
	contTrashDir := filepath.Join(c.trashDir, c.PodID(), c.id)
	err := os.MkdirAll(contTrashDir, 0755)
	if err != nil {
		return fmt.Errorf("could not create trash directory: %v", err)
	}

	err = copyFile(c.ociConfigPath(), filepath.Join(contTrashDir, "config.json"))
	if err != nil {
		return fmt.Errorf("could not save OCI config to trash directory: %v", err)
	}

	if c.logPath == "" {
		return nil
	}

	trashLogs := filepath.Join(contTrashDir, "logs")
	err = os.Mkdir(trashLogs, 0755)
	if err != nil {
		return fmt.Errorf("could not create trash logs directory: %v", err)
	}

	dir := filepath.Dir(c.logPath)
	if dir == c.pod.GetLogDirectory() {
		// container doesn't have its own log directory
		// store a single file only
		err := copyFile(c.logPath, filepath.Join(trashLogs, "1.log"))
		if err != nil {
			return fmt.Errorf("could not copy trash log: %v", err)
		}
		return nil
	}

	// container has its own log directory
	fii, err := ioutil.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("could not read log directory: %v", err)
	}
	for _, fi := range fii {
		err := copyFile(filepath.Join(dir, fi.Name()), filepath.Join(trashLogs, fi.Name()))
		if err != nil {
			return fmt.Errorf("could not copy trash log: %v", err)
		}
	}

	return nil
}
