/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package vsphere

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/vmware/govmomi/simulator"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
)

const (
	testUsername = "test-user"
	testPassword = "test-password"
)

func TestGetDatastoreSummary(t *testing.T) {
	testcases := []struct {
		name          string
		datastoreName string
		password      string
		// expectedName is the name of the datastore in the returned summary
		expectedName  string
		expectedError string
	}{
		{
			name:          "datastore is found by its inventory path",
			datastoreName: "/DC0/datastore/LocalDS_0",
			expectedName:  "LocalDS_0",
		},
		{
			name:          "datastore is found by its name",
			datastoreName: "LocalDS_0",
			expectedName:  "LocalDS_0",
		},
		{
			name:          "unknown datastore results in an error",
			datastoreName: "/DC0/datastore/does-not-exist",
			expectedError: "couldn't find datastore",
		},
		{
			name:          "wrong password results in an error",
			datastoreName: "/DC0/datastore/LocalDS_0",
			password:      "wrong-password",
			expectedError: "failed to login",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			model := simulator.VPX()

			if err := model.Create(); err != nil {
				t.Fatalf("failed to create the simulated vCenter: %v", err)
			}
			defer model.Remove()

			// The simulator accepts any non-empty login for its default credentials, so use
			// dedicated ones to make it actually verify them.
			model.Service.Listen = &url.URL{User: url.UserPassword(testUsername, testPassword)}

			server := model.Service.NewServer()
			defer server.Close()

			username := testUsername
			password := testPassword
			if tc.password != "" {
				password = tc.password
			}

			dc := &kubermaticv1.DatacenterSpecVSphere{
				Endpoint:   strings.TrimSuffix(server.URL.String(), "/sdk"),
				Datacenter: "DC0",
			}

			summary, err := GetDatastoreSummary(context.Background(), dc, username, password, nil, tc.datastoreName)

			if tc.expectedError != "" {
				if err == nil {
					t.Fatalf("expected an error containing %q, got none", tc.expectedError)
				}
				if !strings.Contains(err.Error(), tc.expectedError) {
					t.Fatalf("expected an error containing %q, got %v", tc.expectedError, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("GetDatastoreSummary failed: %v", err)
			}
			if summary.Name != tc.expectedName {
				t.Errorf("expected datastore %q, got %q", tc.expectedName, summary.Name)
			}
			if summary.Capacity <= 0 {
				t.Errorf("expected a positive capacity, got %d", summary.Capacity)
			}
			if summary.FreeSpace < 0 || summary.FreeSpace > summary.Capacity {
				t.Errorf("expected free space between 0 and the capacity %d, got %d", summary.Capacity, summary.FreeSpace)
			}
		})
	}
}

func TestGetDatastoreClusterSummary(t *testing.T) {
	const (
		capacityBytes = int64(1000)
		freeBytes     = int64(250)
	)

	testcases := []struct {
		name                 string
		datastoreClusterName string
		password             string
		expectedName         string
		expectedError        string
	}{
		{
			name:                 "datastore cluster is found by its inventory path",
			datastoreClusterName: "/DC0/datastore/DC0_POD0",
			expectedName:         "DC0_POD0",
		},
		{
			name:                 "datastore cluster is found by its name",
			datastoreClusterName: "DC0_POD0",
			expectedName:         "DC0_POD0",
		},
		{
			name:                 "unknown datastore cluster results in an error",
			datastoreClusterName: "/DC0/datastore/does-not-exist",
			expectedError:        "couldn't find datastore cluster",
		},
		{
			name:                 "wrong password results in an error",
			datastoreClusterName: "/DC0/datastore/DC0_POD0",
			password:             "wrong-password",
			expectedError:        "failed to login",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			model := simulator.VPX()
			model.Pod = 1

			if err := model.Create(); err != nil {
				t.Fatalf("failed to create the simulated vCenter: %v", err)
			}
			defer model.Remove()

			// The simulator creates datastore clusters without capacity information.
			pod := model.Map().Any("StoragePod").(*simulator.StoragePod)
			pod.Summary.Capacity = capacityBytes
			pod.Summary.FreeSpace = freeBytes

			model.Service.Listen = &url.URL{User: url.UserPassword(testUsername, testPassword)}

			server := model.Service.NewServer()
			defer server.Close()

			password := testPassword
			if tc.password != "" {
				password = tc.password
			}

			dc := &kubermaticv1.DatacenterSpecVSphere{
				Endpoint:   strings.TrimSuffix(server.URL.String(), "/sdk"),
				Datacenter: "DC0",
			}

			summary, err := GetDatastoreClusterSummary(context.Background(), dc, testUsername, password, nil, tc.datastoreClusterName)

			if tc.expectedError != "" {
				if err == nil {
					t.Fatalf("expected an error containing %q, got none", tc.expectedError)
				}
				if !strings.Contains(err.Error(), tc.expectedError) {
					t.Fatalf("expected an error containing %q, got %v", tc.expectedError, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("GetDatastoreClusterSummary failed: %v", err)
			}
			if summary.Name != tc.expectedName {
				t.Errorf("expected datastore cluster %q, got %q", tc.expectedName, summary.Name)
			}
			if summary.Capacity != capacityBytes || summary.FreeSpace != freeBytes {
				t.Errorf("expected capacity %d and free space %d, got %d and %d", capacityBytes, freeBytes, summary.Capacity, summary.FreeSpace)
			}
		})
	}
}
