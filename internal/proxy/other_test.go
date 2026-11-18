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

package proxy

import (
	"testing"
)

func TestUnixSocketDir(t *testing.T) {
	tcs := []struct {
		desc    string
		in      string
		want    string
		wantErr bool
	}{
		{
			desc: "good input",
			in:   "projects/proj/locations/reg/clusters/clust/instances/inst",
			want: "proj.reg.clust.inst",
		},
		{
			desc: "irregular casing",
			in:   "projects/PROJ/locations/REG/clusters/CLUST/instances/INST",
			want: "proj.reg.clust.inst",
		},
		{
			desc: "legacy domain project",
			in:   "projects/google.com:proj/locations/reg/clusters/clust/instances/inst",
			want: "google.com_proj.reg.clust.inst",
		},
		{
			desc: "write endpoint DNS name",
			in:   "ep.global.alloydb-psa.goog",
			want: "ep.global.alloydb-psa.goog",
		},
		{
			desc: "write endpoint DNS name with uppercase and trailing dot",
			in:   "EP.Global.AlloyDB-PSA.Goog.",
			want: "ep.global.alloydb-psa.goog",
		},
		{
			desc:    "invalid DNS name",
			in:      "-invalid..domain",
			wantErr: true,
		},
		{
			in:      "projects/myproj/locations/reg/clusters/clust/instances/",
			wantErr: true,
		},
		{
			in:      "projects/google.com:bad:PROJECT/locations/reg/clusters/clust/instances/inst",
			wantErr: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got, gotErr := UnixSocketDir("", tc.in)
			if tc.wantErr {
				if gotErr == nil {
					t.Fatal("want err != nil, got err == nil")
				}
				return
			}
			if got != tc.want {
				t.Fatalf("want = %v, got = %v", tc.want, got)
			}

		})
	}
}

func TestShortInstURI(t *testing.T) {
	tcs := []struct {
		desc    string
		in      string
		want    string
		wantErr bool
	}{
		{
			desc: "good input",
			in:   "projects/proj/locations/reg/clusters/clust/instances/inst",
			want: "proj.reg.clust.inst",
		},
		{
			desc: "legacy domain project",
			in:   "projects/google.com:proj/locations/reg/clusters/clust/instances/inst",
			want: "google.com:proj.reg.clust.inst",
		},
		{
			desc: "write endpoint DNS name",
			in:   "ep.global.alloydb-psa.goog",
			want: "ep.global.alloydb-psa.goog",
		},
		{
			desc: "write endpoint DNS name with uppercase and trailing dot",
			in:   "EP.Global.AlloyDB-PSA.Goog.",
			want: "ep.global.alloydb-psa.goog",
		},
		{
			desc:    "invalid DNS name",
			in:      "-invalid..domain",
			wantErr: true,
		},
		{
			desc:    "invalid instance URI",
			in:      "projects/myproj/locations/reg/clusters/clust/instances/",
			wantErr: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got, gotErr := ShortInstURI(tc.in)
			if tc.wantErr {
				if gotErr == nil {
					t.Fatal("want err != nil, got err == nil")
				}
				return
			}
			if got != tc.want {
				t.Fatalf("want = %v, got = %v", tc.want, got)
			}
		})
	}
}

func TestToFullURI(t *testing.T) {
	tcs := []struct {
		desc    string
		in      string
		want    string
		wantErr bool
	}{
		{
			desc: "properly formatted short name",
			in:   "myproj.reg.clust.inst",
			want: "projects/myproj/locations/reg/clusters/clust/instances/inst",
		},
		{
			desc: "legacy project name",
			in:   "google.com_myproj.reg.clust.inst",
			want: "projects/google.com:myproj/locations/reg/clusters/clust/instances/inst",
		},
		{
			desc:    "invalid name",
			in:      ".Trash",
			wantErr: true,
		},
		{
			desc:    "full URI",
			in:      "projects/myproj/locations/reg/clusters/clust/instances/inst",
			wantErr: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got, gotErr := toFullURI(tc.in)
			if tc.wantErr {
				if gotErr == nil {
					t.Fatal("want err != nil, got err == nil")
				}
				return
			}
			if got != tc.want {
				t.Fatalf("want = %v, got = %v", tc.want, got)
			}

		})
	}
}

func TestParseInstanceURI(t *testing.T) {
	tcs := []struct {
		desc        string
		in          string
		wantProject string
		wantRegion  string
		wantCluster string
		wantInst    string
		wantErr     bool
	}{
		{
			desc:        "standard instance URI",
			in:          "projects/my-project/locations/us-central1/clusters/my-cluster/instances/my-instance",
			wantProject: "my-project",
			wantRegion:  "us-central1",
			wantCluster: "my-cluster",
			wantInst:    "my-instance",
		},
		{
			desc:        "domain-scoped project",
			in:          "projects/google.com:my-project/locations/us-central1/clusters/my-cluster/instances/my-instance",
			wantProject: "google.com:my-project",
			wantRegion:  "us-central1",
			wantCluster: "my-cluster",
			wantInst:    "my-instance",
		},
		{
			desc:        "uppercase and digits",
			in:          "projects/My-Project-1/locations/europe-west1/clusters/Cluster_1/instances/Inst-2",
			wantProject: "My-Project-1",
			wantRegion:  "europe-west1",
			wantCluster: "Cluster_1",
			wantInst:    "Inst-2",
		},
		{
			desc:    "slash in project",
			in:      "projects/org/my-project/locations/us-central1/clusters/my-cluster/instances/my-instance",
			wantErr: true,
		},
		{
			desc:    "slash in domain-scoped project",
			in:      "projects/google.com:org/my-project/locations/us-central1/clusters/my-cluster/instances/my-instance",
			wantErr: true,
		},
		{
			desc:    "multiple colons in project",
			in:      "projects/a:b:c/locations/us-central1/clusters/my-cluster/instances/my-instance",
			wantErr: true,
		},
		{
			desc:    "missing projects prefix",
			in:      "my-project/locations/us-central1/clusters/my-cluster/instances/my-instance",
			wantErr: true,
		},
		{
			desc:    "extra leading segment",
			in:      "extra/projects/my-project/locations/us-central1/clusters/my-cluster/instances/my-instance",
			wantErr: true,
		},
		{
			desc:    "trailing slash",
			in:      "projects/my-project/locations/us-central1/clusters/my-cluster/instances/my-instance/",
			wantErr: true,
		},
		{
			desc:    "extra trailing segment",
			in:      "projects/my-project/locations/us-central1/clusters/my-cluster/instances/my-instance/extra",
			wantErr: true,
		},
		{
			desc:    "empty project",
			in:      "projects//locations/us-central1/clusters/my-cluster/instances/my-instance",
			wantErr: true,
		},
		{
			desc:    "wrong resource segment",
			in:      "projects/my-project/regions/us-central1/clusters/my-cluster/instances/my-instance",
			wantErr: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			proj, reg, clus, inst, err := ParseInstanceURI(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParseInstanceURI(%q) expected error, got (%q, %q, %q, %q)", tc.in, proj, reg, clus, inst)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseInstanceURI(%q) unexpected error: %v", tc.in, err)
			}
			if proj != tc.wantProject || reg != tc.wantRegion || clus != tc.wantCluster || inst != tc.wantInst {
				t.Fatalf("ParseInstanceURI(%q) = (%q, %q, %q, %q), want (%q, %q, %q, %q)",
					tc.in, proj, reg, clus, inst, tc.wantProject, tc.wantRegion, tc.wantCluster, tc.wantInst)
			}
		})
	}
}
