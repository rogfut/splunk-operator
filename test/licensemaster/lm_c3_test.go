// Copyright (c) 2018-2021 Splunk Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package licensemaster

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	enterprisev1 "github.com/splunk/splunk-operator/pkg/apis/enterprise/v1"
	"github.com/splunk/splunk-operator/test/testenv"
)

var _ = Describe("Licensemaster test", func() {

	var deployment *testenv.Deployment

	BeforeEach(func() {
		var err error
		deployment, err = testenvInstance.NewDeployment(testenv.RandomDNSName(3))
		Expect(err).To(Succeed(), "Unable to create deployment")
	})

	AfterEach(func() {
		// When a test spec failed, skip the teardown so we can troubleshoot.
		if CurrentGinkgoTestDescription().Failed {
			testenvInstance.SkipTeardown = true
		}
		if deployment != nil {
			deployment.Teardown()
		}
	})

	Context("Clustered deployment (C3 - clustered indexer, search head cluster)", func() {
		It("licensemaster, integration: Splunk Operator can configure License Master with Indexers and Search Heads in C3 SVA", func() {

			// Download License File
			licenseFilePath, err := testenv.DownloadLicenseFromS3Bucket()
			Expect(err).To(Succeed(), "Unable to download license file")

			// Create License Config Map
			testenvInstance.CreateLicenseConfigMap(licenseFilePath)

			err = deployment.DeploySingleSiteCluster(deployment.GetName(), 3, true /*shc*/)
			Expect(err).To(Succeed(), "Unable to deploy cluster")

			// Ensure that the cluster-master goes to Ready phase
			testenv.ClusterMasterReady(deployment, testenvInstance)

			// Ensure indexers go to Ready phase
			testenv.SingleSiteIndexersReady(deployment, testenvInstance)

			// Ensure search head cluster go to Ready phase
			testenv.SearchHeadClusterReady(deployment, testenvInstance)

			// Verify MC Pod is Ready
			testenv.MCPodReady(testenvInstance.GetName(), deployment)

			// Verify RF SF is met
			testenv.VerifyRFSFMet(deployment, testenvInstance)

			// Verify LM is configured on indexers
			indexerPodName := fmt.Sprintf(testenv.IndexerPod, deployment.GetName(), 0)
			testenv.VerifyLMConfiguredOnPod(deployment, indexerPodName)
			indexerPodName = fmt.Sprintf(testenv.IndexerPod, deployment.GetName(), 1)
			testenv.VerifyLMConfiguredOnPod(deployment, indexerPodName)
			indexerPodName = fmt.Sprintf(testenv.IndexerPod, deployment.GetName(), 2)
			testenv.VerifyLMConfiguredOnPod(deployment, indexerPodName)

			// Verify LM is configured on SHs
			searchHeadPodName := fmt.Sprintf(testenv.SearchHeadPod, deployment.GetName(), 0)
			testenv.VerifyLMConfiguredOnPod(deployment, searchHeadPodName)
			searchHeadPodName = fmt.Sprintf(testenv.SearchHeadPod, deployment.GetName(), 1)
			testenv.VerifyLMConfiguredOnPod(deployment, searchHeadPodName)
			searchHeadPodName = fmt.Sprintf(testenv.SearchHeadPod, deployment.GetName(), 2)
			testenv.VerifyLMConfiguredOnPod(deployment, searchHeadPodName)
		})
	})

	Context("Clustered deployment (C3 - clustered indexer, search head cluster)", func() {
		It("licensemaster: Splunk Operator can configure a C3 SVA and have apps installed locally on LM", func() {

			// Download License File
			s3TestDir := "c3lm-" + testenv.RandomDNSName(4)
			licenseFilePath, err := testenv.DownloadLicenseFromS3Bucket()
			Expect(err).To(Succeed(), "Unable to download license file")

			// Create License Config Map
			testenvInstance.CreateLicenseConfigMap(licenseFilePath)

			// Create App framework Spec
			// volumeSpec: Volume name, Endpoint, Path and SecretRef
			volumeName := "appframework-test-volume-" + testenv.RandomDNSName(3)
			volumeSpec := []enterprisev1.VolumeSpec{testenv.GenerateIndexVolumeSpec(volumeName, testenv.GetS3Endpoint(), testenvInstance.GetIndexSecretName(), "aws", "s3")}

			// AppSourceDefaultSpec: Remote Storage volume name and Scope of App deployment
			appSourceDefaultSpec := enterprisev1.AppSourceDefaultSpec{
				VolName: volumeName,
				Scope:   "local",
			}

			// appSourceSpec: App source name, location and volume name and scope from appSourceDefaultSpec
			appSourceName := "appframework-" + testenv.RandomDNSName(3)
			appSourceSpec := []enterprisev1.AppSourceSpec{testenv.GenerateAppSourceSpec(appSourceName, s3TestDir, appSourceDefaultSpec)}

			// appFrameworkSpec: AppSource settings, Poll Interval, volumes, appSources on volumes
			appFrameworkSpec := enterprisev1.AppFrameworkSpec{
				Defaults:             appSourceDefaultSpec,
				AppsRepoPollInterval: 60,
				VolList:              volumeSpec,
				AppSources:           appSourceSpec,
			}

			// Deploy the LM with App Framework
			_, err = deployment.DeployLicenseMasterWithGivenSpec(deployment.GetName(), appFrameworkSpec)
			Expect(err).To(Succeed(), "Unable to deploy LM with App framework")

			// Wait for LM to be in READY status
			testenv.LicenseMasterReady(deployment, testenvInstance)

			// Verify apps are copied at the correct location on LM (/etc/apps/)
			podName := []string{fmt.Sprintf(testenv.LicenseMasterPod, deployment.GetName(), 0)}
			testenv.VerifyAppsCopied(deployment, testenvInstance, testenvInstance.GetName(), podName, appListV1, true, false)

			// Verify apps are installed on LM
			lmPodName := []string{fmt.Sprintf(testenv.LicenseMasterPod, deployment.GetName(), 0)}
			testenv.VerifyAppInstalled(deployment, testenvInstance, testenvInstance.GetName(), lmPodName, appListV1, false, "enabled", false, false)
		})
	})
})
