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

package provider_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/vmware/govmomi/simulator"

	apiv1 "k8c.io/dashboard/v2/pkg/api/v1"
	"k8c.io/dashboard/v2/pkg/handler/test"
	"k8c.io/dashboard/v2/pkg/handler/test/hack"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
)

const (
	vsphereDatacenterName = "vsphere-test"
	vsphereUsername       = "test-user"
	vspherePassword       = "test-password"

	datastoreClusterCapacity = int64(1000)
	datastoreClusterFree     = int64(250)
)

func TestVSphereDatastoreUsageEndpoint(t *testing.T) {
	testcases := []struct {
		name string
		// datastore configured in the cluster spec
		clusterDatastore string
		// datastore cluster configured in the cluster spec
		clusterDatastoreCluster string
		// default datastore configured in the datacenter
		defaultDatastore string
		// password the cluster uses to log in to the vCenter
		password string
		// whether the cluster uses another provider than vSphere
		nonVSphereCluster bool
		expectedStatus    int
		expectedDatastore string
		// expected capacity and free space, only checked if set (datastore clusters have defined values)
		expectedCapacity int64
		expectedFree     int64
	}{
		{
			name:              "the datastore of the cluster is used",
			clusterDatastore:  "/DC0/datastore/LocalDS_0",
			defaultDatastore:  "does-not-matter",
			password:          vspherePassword,
			expectedStatus:    http.StatusOK,
			expectedDatastore: "LocalDS_0",
		},
		{
			name:              "the default datastore of the datacenter is used if the cluster has none",
			defaultDatastore:  "LocalDS_0",
			password:          vspherePassword,
			expectedStatus:    http.StatusOK,
			expectedDatastore: "LocalDS_0",
		},
		{
			name:                    "the datastore cluster of the cluster is used",
			clusterDatastoreCluster: "/DC0/datastore/DC0_POD0",
			defaultDatastore:        "LocalDS_0",
			password:                vspherePassword,
			expectedStatus:          http.StatusOK,
			expectedDatastore:       "DC0_POD0",
			expectedCapacity:        datastoreClusterCapacity,
			expectedFree:            datastoreClusterFree,
		},
		{
			name:                    "unknown datastore cluster results in an error",
			clusterDatastoreCluster: "/DC0/datastore/does-not-exist",
			password:                vspherePassword,
			expectedStatus:          http.StatusInternalServerError,
		},
		{
			name:           "no datastore in cluster and datacenter results in not found",
			password:       vspherePassword,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:             "unknown datastore results in an error",
			clusterDatastore: "/DC0/datastore/does-not-exist",
			password:         vspherePassword,
			expectedStatus:   http.StatusInternalServerError,
		},
		{
			name:             "wrong credentials result in an error",
			clusterDatastore: "/DC0/datastore/LocalDS_0",
			password:         "wrong-password",
			expectedStatus:   http.StatusInternalServerError,
		},
		{
			name:              "cluster of another provider results in not found",
			nonVSphereCluster: true,
			expectedStatus:    http.StatusNotFound,
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
			pod.Summary.Capacity = datastoreClusterCapacity
			pod.Summary.FreeSpace = datastoreClusterFree

			// The simulator accepts any non-empty login for its default credentials, so use
			// dedicated ones to make it actually verify them.
			model.Service.Listen = &url.URL{User: url.UserPassword(vsphereUsername, vspherePassword)}

			server := model.Service.NewServer()
			defer server.Close()

			seed := test.GenTestSeed(func(seed *kubermaticv1.Seed) {
				seed.Spec.Datacenters[vsphereDatacenterName] = kubermaticv1.Datacenter{
					Spec: kubermaticv1.DatacenterSpec{
						VSphere: &kubermaticv1.DatacenterSpecVSphere{
							Endpoint:         strings.TrimSuffix(server.URL.String(), "/sdk"),
							Datacenter:       "DC0",
							DefaultDatastore: tc.defaultDatastore,
						},
					},
				}
			})

			cluster := test.GenCluster(clusterId, clusterName, test.GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC))
			if !tc.nonVSphereCluster {
				cluster.Spec.Cloud = kubermaticv1.CloudSpec{
					DatacenterName: vsphereDatacenterName,
					VSphere: &kubermaticv1.VSphereCloudSpec{
						Username:         vsphereUsername,
						Password:         tc.password,
						Datastore:        tc.clusterDatastore,
						DatastoreCluster: tc.clusterDatastoreCluster,
					},
				}
			}

			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v2/projects/%s/clusters/%s/providers/vsphere/datastoreusage", test.GenDefaultProject().Name, clusterId), strings.NewReader(""))
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(*test.GenDefaultAPIUser(), nil, test.GenDefaultKubermaticObjects(seed, cluster), nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint: %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.expectedStatus {
				t.Fatalf("expected HTTP status code %d, got %d: %s", tc.expectedStatus, res.Code, res.Body.String())
			}
			if res.Code != http.StatusOK {
				return
			}

			var usage apiv1.VSphereDatastoreUsage
			if err := json.Unmarshal(res.Body.Bytes(), &usage); err != nil {
				t.Fatalf("failed to decode the response %q: %v", res.Body.String(), err)
			}
			if usage.Name != tc.expectedDatastore {
				t.Errorf("expected datastore %q, got %q", tc.expectedDatastore, usage.Name)
			}
			if tc.expectedCapacity != 0 && (usage.CapacityBytes != tc.expectedCapacity || usage.FreeBytes != tc.expectedFree) {
				t.Errorf("expected capacity %d and free space %d, got %d and %d", tc.expectedCapacity, tc.expectedFree, usage.CapacityBytes, usage.FreeBytes)
			}
			if usage.CapacityBytes <= 0 {
				t.Errorf("expected a positive capacity, got %d", usage.CapacityBytes)
			}
			if usage.FreeBytes < 0 || usage.FreeBytes > usage.CapacityBytes {
				t.Errorf("expected free space between 0 and the capacity %d, got %d", usage.CapacityBytes, usage.FreeBytes)
			}
		})
	}
}
