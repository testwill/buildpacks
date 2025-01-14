// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"testing"

	buildpacktest "github.com/GoogleCloudPlatform/buildpacks/internal/buildpacktest"
	"github.com/GoogleCloudPlatform/buildpacks/internal/mockprocess"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/java"
)

func TestDetect(t *testing.T) {
	testCases := []struct {
		name  string
		files map[string]string
		env   []string
		want  int
	}{
		{
			name: "pom.xml",
			files: map[string]string{
				"pom.xml": "",
			},
			want: 0,
		},
		{
			name: ".mvn/extensions.xml",
			files: map[string]string{
				".mvn/extensions.xml": "",
			},
			want: 0,
		},
		{
			name:  "no pom.xml",
			files: map[string]string{},
			want:  100,
		},
		{
			name: "use GOOGLE_BUILDABLE",
			files: map[string]string{
				"testmodule/pom.xml": "",
			},
			env:  []string{"GOOGLE_BUILDABLE=testmodule"},
			want: 0,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buildpacktest.TestDetect(t, detectFn, tc.name, tc.files, tc.env, tc.want)
		})
	}
}

func TestCrLfRewrite(t *testing.T) {
	testCases := []struct {
		name          string
		inputContent  string
		expectContent string
	}{
		{
			name:          "windows-style replaced",
			inputContent:  "#!/bin/sh\r\n\r\necho Windows\r\n",
			expectContent: "#!/bin/sh\n\necho Windows\n",
		},
		{
			name:          "unix-style unmodified",
			inputContent:  "#!/bin/sh\n\necho Unix\n",
			expectContent: "#!/bin/sh\n\necho Unix\n",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp(os.TempDir(), "prefix-")
			if err != nil {
				t.Fatal("Cannot create temporary file", err)
			}
			defer os.Remove(tmpFile.Name())

			_, err = tmpFile.Write([]byte(tc.inputContent))
			if err != nil {
				t.Fatal("Error writing temporary file", err)
			}
			err = tmpFile.Close()
			if err != nil {
				t.Fatal("Error closing temporary file", err)
			}

			ensureUnixLineEndings(gcp.NewContext(), tmpFile.Name())

			newContent, err := ioutil.ReadFile(tmpFile.Name())
			if err != nil {
				t.Fatal("Error reading updated temporary file", err)
			}

			if string(newContent) != tc.expectContent {
				t.Fatal("Unexpected content '%s', want '%s'",
					strconv.QuoteToASCII(string(newContent)),
					strconv.QuoteToASCII(tc.expectContent))
			}

		})
	}
}

func TestBuildCommand(t *testing.T) {
	testCases := []struct {
		name              string
		app               string
		envs              []string
		opts              []buildpacktest.Option
		mocks             []*mockprocess.Mock
		wantCommands      []string
		doNotWantCommands []string
		files             map[string]string
	}{
		{
			name: "maven build argument",
			app:  "hello_quarkus_maven",
			mocks: []*mockprocess.Mock{
				mockprocess.New(`^bash -c command -v mvn || true`, mockprocess.WithStdout("Apache Maven")),
			},
			envs: []string{fmt.Sprintf("%s=clean package", java.MavenBuildArgs)},
			wantCommands: []string{
				"mvn clean package",
			},
			doNotWantCommands: []string{
				"mvn clean package --batch-mode -DskipTests -Dhttp.keepAlive=false",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opts := []buildpacktest.Option{
				buildpacktest.WithTestName(tc.name),
				buildpacktest.WithApp(tc.app),
				buildpacktest.WithEnvs(tc.envs...),
				buildpacktest.WithExecMocks(tc.mocks...),
			}

			opts = append(opts, tc.opts...)
			result, err := buildpacktest.RunBuild(t, buildFn, opts...)
			if err != nil {
				t.Fatalf("error running build: %v, logs: %s", err, result.Output)
			}

			for _, cmd := range tc.wantCommands {
				if !result.CommandExecuted(cmd) {
					t.Errorf("expected command %q to be executed, but it was not, build output: %s", cmd, result.Output)
				}
			}

			for _, cmd := range tc.doNotWantCommands {
				if result.CommandExecuted(cmd) {
					t.Errorf("expected command %q not to be executed, but it was, build output: %s", cmd, result.Output)
				}
			}
		})
	}
}
