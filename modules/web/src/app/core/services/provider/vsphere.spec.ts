// Copyright 2026 The Kubermatic Kubernetes Platform contributors.
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

import {HttpClientTestingModule, HttpTestingController} from '@angular/common/http/testing';
import {TestBed} from '@angular/core/testing';
import {VSphereService} from '@core/services/provider/vsphere';
import {VSphereDatastoreUsage} from '@shared/entity/provider/vsphere';

describe('vSphere provider service', () => {
  const projectID = 'project-id';
  const clusterID = 'cluster-id';
  let httpMock: HttpTestingController;
  let service: VSphereService;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [HttpClientTestingModule],
      providers: [VSphereService],
      teardown: {destroyAfterEach: false},
    });
    httpMock = TestBed.inject(HttpTestingController);
    service = TestBed.inject(VSphereService);
  });

  afterEach(() => {
    httpMock.verify();
  });

  describe('#getDatastoreUsage', () => {
    it('should load the datastore usage of the cluster', () => {
      let result: VSphereDatastoreUsage;
      service.getDatastoreUsage(projectID, clusterID).subscribe(usage => (result = usage));

      const req = httpMock.expectOne(r =>
        r.url.endsWith(`/projects/${projectID}/clusters/${clusterID}/providers/vsphere/datastoreusage`)
      );
      expect(req.request.method).toBe('GET');
      req.flush({name: 'datastore-1', capacityBytes: 1000, freeBytes: 250, uncommittedBytes: 50});

      expect(result).toEqual({name: 'datastore-1', capacityBytes: 1000, freeBytes: 250, uncommittedBytes: 50});
    });

    it('should pass on errors of the API', () => {
      let error: unknown;
      service.getDatastoreUsage(projectID, clusterID).subscribe({error: err => (error = err)});

      httpMock
        .expectOne(r => r.url.endsWith('/providers/vsphere/datastoreusage'))
        .flush('forbidden', {status: 403, statusText: 'Forbidden'});

      expect(error).toBeTruthy();
    });
  });
});
