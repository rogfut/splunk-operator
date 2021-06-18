// Copyright (c) 2018-2021 Splunk Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.s
package m4appfw

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	testenv "github.com/splunk/splunk-operator/test/testenv"

	enterprisev1 "github.com/splunk/splunk-operator/pkg/apis/enterprise/v1"
)

var _ = Describe("m4appfw test", func() {

	var deployment *testenv.Deployment
	var s3TestDir string
	var uploadedApps []string

	BeforeEach(func() {
		var err error
		deployment, err = testenvInstance.NewDeployment(testenv.RandomDNSName(3))
		Expect(err).To(Succeed(), "Unable to create deployment")

		// Upload V1 apps to S3
		s3TestDir = "m4appfw-" + testenv.RandomDNSName(4)
		uploadedFiles, err := testenv.UploadFilesToS3(testS3Bucket, s3TestDir, appListV1, downloadDirV1)
		Expect(err).To(Succeed(), "Unable to upload apps to S3 test directory")
		uploadedApps = append(uploadedApps, uploadedFiles...)

	})

	AfterEach(func() {
		// When a test spec failed, skip the teardown so we can troubleshoot.
		if CurrentGinkgoTestDescription().Failed {
			testenvInstance.SkipTeardown = true
		}
		if deployment != nil {
			deployment.Teardown()
		}
		// Delete files uploaded to S3
		if !testenvInstance.SkipTeardown {
			testenv.DeleteFilesOnS3(testS3Bucket, uploadedApps)
		}
	})

	Context("Multi Site Indexer Cluster with SHC (m4) with App Framework", func() {
		It("integration, m4, appframework: can deploy a M4 SVA with App Framework enabled", func() {

			// Create App framework Spec
			volumeName := "appframework-test-volume-" + testenv.RandomDNSName(3)
			volumeSpec := []enterprisev1.VolumeSpec{testenv.GenerateIndexVolumeSpec(volumeName, testenv.GetS3Endpoint(), testenvInstance.GetIndexSecretName(), "aws", "s3")}

			// AppSourceDefaultSpec: Remote Storage volume name and Scope of App deployment
			appSourceDefaultSpec := enterprisev1.AppSourceDefaultSpec{
				VolName: volumeName,
				Scope:   "cluster",
			}

			// appSourceSpec: App source name, location and volume name and scope from appSourceDefaultSpec
			appSourceName := "appframework" + testenv.RandomDNSName(3)
			appSourceSpec := []enterprisev1.AppSourceSpec{testenv.GenerateAppSourceSpec(appSourceName, s3TestDir, appSourceDefaultSpec)}

			// appFrameworkSpec: AppSource settings, Poll Interval, volumes, appSources on volumes
			appFrameworkSpec := enterprisev1.AppFrameworkSpec{
				Defaults:             appSourceDefaultSpec,
				AppsRepoPollInterval: 60,
				VolList:              volumeSpec,
				AppSources:           appSourceSpec,
			}

			siteCount := 3
			indexersPerSite := 1

			err := deployment.DeployMultisiteClusterWithSearchHeadAndAppFramework(deployment.GetName(), indexersPerSite, siteCount, appFrameworkSpec, true)
			Expect(err).To(Succeed(), "Unable to deploy Multi Site Indexer Cluster with App framework")

			// Ensure that the cluster-master goes to Ready phase
			testenv.ClusterMasterReady(deployment, testenvInstance)

			// Ensure the indexers of all sites go to Ready phase
			testenv.IndexersReady(deployment, testenvInstance, siteCount)

			// Ensure cluster configured as multisite
			testenv.IndexerClusterMultisiteStatus(deployment, testenvInstance, siteCount)

			// Ensure search head cluster go to Ready phase
			testenv.SearchHeadClusterReady(deployment, testenvInstance)

			// Verify RF SF is met
			testenv.VerifyRFSFMet(deployment, testenvInstance)

			// Verify Apps are downloaded by init-container
			initContDownloadLocation := "/init-apps/" + appSourceName
			podNames := []string{fmt.Sprintf(testenv.ClusterMasterPod, deployment.GetName()), fmt.Sprintf(testenv.DeployerPod, deployment.GetName())}
			testenv.VerifyAppsDownloadedByInitContainer(deployment, testenvInstance, testenvInstance.GetName(), podNames, appListV1, initContDownloadLocation)

			//Verify Apps are copied to location
			allPodNames := testenv.DumpGetPods(testenvInstance.GetName())
			testenv.VerifyAppsCopied(deployment, testenvInstance, testenvInstance.GetName(), allPodNames, appListV1, true, true)

			//Verify Apps are installed cluster-wide
			testenv.VerifyAppInstalled(deployment, testenvInstance, testenvInstance.GetName(), allPodNames, appListV1, true, "enabled", false, true)

			//Delete apps on S3 for new Apps
			testenv.DeleteFilesOnS3(testS3Bucket, uploadedApps)
			uploadedApps = nil

			//Upload new Versioned Apps to S3
			uploadedFiles, err := testenv.UploadFilesToS3(testS3Bucket, s3TestDir, appListV2, downloadDirV2)
			Expect(err).To(Succeed(), "Unable to upload apps to S3 test directory")
			uploadedApps = append(uploadedApps, uploadedFiles...)

			// Wait for the poll period for the apps to be downloaded
			time.Sleep(2 * time.Minute)

			// Verify Apps are downloaded by init-container
			testenv.VerifyAppsDownloadedByInitContainer(deployment, testenvInstance, testenvInstance.GetName(), podNames, appListV2, initContDownloadLocation)

			//Verify Apps are copied to location
			testenv.VerifyAppsCopied(deployment, testenvInstance, testenvInstance.GetName(), allPodNames, appListV2, true, true)

			//Verify Apps are installed cluster-wide
			testenv.VerifyAppInstalled(deployment, testenvInstance, testenvInstance.GetName(), allPodNames, appListV1, true, "enabled", true, true)
		})
	})

	Context("Clustered deployment (M4 - clustered indexer, search head cluster)", func() {
		It("m4, appframework: can deploy a M4 SVA and have apps installed locally on CM and SHC Deployer", func() {

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

			// Create Multi site Cluster and SHC, with App Framework enabled on CM and SHC Deployer
			siteCount := 3
			indexersPerSite := 1
			err := deployment.DeployMultisiteClusterWithSearchHeadAndAppFramework(deployment.GetName(), indexersPerSite, siteCount, appFrameworkSpec, true)
			Expect(err).To(Succeed(), "Unable to deploy Multi Site Indexer Cluster with App framework")

			// Ensure that the CM goes to Ready phase
			testenv.ClusterMasterReady(deployment, testenvInstance)

			// Ensure the indexers of all sites go to Ready phase
			testenv.IndexersReady(deployment, testenvInstance, siteCount)

			// Ensure SHC go to Ready phase
			testenv.SearchHeadClusterReady(deployment, testenvInstance)

			// Verify apps are copied at the correct location on CM and on Deployer (/etc/apps/)
			podNames := []string{fmt.Sprintf(testenv.ClusterMasterPod, deployment.GetName()), fmt.Sprintf(testenv.DeployerPod, deployment.GetName())}
			testenv.VerifyAppsCopied(deployment, testenvInstance, testenvInstance.GetName(), podNames, appListV1, true, false)

			// Verify apps are installed locally on CM and on SHC Deployer
			testenv.VerifyAppInstalled(deployment, testenvInstance, testenvInstance.GetName(), podNames, appListV1, false, "enabled", false, false)

			// Verify apps are not copied in /etc/master-apps/ on CM and /etc/shcluster/ on Deployer (therefore not installed on peers and on SH)
			testenv.VerifyAppsCopied(deployment, testenvInstance, testenvInstance.GetName(), podNames, appListV1, false, true)
		})
	})
})
