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
	"unsafe"
)

func TestClientUsesSyncAtomicAlignment(t *testing.T) {
	// The sync/atomic pkg has a bug that requires the developer to guarantee
	// 64-bit alignment when using 64-bit functions on 32-bit systems.
	c := &Client{}

	if a := unsafe.Offsetof(c.connCount); a%64 != 0 {
		t.Errorf("Client.connCount is not 64-bit aligned: want 0, got %v", a)
	}
}

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
