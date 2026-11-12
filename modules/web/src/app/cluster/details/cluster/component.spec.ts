// Copyright 2020 The Kubermatic Kubernetes Platform contributors.
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

import {HttpClientModule} from '@angular/common/http';
import {ComponentFixture, discardPeriodicTasks, fakeAsync, TestBed, tick, waitForAsync} from '@angular/core/testing';
import {MatDialog, MatDialogRef} from '@angular/material/dialog';
import {BrowserModule, By} from '@angular/platform-browser';
import {NoopAnimationsModule} from '@angular/platform-browser/animations';
import {ActivatedRoute, Router} from '@angular/router';
import {PresetsService} from '@app/core/services/wizard/presets';
import {GoogleAnalyticsService} from '@app/google-analytics.service';
import {AddonService} from '@core/services/addon';
import {ApplicationService} from '@core/services/application';
import {Auth} from '@core/services/auth/service';
import {ClusterService} from '@core/services/cluster';
import {DatacenterService} from '@core/services/datacenter';
import {EndOfLifeService} from '@core/services/eol';
import {MachineDeploymentService} from '@core/services/machine-deployment';
import {MLAService} from '@core/services/mla';
import {NodeService} from '@core/services/node';
import {NotificationService} from '@core/services/notification';
import {OPAService} from '@core/services/opa';
import {ProjectService} from '@core/services/project';
import {RBACService} from '@core/services/rbac';
import {SettingsService} from '@core/services/settings';
import {UserService} from '@core/services/user';
import {SharedModule} from '@shared/module';
import {fakeDigitaloceanCluster} from '@test/data/cluster';
import {nodesFake} from '@test/data/node';
import {AddonServiceMock} from '@test/services/addon-mock';
import {ApplicationServiceMock} from '@test/services/application-mock';
import {AuthMockService} from '@test/services/auth-mock';
import {ClusterMockService} from '@test/services/cluster-mock';
import {DatacenterMockService} from '@test/services/datacenter-mock';
import {MachineDeploymentServiceMock} from '@test/services/machine-deployment-mock';
import {MatDialogMock} from '@test/services/mat-dialog-mock';
import {MatDialogRefMock} from '@test/services/mat-dialog-ref-mock';
import {NodeMockService} from '@test/services/node-mock';
import {PresetServiceMock} from '@test/services/preset-mock';
import {ProjectMockService} from '@test/services/project-mock';
import {ActivatedRouteStub, RouterStub} from '@test/services/router-stubs';
import {SettingsMockService} from '@test/services/settings-mock';
import {UserMockService} from '@test/services/user-mock';
import {VersionPickerComponent} from '../shared/version-picker/component';
import {CNIVersionComponent} from './cni-version/component';
import {ClusterDetailsComponent} from './component';
import {MachineDeploymentListComponent} from './machine-deployment-list/component';
import {MachineNetworksDisplayComponent} from './machine-networks-display/component';
import {MLAComponent} from './mla/component';
import {NodeListComponent} from './node-list/component';
import {RBACComponent} from './rbac/component';
import {FeatureGateService} from '@app/core/services/feature-gate';
import {FeatureGatesMockService} from '@test/services/feature-gate-mock';
import {fakeVSphereCluster} from '@test/data/cluster';
import {VSphereService} from '@core/services/provider/vsphere';
import {VSphereDatastoreUsage} from '@shared/entity/provider/vsphere';
import {NEVER, of, throwError} from 'rxjs';

describe('ClusterDetailsComponent', () => {
  let vsphereMock: {getDatastoreUsage: jest.Mock};

  let fixture: ComponentFixture<ClusterDetailsComponent>;
  let component: ClusterDetailsComponent;
  let activatedRoute: ActivatedRouteStub;

  beforeEach(waitForAsync(() => {
    const rbacMock = {
      getClusterBindings: jest.fn(),
      getBindings: jest.fn(),
      deleteClusterBinding: jest.fn(),
      deleteBinding: jest.fn(),
    };
    rbacMock.deleteClusterBinding.mockReturnValue(of(null));
    rbacMock.deleteBinding.mockReturnValue(of(null));

    vsphereMock = {getDatastoreUsage: jest.fn()};

    const opaMock = {constraints: jest.fn(), gatekeeperConfig: jest.fn()};
    opaMock.constraints.mockReturnValue(of([]));
    opaMock.gatekeeperConfig.mockReturnValue(of(null));

    const mlaMock = {alertmanagerConfig: jest.fn(), ruleGroups: jest.fn()};
    mlaMock.alertmanagerConfig.mockReturnValue(of(null));
    mlaMock.ruleGroups.mockReturnValue(of(null));

    TestBed.configureTestingModule({
      imports: [BrowserModule, HttpClientModule, NoopAnimationsModule, SharedModule],
      declarations: [
        ClusterDetailsComponent,
        NodeListComponent,
        MachineDeploymentListComponent,
        MachineNetworksDisplayComponent,
        VersionPickerComponent,
        RBACComponent,
        CNIVersionComponent,
        MLAComponent,
      ],
      providers: [
        {provide: ClusterService, useClass: ClusterMockService},
        {provide: DatacenterService, useClass: DatacenterMockService},
        {provide: Auth, useClass: AuthMockService},
        {provide: Router, useClass: RouterStub},
        {provide: ActivatedRoute, useClass: ActivatedRouteStub},
        {provide: UserService, useClass: UserMockService},
        {provide: NodeService, useClass: NodeMockService},
        {provide: ProjectService, useClass: ProjectMockService},
        {provide: SettingsService, useClass: SettingsMockService},
        {provide: RBACService, useValue: rbacMock},
        {provide: OPAService, useValue: opaMock},
        {provide: MLAService, useValue: mlaMock},
        {provide: MatDialogRef, useClass: MatDialogRefMock},
        {provide: MatDialog, useClass: MatDialogMock},
        {provide: MachineDeploymentService, useClass: MachineDeploymentServiceMock},
        {provide: AddonService, useClass: AddonServiceMock},
        {provide: ApplicationService, useClass: ApplicationServiceMock},
        {provide: PresetsService, useClass: PresetServiceMock},
        {provide: FeatureGateService, useClass: FeatureGatesMockService},
        {provide: VSphereService, useValue: vsphereMock},
        EndOfLifeService,
        GoogleAnalyticsService,
        NotificationService,
      ],
      teardown: {destroyAfterEach: false},
    }).compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(ClusterDetailsComponent);
    component = fixture.componentInstance;

    activatedRoute = fixture.debugElement.injector.get(ActivatedRoute) as any;
    activatedRoute.testParamMap = {clusterName: '4k6txp5sq'};

    fixture.debugElement.query(By.css('.km-spinner'));
    fixture.debugElement.query(By.css('.cluster-detail-actions'));
  });

  it('should initialize', () => {
    expect(component).toBeTruthy();
  });

  it('should get cluster', fakeAsync(() => {
    fixture.detectChanges();

    const expectedCluster = fakeDigitaloceanCluster();

    tick();
    expect(component.cluster).toMatchObject({
      ...expectedCluster,
      creationTimestamp: expect.any(Date),
    });
  }));

  it('should get nodes', fakeAsync(() => {
    fixture.detectChanges();

    const expectedNodes = nodesFake();

    tick();
    expect(component.nodes).toMatchObject(expectedNodes.map(node => ({...node, creationTimestamp: expect.any(Date)})));
  }));
  /* eslint-disable @typescript-eslint/no-magic-numbers */
  describe('datastore usage', () => {
    const TIB = 1024 ** 4;
    const usage = (usedPercentage: number, capacityBytes = 15 * TIB): VSphereDatastoreUsage => ({
      name: 'datastore-1',
      capacityBytes,
      freeBytes: capacityBytes * (1 - usedPercentage / 100),
      uncommittedBytes: 0,
    });

    function useVSphereCluster(): void {
      (TestBed.inject(ClusterService) as unknown as {_cluster: unknown})._cluster = fakeVSphereCluster();
    }

    it('should not request the datastore usage for clusters of other providers', fakeAsync(() => {
      fixture.detectChanges();
      tick();

      expect(vsphereMock.getDatastoreUsage).not.toHaveBeenCalled();
      expect(component.datastoreUsage).toBeFalsy();
    }));

    it('should show the datastore usage of vSphere clusters', fakeAsync(() => {
      useVSphereCluster();
      vsphereMock.getDatastoreUsage.mockReturnValue(of(usage(80)));

      fixture.detectChanges();
      tick();

      expect(vsphereMock.getDatastoreUsage).toHaveBeenCalledTimes(1);
      expect(component.datastoreUsage.name).toBe('datastore-1');
      expect(component.datastoreUnit).toBe('TiB');
      expect(component.datastoreUsed).toBe(12);
      expect(component.datastoreTotal).toBe(15);
      discardPeriodicTasks();
    }));

    it('should refresh the datastore usage periodically', fakeAsync(() => {
      useVSphereCluster();
      vsphereMock.getDatastoreUsage.mockReturnValue(of(usage(50)));

      fixture.detectChanges();
      tick();
      expect(vsphereMock.getDatastoreUsage).toHaveBeenCalledTimes(1);

      tick(60 * 1000);
      expect(vsphereMock.getDatastoreUsage).toHaveBeenCalledTimes(2);
      discardPeriodicTasks();
    }));

    it('should not show the datastore usage if it cannot be loaded', fakeAsync(() => {
      useVSphereCluster();
      vsphereMock.getDatastoreUsage.mockReturnValue(throwError(() => new Error('vCenter not reachable')));

      fixture.detectChanges();
      tick();

      expect(component.datastoreUsage).toBeNull();
      // the rest of the page is still loaded
      expect(component.sshKeys.length).toBeGreaterThan(0);
      discardPeriodicTasks();
    }));

    it('should not wait for the datastore usage to show the rest of the cluster details', fakeAsync(() => {
      useVSphereCluster();
      vsphereMock.getDatastoreUsage.mockReturnValue(NEVER);

      fixture.detectChanges();
      tick();

      expect(component.sshKeys.length).toBeGreaterThan(0);
      expect(component.datastoreUsage).toBeUndefined();
      discardPeriodicTasks();
    }));

    it.each([
      [0, 'primary'],
      [74.9, 'primary'],
      [75, 'accent'],
      [84.9, 'accent'],
      [85, 'warn'],
      [100, 'warn'],
    ])('should use the color for %s %% used datastore capacity: %s', (usedPercentage, expectedColor) => {
      component.datastoreUsage = usage(usedPercentage);

      expect(component.datastoreColor).toBe(expectedColor);
    });

    it('should show small datastores in GiB', () => {
      component.datastoreUsage = usage(50, 500 * 1024 ** 3);

      expect(component.datastoreUnit).toBe('GiB');
      expect(component.datastoreUsed).toBe(250);
      expect(component.datastoreTotal).toBe(500);
    });
  });
  /* eslint-enable @typescript-eslint/no-magic-numbers */
});
