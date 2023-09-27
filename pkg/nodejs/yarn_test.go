// Copyright 2021 Google LLC
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

package nodejs

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/testserver"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/testdata"
	"github.com/google/go-cmp/cmp"
)

func TestUseFrozenLockfile(t *testing.T) {
	testCases := []struct {
		name    string
		version string
		want    bool
	}{
		{
			version: "v10.1.1",
			want:    false,
		},
		{
			version: "v8.17.0",
			want:    false,
		},
		{
			version: "v15.11.0",
			want:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Node.js %s", tc.version), func(t *testing.T) {
			defer func(fn func(*gcpbuildpack.Context) (string, error)) { nodeVersion = fn }(nodeVersion)
			nodeVersion = func(*gcpbuildpack.Context) (string, error) { return tc.version, nil }

			got, err := UseFrozenLockfile(nil)
			if err != nil {
				t.Fatalf("Node.js %v: LockfileFlag(nil) got error: %v", tc.version, err)
			}

			if diff := cmp.Diff(got, tc.want); diff != "" {
				t.Errorf("Node.js %v: LockfileFlag(nil) (+got, -want):\n %v", tc.version, diff)
			}
		})
	}
}

func TestIsYarn2(t *testing.T) {
	testCases := []struct {
		name      string
		content   string
		want      bool
		wantError bool
	}{
		{
			name: "Yarn1 yarn.lock",
			content: `
# THIS IS AN AUTOGENERATED FILE. DO NOT EDIT THIS FILE DIRECTLY.
# yarn lockfile v1

"@arr/every@^1.0.0":
  version "1.0.1"
  resolved "https://registry.yarnpkg.com/@arr/every/-/every-1.0.1.tgz#22fe1f8e6355beca6c7c7bde965eb15cf994387b"
  integrity sha512-UQFQ6SgyJ6LX42W8rHCs8KVc0JS0tzVL9ct4XYedJukskYVWTo49tNiMEK9C2HTyarbNiT/RVIRSY82vH+6sTg==
`,
		},
		{
			name: "Yarn2 yarn.lock",
			content: `
# This file is generated by running "yarn install" inside your project.

__metadata:
  version: 5
  cacheKey: 8

"accepts@npm:~1.3.7":
  version: 1.3.7
  resolution: "accepts@npm:1.3.7"
  dependencies:
    mime-types: ~2.1.24
    negotiator: 0.6.2
  checksum: 27fc8060ffc69481ff6719cd3ee06387d2b88381cb0ce626f087781bbd02201a645a9febc8e7e7333558354b33b1d2f922ad13560be4ec1b7ba9e76fc1c1241d
  languageName: node
  linkType: hard
`,
			want: true,
		},
		{
			name: "Yarn1 invalid YAML",
			content: `
# THIS IS AN AUTOGENERATED FILE. DO NOT EDIT THIS FILE DIRECTLY.
# yarn lockfile v1


"@google-cloud/functions-framework@^1.0.0":
	version "1.5.1"
	resolved "https://registry.yarnpkg.com/@google-cloud/functions-framework/-/functions-framework-1.5.1.tgz#d8b0fd4a0481dcb564a5ab35fac08b6a69b518b1"
	integrity sha512-QvEB0WxP9P+/ykXVM2l41MhPvq0mDKIMoINHPi1Rm4CDuhFEShgOimNXKf/g0ROEyCKVFWO7IfWr4lrVH5CP0Q==
	dependencies:
		body-parser "^1.18.3"
		express "^4.16.4"
		minimist "^1.2.0"
		on-finished "^2.3.0"
`,
			want: false,
		},
		{
			name:      "no yarn.lock",
			wantError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()

			if tc.content != "" {
				fp := path.Join(dir, YarnLock)
				if err := os.WriteFile(fp, []byte(tc.content), 0644); err != nil {
					t.Fatalf("writing %s: %v", fp, err)
				}
			}

			got, err := IsYarn2(dir)

			if tc.wantError && err == nil {
				t.Fatalf("YarnLock(%q) want error but got nil", dir)
			}
			if !tc.wantError && err != nil {
				t.Fatalf("YarnLock(%q) got error: %v", dir, err)
			}
			if got != tc.want {
				t.Errorf("YarnLock(%q) = (%t, %v), want %t", dir, got, err, tc.want)
			}
		})
	}
}

func TestInstallYarn(t *testing.T) {
	testCases := []struct {
		name       string
		version    string
		httpStatus int
		wantFile   string
		wantError  bool
	}{
		{
			name:     "Yarn 1",
			version:  "1.1.1",
			wantFile: "foo.txt",
		},
		{
			name:     "Yarn 2",
			version:  "2.2.2",
			wantFile: "bin/yarn",
		},
		{
			name:       "invalid version",
			version:    "9.9.9",
			httpStatus: http.StatusNotFound,
			wantError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testserver.New(
				t,
				testserver.WithStatus(tc.httpStatus),
				testserver.WithJSON(`yarn!`),
				testserver.WithMockURL(&yarn2URL),
			)

			testserver.New(
				t,
				testserver.WithStatus(tc.httpStatus),
				testserver.WithFile(testdata.MustGetPath("testdata/dummy-yarn.tar.gz")),
				testserver.WithMockURL(&yarnURL),
			)

			dir := t.TempDir()
			err := InstallYarn(nil, dir, tc.version)
			if tc.wantError == (err == nil) {
				t.Fatalf("InstallYarn(nil, %q, %q) got error: %v, want error? %v", dir, tc.version, err, tc.wantError)
			}

			if tc.wantFile != "" {
				fp := filepath.Join(dir, tc.wantFile)
				if _, err := os.Stat(fp); err != nil {
					t.Errorf("Missing file: %s (%v)", fp, err)
				}
			}
		})
	}
}
