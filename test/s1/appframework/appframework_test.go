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
package s1appfw

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	testenv "github.com/splunk/splunk-operator/test/testenv"

	enterprisev1 "github.com/splunk/splunk-operator/pkg/apis/enterprise/v1"
	splcommon "github.com/splunk/splunk-operator/pkg/splunk/common"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("s1appfw test", func() {

	var deployment *testenv.Deployment
	var s3TestDir string
	var uploadedApps []string

	BeforeEach(func() {
		var err error
		deployment, err = testenvInstance.NewDeployment(testenv.RandomDNSName(3))
		Expect(err).To(Succeed(), "Unable to create deployment")

		// Upload V1 apps to S3
		s3TestDir = "s1appfw-" + testenv.RandomDNSName(4)
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

	Context("appframework Standalone deployment (S1) with App Framework", func() {
		It("s1, appframework: can deploy a standalone instance with App Framework enabled", func() {

			// Create App framework Spec
			volumeName := "appframework-test-volume-" + testenv.RandomDNSName(3)
			volumeSpec := []enterprisev1.VolumeSpec{testenv.GenerateIndexVolumeSpec(volumeName, testenv.GetS3Endpoint(), testenvInstance.GetIndexSecretName(), "aws", "s3")}

			// AppSourceDefaultSpec: Remote Storage volume name and Scope of App deployment
			appSourceDefaultSpec := enterprisev1.AppSourceDefaultSpec{
				VolName: volumeName,
				Scope:   "local",
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

			spec := enterprisev1.StandaloneSpec{
				CommonSplunkSpec: enterprisev1.CommonSplunkSpec{
					Spec: splcommon.Spec{
						ImagePullPolicy: "Always",
					},
					Volumes: []corev1.Volume{},
				},
				AppFrameworkConfig: appFrameworkSpec,
			}

			// Create Standalone Deployment with App Framework
			standalone, err := deployment.DeployStandalonewithGivenSpec(deployment.GetName(), spec)
			Expect(err).To(Succeed(), "Unable to deploy standalone instance with App framework")

			// Wait for Standalone to be in READY status
			testenv.StandaloneReady(deployment, deployment.GetName(), standalone, testenvInstance)

			// Wait for Standalone to be in READY status
			testenv.MCPodReady(testenvInstance.GetName(), deployment)

			// Verify Apps are downloaded by init-container
			initContDownloadLocation := "/init-apps/" + appSourceName
			podName := fmt.Sprintf(testenv.StandalonePod, deployment.GetName(), 0)
			testenv.VerifyAppsDownloadedByInitContainer(deployment, testenvInstance, testenvInstance.GetName(), []string{podName}, appListV1, initContDownloadLocation)

			//Verify Apps are copied to location
			testenv.VerifyAppsCopied(deployment, testenvInstance, testenvInstance.GetName(), []string{podName}, appListV1, true, true)

			//Verify Apps are installed
			testenv.VerifyAppInstalled(deployment, testenvInstance, testenvInstance.GetName(), []string{podName}, appListV1, true, "enabled", false, false)

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
			testenv.VerifyAppsDownloadedByInitContainer(deployment, testenvInstance, testenvInstance.GetName(), []string{podName}, appListV2, initContDownloadLocation)

			//Verify Apps are copied to location
			testenv.VerifyAppsCopied(deployment, testenvInstance, testenvInstance.GetName(), []string{podName}, appListV2, true, true)

			//Verify Apps are installed
			testenv.VerifyAppInstalled(deployment, testenvInstance, testenvInstance.GetName(), []string{podName}, appListV2, true, "enabled", true, false)

		})
	})
})