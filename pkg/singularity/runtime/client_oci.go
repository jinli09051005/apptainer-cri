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

package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/apptainer/apptainer/pkg/ociruntime"
	"github.com/creack/pty"
	"github.com/golang/glog"
	"github.com/opencontainers/runtime-spec/specs-go"
	syio "github.com/sylabs/singularity-cri/pkg/io"
)

// ErrNotFound us returned when Singularity OCI engine responds with
// corresponding error message and exit status 255
var ErrNotFound = fmt.Errorf("no instance found for provided name")

type (
	// ExecResponse holds result of command execution inside a container.
	ExecResponse struct {
		// Captured command stdout output.
		Stdout []byte
		// Captured command stderr output.
		Stderr []byte
		// Exit code the command finished with.
		ExitCode int32
	}
)

// State returns state of a container with passed id. If runtime fails
// to find object with given id, ErrNotFound is returned.
func (c *CLIClient) State(id string) (*ociruntime.State, error) {
	cmd := append(c.ociBaseCmd, "state", id)
	stateCmd := exec.Command(cmd[0], cmd[1:]...)

	cliResp, err := stateCmd.Output()
	if err != nil {
		if eErr, ok := err.(*exec.ExitError); ok {
			if strings.Contains(string(eErr.Stderr), "no instance found") {
				return nil, ErrNotFound
			}
			return nil, fmt.Errorf("could not query state: %s", eErr.Stderr)
		}
		return nil, fmt.Errorf("could not query state: %v", err)
	}

	var state *ociruntime.State
	err = json.Unmarshal(cliResp, &state)
	if err != nil {
		return nil, fmt.Errorf("could not decode state: %v", err)
	}
	return state, nil
}

// Delete asks runtime to delete container with passed id. If runtime fails
// to find object with given id, ErrNotFound is returned.
func (c *CLIClient) Delete(id string) error {
	cmd := append(c.ociBaseCmd, "delete", id)
	deleteCmd := exec.Command(cmd[0], cmd[1:]...)

	_, err := deleteCmd.Output()
	if err != nil {
		if eErr, ok := err.(*exec.ExitError); ok {
			if strings.Contains(string(eErr.Stderr), "no instance found") {
				return ErrNotFound
			}
			return fmt.Errorf("could not delete instance %s: %s", id, eErr.Stderr)
		}
		return fmt.Errorf("could not delete instance %s: %s", id, err)
	}

	return nil
}

// Create asks runtime to create a container with passed parameters. When stdin is false
// no stdin stream is allocated and all reads from stdin in the container will always result in EOF.
// When no tty is allocated by the runtime, Create returns master end of the allocated tty
// (need to allocate it to separate stderr) that can be used to propagate any input into container,
// if stdin was requested. Master end should be closed as soon as container is
// not running anymore. For pod master end can be closed immediately.
func (c *CLIClient) Create(id, bundle string, stdin, tty bool, flags ...string) (io.WriteCloser, error) {
	var stdinWrite io.WriteCloser

	cmd := append(c.ociBaseCmd, "create")
	cmd = append(cmd, flags...)
	cmd = append(cmd, "-b", bundle, id)

	createCmd := exec.Command(cmd[0], cmd[1:]...)
	createCmd.Stderr = os.Stderr
	if !tty {
		master, slave, err := pty.Open()
		if err != nil {
			return nil, fmt.Errorf("could not allcate pty: %v", err)
		}
		createCmd.Stderr = slave
		defer slave.Close()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go func() {
			glog.V(5).Info("Starting stream copying from master to stderr")
			_, err := io.Copy(os.Stderr, syio.NewContextReader(ctx, master))
			glog.V(5).Infof("Stream copying returned: %v", err)
			// we need to drain master to prevent buffer overflow,
			// see https://github.com/sylabs/singularity-cri/pull/348
			go io.Copy(ioutil.Discard, master)
		}()
		stdinWrite = master

		if stdin {
			createCmd.Stdin = slave
		}
	}

	glog.V(5).Infof("Executing %v", cmd)
	err := createCmd.Run()
	if err != nil {
		return nil, fmt.Errorf("could not execute create container command: %v", err)
	}

	return stdinWrite, nil
}

// Start asks runtime to start container with passed id.
func (c *CLIClient) Start(id string) error {
	cmd := append(c.ociBaseCmd, "start", id)
	return run(cmd)
}

// ExecSync executes a command inside a container synchronously until
// context is done and returns the result.
func (c *CLIClient) ExecSync(ctx context.Context, id string, args, envs []string) (*ExecResponse, error) {
	cmd := append(c.ociBaseCmd, "exec", id)
	cmd = append(cmd, args...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	runCmd := exec.CommandContext(ctx, cmd[0], cmd[1:]...)
	runCmd.Stdout = &stdout
	runCmd.Stderr = &stderr
	runCmd.Env = envs

	glog.V(5).Infof("Executing %v", cmd)
	err := runCmd.Run()
	var exitCode int32
	exitErr, ok := err.(*exec.ExitError)
	if ok {
		// TODO use unix package here
		var waitStatus syscall.WaitStatus
		waitStatus, ok = exitErr.Sys().(syscall.WaitStatus)
		if ok {
			exitCode = int32(waitStatus.ExitStatus())
		}
	}
	if !ok && err != nil {
		return nil, fmt.Errorf("could not execute: %v", err)
	}
	return &ExecResponse{
		Stdout:   stdout.Bytes(),
		Stderr:   stderr.Bytes(),
		ExitCode: exitCode,
	}, nil
}

// Exec executes passed command inside a container setting io streams to passed ones.
func (c *CLIClient) Exec(ctx context.Context, id string,
	stdin io.Reader, stdout, stderr io.Writer,
	args, envs []string) error {

	runCmd := c.PrepareExec(ctx, id, args, envs)
	runCmd.Stdout = stdout
	runCmd.Stderr = stderr
	runCmd.Stdin = stdin

	err := runCmd.Run()
	_, ok := err.(*exec.ExitError)
	if !ok && err != nil {
		return fmt.Errorf("could not execute: %v", err)
	}
	return nil
}

// PrepareExec simply prepares command to call to execute inside a
// given container. It makes sure singularity exec script is called.
func (c *CLIClient) PrepareExec(ctx context.Context, id string, args, envs []string) *exec.Cmd {
	cmd := append(c.ociBaseCmd, "exec", id)
	cmd = append(cmd, args...)

	glog.V(5).Infof("Prepared %v", cmd)
	cmdCtx := exec.CommandContext(ctx, cmd[0], cmd[1:]...)
	cmdCtx.Env = envs
	return cmdCtx
}

// Kill asks runtime to send SIGINT to container with passed id.
// If force is true that SIGKILL is sent instead.
func (c *CLIClient) Kill(id string, force bool) error {
	sig := "SIGINT"
	if force {
		sig = "SIGKILL"
	}
	return c.Signal(id, sig)
}

// Signal asks runtime to send passed sig to container with passed id.
func (c *CLIClient) Signal(id, sig string) error {
	cmd := append(c.ociBaseCmd, "kill", "-s", sig, id)
	return run(cmd)
}

// UpdateContainerResources asks runtime to update container resources
// according to the passed parameter.
func (c *CLIClient) UpdateContainerResources(id string, req *specs.LinuxResources) error {
	buf := bytes.NewBuffer(nil)
	err := json.NewEncoder(buf).Encode(req)
	if err != nil {
		return fmt.Errorf("could not encode update request: %v", err)
	}

	cmd := append(c.ociBaseCmd, "update", "--from-file", "-", id)
	updCmd := exec.Command(cmd[0], cmd[1:]...)
	updCmd.Stderr = os.Stderr
	updCmd.Stdin = buf

	glog.V(5).Infof("Executing %v", cmd)
	err = updCmd.Run()
	if err != nil {
		return fmt.Errorf("could not execute: %v", err)
	}
	return nil
}
