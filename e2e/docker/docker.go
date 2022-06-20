// Copyright (c) 2019-2021 Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package docker

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	dockerclient "github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/e2e/internal/testhelper"
	"github.com/sylabs/singularity/internal/pkg/test/tool/require"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
	"golang.org/x/sys/unix"
)

type ctx struct {
	env e2e.TestEnv
}

func (c ctx) testDockerPulls(t *testing.T) {
	const tmpContainerFile = "test_container.sif"

	tmpPath, err := fs.MakeTmpDir(c.env.TestDir, "docker-", 0o755)
	err = errors.Wrapf(err, "creating temporary directory in %q for docker pull test", c.env.TestDir)
	if err != nil {
		t.Fatalf("failed to create temporary directory: %+v", err)
	}
	defer os.RemoveAll(tmpPath)

	tmpImage := filepath.Join(tmpPath, tmpContainerFile)

	tests := []struct {
		name    string
		options []string
		image   string
		uri     string
		exit    int
	}{
		{
			name:  "AlpineLatestPull",
			image: tmpImage,
			uri:   "docker://alpine:latest",
			exit:  0,
		},
		{
			name:  "Alpine3.9Pull",
			image: filepath.Join(tmpPath, "alpine.sif"),
			uri:   "docker://alpine:3.9",
			exit:  0,
		},
		{
			name:    "Alpine3.9ForcePull",
			options: []string{"--force"},
			image:   tmpImage,
			uri:     "docker://alpine:3.9",
			exit:    0,
		},
		{
			name:    "BusyboxLatestPull",
			options: []string{"--force"},
			image:   tmpImage,
			uri:     "docker://busybox:latest",
			exit:    0,
		},
		{
			name:  "BusyboxLatestPullFail",
			image: tmpImage,
			uri:   "docker://busybox:latest",
			exit:  255,
		},
		{
			name:    "Busybox1.28Pull",
			options: []string{"--force", "--dir", tmpPath},
			image:   tmpContainerFile,
			uri:     "docker://busybox:1.28",
			exit:    0,
		},
		{
			name:  "Busybox1.28PullFail",
			image: tmpImage,
			uri:   "docker://busybox:1.28",
			exit:  255,
		},
		{
			name:  "Busybox1.28PullDirFail",
			image: "/foo/sif.sif",
			uri:   "docker://busybox:1.28",
			exit:  255,
		},
	}

	for _, tt := range tests {
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("pull"),
			e2e.WithArgs(append(tt.options, tt.image, tt.uri)...),
			e2e.PostRun(func(t *testing.T) {
				if !t.Failed() && tt.exit == 0 {
					path := tt.image
					// handle the --dir case
					if path == tmpContainerFile {
						path = filepath.Join(tmpPath, tmpContainerFile)
					}
					c.env.ImageVerify(t, path, e2e.UserProfile)
				}
			}),
			e2e.ExpectExit(tt.exit),
		)
	}
}

// Testing DOCKER_ host support (only if docker available)
func (c ctx) testDockerHost(t *testing.T) {
	require.Command(t, "docker")

	// Write a temporary "empty" Dockerfile (from scratch)
	tmpPath, err := fs.MakeTmpDir(c.env.TestDir, "docker-", 0o755)
	err = errors.Wrapf(err, "creating temporary directory in %q for docker host test", c.env.TestDir)
	if err != nil {
		t.Fatalf("failed to create temporary directory: %+v", err)
	}
	defer os.RemoveAll(tmpPath)

	// Here is the Dockerfile, and a temporary image path
	dockerfile := filepath.Join(tmpPath, "Dockerfile")
	emptyfile := filepath.Join(tmpPath, "file")
	tmpImage := filepath.Join(tmpPath, "scratch-tmp.sif")

	// Write Dockerfile and empty file to file
	dockerfileContent := []byte("FROM scratch\nCOPY file /mrbigglesworth")
	err = os.WriteFile(dockerfile, dockerfileContent, 0o644)
	if err != nil {
		t.Fatalf("failed to create temporary Dockerfile: %+v", err)
	}

	fileContent := []byte("")
	err = os.WriteFile(emptyfile, fileContent, 0o644)
	if err != nil {
		t.Fatalf("failed to create empty file: %+v", err)
	}

	dockerURI := "docker-test-image:host"
	pullURI := "docker-daemon://" + dockerURI

	// Invoke docker build to build an empty scratch image in the docker daemon.
	// Use os/exec because easier to generate a command with a working directory
	cmd := exec.Command("docker", "build", "-t", dockerURI, ".")
	cmd.Dir = tmpPath
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("Unexpected error while running command.\n%s: %s", err, string(out))
	}

	tests := []struct {
		name       string
		envarName  string
		envarValue string
		exit       int
	}{
		// Unset docker host should use default and succeed
		{
			name:       "singularityDockerHostEmpty",
			envarName:  "SINGULARITY_DOCKER_HOST",
			envarValue: "",
			exit:       0,
		},
		{
			name:       "dockerHostEmpty",
			envarName:  "DOCKER_HOST",
			envarValue: "",
			exit:       0,
		},

		// bad Docker host should fail
		{
			name:       "singularityDockerHostInvalid",
			envarName:  "SINGULARITY_DOCKER_HOST",
			envarValue: "tcp://192.168.59.103:oops",
			exit:       255,
		},
		{
			name:       "dockerHostInvalid",
			envarName:  "DOCKER_HOST",
			envarValue: "tcp://192.168.59.103:oops",
			exit:       255,
		},

		// Set to default should succeed
		// The default host varies based on OS, so we use dockerclient default
		{
			name:       "singularityDockerHostValid",
			envarName:  "SINGULARITY_DOCKER_HOST",
			envarValue: dockerclient.DefaultDockerHost,
			exit:       0,
		},
		{
			name:       "dockerHostValid",
			envarName:  "DOCKER_HOST",
			envarValue: dockerclient.DefaultDockerHost,
			exit:       0,
		},
	}

	for _, tt := range tests {
		// Export variable to environment if it's defined
		if tt.envarValue != "" {
			c.env.RunSingularity(
				t,
				e2e.WithEnv(append(os.Environ(), tt.envarValue)),
				e2e.AsSubtest(tt.name),
				e2e.WithProfile(e2e.UserProfile),
				e2e.WithCommand("pull"),
				e2e.WithArgs(tmpImage, pullURI),
				e2e.ExpectExit(tt.exit),
			)
		} else {
			c.env.RunSingularity(
				t,
				e2e.AsSubtest(tt.name),
				e2e.WithProfile(e2e.UserProfile),
				e2e.WithCommand("pull"),
				e2e.WithArgs(tmpImage, pullURI),
				e2e.ExpectExit(tt.exit),
			)
		}
	}

	// Clean up docker image
	cmd = exec.Command("docker", "rmi", dockerURI)
	_, err = cmd.Output()
	if err != nil {
		t.Fatalf("Unexpected error while cleaning up docker image.\n%s", err)
	}
}

// AUFS sanity tests
func (c ctx) testDockerAUFS(t *testing.T) {
	imageDir, cleanup := e2e.MakeTempDir(t, c.env.TestDir, "aufs-", "")
	defer cleanup(t)
	imagePath := filepath.Join(imageDir, "container")

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("build"),
		e2e.WithArgs([]string{imagePath, "docker://sylabsio/aufs-sanity"}...),
		e2e.ExpectExit(0),
	)

	if t.Failed() {
		return
	}

	fileTests := []struct {
		name string
		argv []string
		exit int
	}{
		{
			name: "File 2",
			argv: []string{imagePath, "ls", "/test/whiteout-dir/file2", "/test/whiteout-file/file2", "/test/normal-dir/file2"},
			exit: 0,
		},
		{
			name: "File1",
			argv: []string{imagePath, "ls", "/test/whiteout-dir/file1", "/test/whiteout-file/file1"},
			exit: 1,
		},
	}

	for _, tt := range fileTests {
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("exec"),
			e2e.WithArgs(tt.argv...),
			e2e.ExpectExit(tt.exit),
		)
	}
}

// Check force permissions for user builds #977
func (c ctx) testDockerPermissions(t *testing.T) {
	imageDir, cleanup := e2e.MakeTempDir(t, c.env.TestDir, "perm-", "")
	defer cleanup(t)
	imagePath := filepath.Join(imageDir, "container")

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("build"),
		e2e.WithArgs([]string{imagePath, "docker://sylabsio/userperms"}...),
		e2e.ExpectExit(0),
	)

	if t.Failed() {
		return
	}

	fileTests := []struct {
		name string
		argv []string
		exit int
	}{
		{
			name: "TestDir",
			argv: []string{imagePath, "ls", "/testdir/"},
			exit: 0,
		},
		{
			name: "TestDirFile",
			argv: []string{imagePath, "ls", "/testdir/testfile"},
			exit: 1,
		},
	}
	for _, tt := range fileTests {
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("exec"),
			e2e.WithArgs(tt.argv...),
			e2e.ExpectExit(tt.exit),
		)
	}
}

// Check whiteout of symbolic links #1592 #1576
func (c ctx) testDockerWhiteoutSymlink(t *testing.T) {
	imageDir, cleanup := e2e.MakeTempDir(t, c.env.TestDir, "whiteout-", "")
	defer cleanup(t)
	imagePath := filepath.Join(imageDir, "container")

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("build"),
		e2e.WithArgs([]string{imagePath, "docker://sylabsio/linkwh"}...),
		e2e.PostRun(func(t *testing.T) {
			if t.Failed() {
				return
			}
			c.env.ImageVerify(t, imagePath, e2e.UserProfile)
		}),
		e2e.ExpectExit(0),
	)
}

func (c ctx) testDockerDefFile(t *testing.T) {
	imageDir, cleanup := e2e.MakeTempDir(t, c.env.TestDir, "def-", "")
	defer cleanup(t)
	imagePath := filepath.Join(imageDir, "container")

	getKernelMajor := func(t *testing.T) (major int) {
		var buf unix.Utsname
		if err := unix.Uname(&buf); err != nil {
			err = errors.Wrap(err, "getting current kernel information")
			t.Fatalf("uname failed: %+v", err)
		}
		n, err := fmt.Sscanf(string(buf.Release[:]), "%d.", &major)
		err = errors.Wrap(err, "getting current kernel release")
		if err != nil {
			t.Fatalf("Sscanf failed, n=%d: %+v", n, err)
		}
		if n != 1 {
			t.Fatalf("Unexpected result while getting major release number: n=%d", n)
		}
		return
	}

	tests := []struct {
		name                string
		kernelMajorRequired int
		archRequired        string
		from                string
	}{
		// This only provides Arch for amd64
		{
			name:                "Arch",
			kernelMajorRequired: 3,
			archRequired:        "amd64",
			from:                "dock0/arch:latest",
		},
		{
			name:                "BusyBox",
			kernelMajorRequired: 0,
			from:                "busybox:latest",
		},
		// CentOS6 is for amd64 (or i386) only
		{
			name:                "CentOS_6",
			kernelMajorRequired: 0,
			archRequired:        "amd64",
			from:                "centos:6",
		},
		{
			name:                "CentOS_7",
			kernelMajorRequired: 0,
			from:                "centos:7",
		},
		{
			name:                "AlmaLinux_8",
			kernelMajorRequired: 3,
			from:                "almalinux:8",
		},
		{
			name:                "Ubuntu_1604",
			kernelMajorRequired: 0,
			from:                "ubuntu:16.04",
		},
		{
			name:                "Ubuntu_1804",
			kernelMajorRequired: 3,
			from:                "ubuntu:18.04",
		},
	}

	for _, tt := range tests {
		deffile := e2e.PrepareDefFile(e2e.DefFileDetails{
			Bootstrap: "docker",
			From:      tt.from,
		})

		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.RootProfile),
			e2e.WithCommand("build"),
			e2e.WithArgs([]string{imagePath, deffile}...),
			e2e.PreRun(func(t *testing.T) {
				require.Arch(t, tt.archRequired)
				if getKernelMajor(t) < tt.kernelMajorRequired {
					t.Skipf("kernel >=%v.x required", tt.kernelMajorRequired)
				}
			}),
			e2e.PostRun(func(t *testing.T) {
				defer os.Remove(imagePath)
				defer os.Remove(deffile)

				if t.Failed() {
					return
				}

				c.env.ImageVerify(t, imagePath, e2e.RootProfile)
			}),
			e2e.ExpectExit(0),
		)
	}
}

func (c ctx) testDockerRegistry(t *testing.T) {
	imageDir, cleanup := e2e.MakeTempDir(t, c.env.TestDir, "registry-", "")
	defer cleanup(t)
	imagePath := filepath.Join(imageDir, "container")

	e2e.EnsureRegistry(t)

	tests := []struct {
		name string
		exit int
		dfd  e2e.DefFileDetails
	}{
		{
			name: "BusyBox",
			exit: 0,
			dfd: e2e.DefFileDetails{
				Bootstrap: "docker",
				From:      "localhost:5000/my-busybox",
			},
		},
		{
			name: "BusyBoxRegistry",
			exit: 0,
			dfd: e2e.DefFileDetails{
				Bootstrap: "docker",
				From:      "my-busybox",
				Registry:  "localhost:5000",
			},
		},
		{
			name: "BusyBoxNamespace",
			exit: 255,
			dfd: e2e.DefFileDetails{
				Bootstrap: "docker",
				From:      "my-busybox",
				Registry:  "localhost:5000",
				Namespace: "not-a-namespace",
			},
		},
	}

	for _, tt := range tests {
		defFile := e2e.PrepareDefFile(tt.dfd)

		c.env.RunSingularity(
			t,
			e2e.WithProfile(e2e.RootProfile),
			e2e.WithCommand("build"),
			e2e.WithArgs("--no-https", imagePath, defFile),
			e2e.PostRun(func(t *testing.T) {
				defer os.Remove(imagePath)
				defer os.Remove(defFile)

				if t.Failed() || tt.exit != 0 {
					return
				}

				c.env.ImageVerify(t, imagePath, e2e.RootProfile)
			}),
			e2e.ExpectExit(tt.exit),
		)
	}
}

func (c ctx) testDockerLabels(t *testing.T) {
	imageDir, cleanup := e2e.MakeTempDir(t, c.env.TestDir, "labels-", "")
	defer cleanup(t)
	imagePath := filepath.Join(imageDir, "container")

	// Test container & set labels
	// See: https://github.com/sylabs/singularity-test-containers/pull/1
	imgSrc := "docker://sylabsio/labels"
	label1 := "LABEL1: 1"
	label2 := "LABEL2: TWO"

	c.env.RunSingularity(
		t,
		e2e.AsSubtest("build"),
		e2e.WithProfile(e2e.RootProfile),
		e2e.WithCommand("build"),
		e2e.WithArgs(imagePath, imgSrc),
		e2e.ExpectExit(0),
	)

	verifyOutput := func(t *testing.T, r *e2e.SingularityCmdResult) {
		output := string(r.Stdout)
		for _, l := range []string{label1, label2} {
			if !strings.Contains(output, l) {
				t.Errorf("Did not find expected label %s in inspect output", l)
			}
		}
	}

	c.env.RunSingularity(
		t,
		e2e.AsSubtest("inspect"),
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("inspect"),
		e2e.WithArgs([]string{"--labels", imagePath}...),
		e2e.ExpectExit(0, verifyOutput),
	)
}

//nolint:dupl
func (c ctx) testDockerCMD(t *testing.T) {
	imageDir, cleanup := e2e.MakeTempDir(t, c.env.TestDir, "docker-", "")
	defer cleanup(t)
	imagePath := filepath.Join(imageDir, "container")

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("while getting $HOME: %s", err)
	}

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("pull"),
		e2e.WithArgs(imagePath, "docker://sylabsio/docker-cmd"),
		e2e.ExpectExit(0),
	)

	tests := []struct {
		name         string
		args         []string
		noeval       bool
		expectOutput string
	}{
		// Singularity historic behavior (without --no-eval)
		// These do not all match Docker, due to evaluation, consumption of quoting.
		{
			name:         "default",
			args:         []string{},
			noeval:       false,
			expectOutput: `CMD 'quotes' "quotes" $DOLLAR s p a c e s`,
		},
		{
			name:         "override",
			args:         []string{"echo", "test"},
			noeval:       false,
			expectOutput: `test`,
		},
		{
			name:         "override env var",
			args:         []string{"echo", "$HOME"},
			noeval:       false,
			expectOutput: home,
		},
		// This looks very wrong, but is historic behavior
		{
			name:         "override sh echo",
			args:         []string{"sh", "-c", `echo "hello there"`},
			noeval:       false,
			expectOutput: "hello",
		},
		// Docker/OCI behavior (with --no-eval)
		{
			name:         "no-eval/default",
			args:         []string{},
			noeval:       false,
			expectOutput: `CMD 'quotes' "quotes" $DOLLAR s p a c e s`,
		},
		{
			name:         "no-eval/override",
			args:         []string{"echo", "test"},
			noeval:       true,
			expectOutput: `test`,
		},
		{
			name:         "no-eval/override env var",
			noeval:       true,
			args:         []string{"echo", "$HOME"},
			expectOutput: "$HOME",
		},
		{
			name:         "no-eval/override sh echo",
			noeval:       true,
			args:         []string{"sh", "-c", `echo "hello there"`},
			expectOutput: "hello there",
		},
	}

	for _, tt := range tests {
		cmdArgs := []string{}
		if tt.noeval {
			cmdArgs = append(cmdArgs, "--no-eval")
		}
		cmdArgs = append(cmdArgs, imagePath)
		cmdArgs = append(cmdArgs, tt.args...)
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("run"),
			e2e.WithArgs(cmdArgs...),
			e2e.ExpectExit(0,
				e2e.ExpectOutput(e2e.ExactMatch, tt.expectOutput),
			),
		)
	}
}

//nolint:dupl
func (c ctx) testDockerENTRYPOINT(t *testing.T) {
	imageDir, cleanup := e2e.MakeTempDir(t, c.env.TestDir, "docker-", "")
	defer cleanup(t)
	imagePath := filepath.Join(imageDir, "container")

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("while getting $HOME: %s", err)
	}

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("pull"),
		e2e.WithArgs(imagePath, "docker://sylabsio/docker-entrypoint"),
		e2e.ExpectExit(0),
	)

	tests := []struct {
		name         string
		args         []string
		noeval       bool
		expectOutput string
	}{
		// Singularity historic behavior (without --no-eval)
		// These do not all match Docker, due to evaluation, consumption of quoting.
		{
			name:         "default",
			args:         []string{},
			noeval:       false,
			expectOutput: `ENTRYPOINT 'quotes' "quotes" $DOLLAR s p a c e s`,
		},
		{
			name:         "override",
			args:         []string{"echo", "test"},
			noeval:       false,
			expectOutput: `ENTRYPOINT 'quotes' "quotes" $DOLLAR s p a c e s echo test`,
		},
		{
			name:         "override env var",
			args:         []string{"echo", "$HOME"},
			noeval:       false,
			expectOutput: `ENTRYPOINT 'quotes' "quotes" $DOLLAR s p a c e s echo ` + home,
		},
		// Docker/OCI behavior (with --no-eval)
		{
			name:         "no-eval/default",
			args:         []string{},
			noeval:       false,
			expectOutput: `ENTRYPOINT 'quotes' "quotes" $DOLLAR s p a c e s`,
		},
		{
			name:         "no-eval/override",
			args:         []string{"echo", "test"},
			noeval:       true,
			expectOutput: `ENTRYPOINT 'quotes' "quotes" $DOLLAR s p a c e s echo test`,
		},
		{
			name:         "no-eval/override env var",
			noeval:       true,
			args:         []string{"echo", "$HOME"},
			expectOutput: `ENTRYPOINT 'quotes' "quotes" $DOLLAR s p a c e s echo $HOME`,
		},
	}

	for _, tt := range tests {
		cmdArgs := []string{}
		if tt.noeval {
			cmdArgs = append(cmdArgs, "--no-eval")
		}
		cmdArgs = append(cmdArgs, imagePath)
		cmdArgs = append(cmdArgs, tt.args...)
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("run"),
			e2e.WithArgs(cmdArgs...),
			e2e.ExpectExit(0,
				e2e.ExpectOutput(e2e.ExactMatch, tt.expectOutput),
			),
		)
	}
}

//nolint:dupl
func (c ctx) testDockerCMDENTRYPOINT(t *testing.T) {
	imageDir, cleanup := e2e.MakeTempDir(t, c.env.TestDir, "docker-", "")
	defer cleanup(t)
	imagePath := filepath.Join(imageDir, "container")

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("while getting $HOME: %s", err)
	}

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("pull"),
		e2e.WithArgs(imagePath, "docker://sylabsio/docker-cmd-entrypoint"),
		e2e.ExpectExit(0),
	)

	tests := []struct {
		name         string
		args         []string
		noeval       bool
		expectOutput string
	}{
		// Singularity historic behavior (without --no-eval)
		// These do not all match Docker, due to evaluation, consumption of quoting.
		{
			name:         "default",
			args:         []string{},
			noeval:       false,
			expectOutput: `ENTRYPOINT 'quotes' "quotes" $DOLLAR s p a c e s CMD 'quotes' "quotes" $DOLLAR s p a c e s`,
		},
		{
			name:         "override",
			args:         []string{"echo", "test"},
			noeval:       false,
			expectOutput: `ENTRYPOINT 'quotes' "quotes" $DOLLAR s p a c e s echo test`,
		},
		{
			name:         "override env var",
			args:         []string{"echo", "$HOME"},
			noeval:       false,
			expectOutput: `ENTRYPOINT 'quotes' "quotes" $DOLLAR s p a c e s echo ` + home,
		},
		// Docker/OCI behavior (with --no-eval)
		{
			name:         "no-eval/default",
			args:         []string{},
			noeval:       false,
			expectOutput: `ENTRYPOINT 'quotes' "quotes" $DOLLAR s p a c e s CMD 'quotes' "quotes" $DOLLAR s p a c e s`,
		},
		{
			name:         "no-eval/override",
			args:         []string{"echo", "test"},
			noeval:       true,
			expectOutput: `ENTRYPOINT 'quotes' "quotes" $DOLLAR s p a c e s echo test`,
		},
		{
			name:         "no-eval/override env var",
			noeval:       true,
			args:         []string{"echo", "$HOME"},
			expectOutput: `ENTRYPOINT 'quotes' "quotes" $DOLLAR s p a c e s echo $HOME`,
		},
	}

	for _, tt := range tests {
		cmdArgs := []string{}
		if tt.noeval {
			cmdArgs = append(cmdArgs, "--no-eval")
		}
		cmdArgs = append(cmdArgs, imagePath)
		cmdArgs = append(cmdArgs, tt.args...)
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("run"),
			e2e.WithArgs(cmdArgs...),
			e2e.ExpectExit(0,
				e2e.ExpectOutput(e2e.ExactMatch, tt.expectOutput),
			),
		)
	}
}

// https://github.com/sylabs/singularity/issues/233
// This tests quotes in the CMD shell form, not the [ .. ] exec form.
func (c ctx) testDockerCMDQuotes(t *testing.T) {
	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("run"),
		e2e.WithArgs("docker://sylabsio/issue233"),
		e2e.ExpectExit(0,
			e2e.ExpectOutput(e2e.ContainMatch, "Test run"),
		),
	)
}

// E2ETests is the main func to trigger the test suite
func E2ETests(env e2e.TestEnv) testhelper.Tests {
	c := ctx{
		env: env,
	}

	return testhelper.Tests{
		"AUFS":             c.testDockerAUFS,
		"def file":         c.testDockerDefFile,
		"docker host":      c.testDockerHost,
		"permissions":      c.testDockerPermissions,
		"pulls":            c.testDockerPulls,
		"registry":         c.testDockerRegistry,
		"whiteout symlink": c.testDockerWhiteoutSymlink,
		"labels":           c.testDockerLabels,
		"cmd":              c.testDockerCMD,
		"entrypoint":       c.testDockerENTRYPOINT,
		"cmdentrypoint":    c.testDockerCMDENTRYPOINT,
		"cmd quotes":       c.testDockerCMDQuotes,
	}
}
