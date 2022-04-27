// Copyright 2022 Google LLC
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
package acceptance_test

import (
	"net/http"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/acceptance"
)

func TestGCFAcceptanceGo111(t *testing.T) {
	builder, cleanup := acceptance.CreateBuilder(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			Name: "function without deps",
			App:  "no_deps",
			Env:  []string{"GOOGLE_FUNCTION_TARGET=Func"},
			Path: "/Func",
		},
		{
			Name: "vendored function without dependencies",
			App:  "no_framework_vendored",
			Env:  []string{"GOOGLE_FUNCTION_TARGET=Func"},
			Path: "/Func",
		},
		{
			Name: "function without framework",
			App:  "no_framework",
			Env:  []string{"GOOGLE_FUNCTION_TARGET=Func"},
			Path: "/Func",
		},
		{
			Name: "function with go.sum",
			App:  "no_framework_go_sum",
			Env:  []string{"GOOGLE_FUNCTION_TARGET=Func"},
			Path: "/Func",
		},
		{
			Name:  "vendored function without framework",
			App:   "no_framework",
			Env:   []string{"GOOGLE_FUNCTION_TARGET=Func"},
			Path:  "/Func",
			Setup: vendorSetup,
		},
		{
			Name: "function with old framework",
			App:  "with_framework_old_version",
			Env:  []string{"GOOGLE_FUNCTION_TARGET=Func"},
			Path: "/Func",
		},
		{
			Name:  "vendored function with old framework",
			App:   "with_framework_old_version",
			Env:   []string{"GOOGLE_FUNCTION_TARGET=Func"},
			Path:  "/Func",
			Setup: vendorSetup,
		},
		{
			Name: "function at /*",
			App:  "no_framework",
			Env:  []string{"GOOGLE_FUNCTION_TARGET=Func"},
			Path: "/",
		},
		{
			Name: "function with subdirectories",
			App:  "with_subdir",
			Env:  []string{"GOOGLE_FUNCTION_TARGET=Func"},
		},
		{
			Name:  "set GOPATH incorrectly",
			App:   "no_framework",
			Env:   []string{"GOOGLE_FUNCTION_TARGET=Func", "GOPATH=/tmp"},
			Path:  "/Func",
			Setup: vendorSetup,
		},
		{
			Name: "X_GOOGLE_ENTRY_POINT ignored",
			App:  "invalid_signature",
			// "Func" is the correct name of target function.
			// X_GOOGLE_ENTRY_POINT is irrelevant for function execution (only
			// used for logging an error message when there's an invalid signature).
			Env:                 []string{"GOOGLE_FUNCTION_TARGET=Func"},
			RunEnv:              []string{"X_GOOGLE_ENTRY_POINT=EntryPoint"},
			MustMatchStatusCode: http.StatusInternalServerError,
			MustMatch:           "func EntryPoint is of the type func(http.ResponseWriter, string), expected func(http.ResponseWriter, *http.Request)",
		},
		{
			Name: "X_GOOGLE_WORKER_PORT used over PORT",
			App:  "no_deps",
			Env:  []string{"GOOGLE_FUNCTION_TARGET=Func"},
			// "8080" is the correct port to serve on.
			RunEnv: []string{"PORT=1234", "X_GOOGLE_WORKER_PORT=8080"},
		},
		{
			Name: "user module name without dot in path",
			App:  "no_framework_relative",
			Env:  []string{"GOOGLE_FUNCTION_TARGET=Func"},
		},
	}

	for _, tc := range testCases {
		tc.Env = append(tc.Env, "X_GOOGLE_TARGET_PLATFORM=gcf")
		tc := applyRuntimeVersionTest(t, tc, "1.11")
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			if tc.Setup != nil {
				t.Skip("TODO: The setup functions require go to be pre-installed which is not true for the unified builder")
			}
			tc.FilesMustExist = append(tc.FilesMustExist,
				"/layers/google.utils.archive-source/src/source-code.tar.gz",
				"/workspace/.googlebuild/source-code.tar.gz",
			)
			acceptance.TestApp(t, builder, tc)
		})
	}
}

func TestGCFFailuresGo111(t *testing.T) {
	builder, cleanup := acceptance.CreateBuilder(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.FailureTest{
		{
			App: "with_framework",
			Env: []string{"GOOGLE_FUNCTION_TARGET=Func"},
			// Functions Framework v1.1.0+ supports CloudEvents functions,
			// which requires the cloudevents SDK v2.2.0 which requires Go 1.13+
			MustMatch: "module requires Go 1.13",
		},
	}

	for _, tc := range testCases {
		tc.Env = append(tc.Env, "X_GOOGLE_TARGET_PLATFORM=gcf")
		tc := applyRuntimeVersionFailureTest(t, tc, "1.11")
		t.Run(tc.App, func(t *testing.T) {
			t.Parallel()
			acceptance.TestBuildFailure(t, builder, tc)
		})
	}
}