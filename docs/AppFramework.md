# App Framework Resource Guide

The Splunk Operator supports Splunk Application installations  as configured in the App framework specification of [Custom Resource](https://splunk.github.io/splunk-operator/CustomResources.html). App Framework detects the Application package changes(add/modify/delete) on the remote storage(Ex. S3), and deploys those Apps to the corresponding Splunk Pods.

Note: App names should comply with the Splunk App naming conventions. App names ending with .spl and .tgz are treated as valid App names, and all other files are ignored. App packaging should be as specified in [Package apps for Splunk Cloud or Splunk Enterprise](https://dev.splunk.com/enterprise/docs/releaseapps/packageapps/)

 * App Framework configuration is supported on these Custom Resources: Standalone, ClusterMaster, SearchHeadCluster, and LicenseMaster.
 * App Framework support in the Splunk Operator is limited to Amazon S3 & S3-API-compliant object stores only. 

Configuring the App Framework involves answering the 3 questions:
1. Where the apps are located: Information about how to reach the remote storage, bucket, and the path in the bucket
2. Which CRDs to deploy the apps: Configure the CRD with the information from #1
3. What is the scope of the App installation: Whether the Apps are scoped as local(Ex. Standalone, Deployer, LM, MC, CM) OR scoped as cluster(CM, Deployer) 

App Framework configuration doesn't involve configuring the individual applications. Instead, the remote storage location(s) is specified as appSource(s) under `appRepo:` specification of the CR.  The Splunk Operator will manage(add/modify/delete) all the Apps available in that location(s). 

An App Source requires the folowing config parameters:
1. Logical name of the app source(uniquely identified per CR)
2. Remote storage end point, remote bucket name and path, and the credentials to access the bucket
3. Location(relative path to the remote bucket path)
4. Scope of the App installation(local/cluster wide)

There can be multiple App Sources configured in a CR. These App Sources can be either on the same remote storage bucket, OR spread across multiple remote storage buckets. In order to accommodate this and to avoid duplication, bucket specific configuration is specified as part of Volume config, and referred in the App Source spec. Also, to avoid duplicating the `scope` and `volumeName`, there is a default configuration for App Sources.

`volumes` config entry:
 1. Logical volume name
 2. Remote storage end point
 3. Remote bucket name and path
 4. A reference to Kubernetes secret object
 5. Storage type(Ex. s3) and Provider(Ex. aws, minio)

`appSources` config entry:
1. Logical name of the app source
2. Location(relative path to the remote bucket path)
3. Scope of the App Installation(local/cluster)
4. Reference to volume spec


App Framework configuration involves App Sources, Volumes, and the access credentials. App Sources and volume configurations are configured through the CR. However, the access credentials are configured securely in a Kubernetes secret object, and that secret object is referred by the Custome Resource with volume spec, through `SecretRef`

## Storing Secrets In Kubernetes Secret Object
Here is an example command to encode and load your remote storage volume secret key and access key in the kubernetes secret object: `kubectl create secret generic <secret_store_obj> --from-literal='s3_access_key=<access_key>' --from-literal='s3_secret_key=<secret_key>'`


## Creating a App Framework enabled Standalone instance
1. Create a Kubernetes Secret Object with credentials, as explained in [Storing Secrets In Kubernetes Secret Object](#Storing-Secrets-In-Kubernetes-Secret-Object)
2. Confirm your S3-based storage volume path and URL.
3. Confirm the App Source locations.
4. Copy the Splunk Application Packages to the App Source location(s).
5. Create/Update the Standalone Customer Resource specification with volume and App Source configuration (see Example below).
6. Apply the Customer Resource specification: `kubectl -f apply Standalone.yaml`

Example. Standalone.yaml:

```yaml
apiVersion: enterprise.splunk.com/v1
kind: Standalone
metadata:
  name: <example name>
  finalizers:
  - enterprise.splunk.com/delete-pvc
spec:
  replicas: 1
  appRepo:
    appsRepoPollIntervalSeconds: <#(seconds)>
    defaults:
      volumeName: <remote_volume_name_1>
      scope: local
    volumes:
      - name: <remote_volume_name_1>
        path: <remote_volume_path_1>
        endpoint: https://s3-<region>.amazonaws.com
        secretRef: <K8_secret_obj_name_1>
        provider: aws
        storageType: s3
      - name: <remote_volume_name_2>
        path: <remote_volume_path_2> 
        endpoint: https://s3-<region>.amazonaws.com
        secretRef: <K8_secret_obj_name_2>
        provider: aws
        storageType: s3
    AppSources:
      - name: <appSrc_name_1>
        location: <remote storage location relative to the volume path>
      - name: <appSrc_name_2>
        location: <remote storage location relative to the volume path>
        volumeName: <remote_volume_name_1>
        scope: local 
      - name: <appSrc_name_3>
        location: <remote storage location relative to the volume path>
        volumeName: <remote_volume_name_1>
```

## Spec field details:
    appRepo:
      volumes:
        - name: <Logical volume name>
          path: <Remote storage bucket path: i.e bucket_name/[path]>
          endpoint: <URI to reach the remote storage: Ex. https://s3-<region>.amazonaws.com>
          secretRef: <K8 secret object name in which the credentials for accessing the remote storage are stored>
          storageType: <Remote storage type. Ex: s3>
          provider: <Remote storage provider: Ex. aws>

      AppSources:
        - name: <Logical name of the App source>
          volumeName: <Logical valume name as specified in volumes: spec>
          location: <App source location relative to the Volume path(as specified in volumes section). Ex: AppPath/NetowrkingApps/>
          scope: <`local`: Apps should be installed to the Splunk instance of this CR Kind. 
                  `cluster`: Apps should be installed to the Splunk instance cluster members managed by this CR Kind. Applicable only to CM and SearchHeadCluster CRs>     

      appsRepoPollIntervalSeconds: <Time interval at which Operator has to probe the appSource location for any App related changes>


## App Framework Spec Parameters
The App Framework config applies to the `Cluster Manager`, `SearchHeadCluster`, `Standalone`,  and `LicenseMaster` CRDs, and adds the following `Spec` parameters:

```
              appRepo:
                description: Splunk Enterprise App repository. Specifies remote App
                  location and scope for Splunk App management
                properties:
                  appSources:
                    description: List of App sources on remote storage
                    items:
                      description: AppSourceSpec defines list of App package (*.spl,
                        *.tgz) locations on remote volumes
                      properties:
                        location:
                          description: Location relative to the volume path
                          type: string
                        name:
                          description: Logical name for the set of apps placed in
                            this location. Logical name must be unique to the appRepo
                          type: string
                        scope:
                          description: 'Scope of the App deployment: cluster, local.
                            Scope determines whether the App(s) is/are installed locally
                            or cluster-wide'
                          type: string
                        volumeName:
                          description: Remote Storage Volume name
                          type: string
                      type: object
                    type: array
                  appsRepoPollIntervalSeconds:
                    description: Interval in seconds to check the Remote Storage for
                      App changes
                    type: integer
                  defaults:
                    description: Defines the default configuration settings for App
                      sources
                    properties:
                      scope:
                        description: 'Scope of the App deployment: cluster, local.
                          Scope determines whether the App(s) is/are installed locally
                          or cluster-wide'
                        type: string
                      volumeName:
                        description: Remote Storage Volume name
                        type: string
                    type: object
                  volumes:
                    description: List of remote storage volumes
                    items:
                      description: VolumeSpec defines remote volume config
                      properties:
                        endpoint:
                          description: Remote volume URI
                          type: string
                        name:
                          description: Remote volume name
                          type: string
                        path:
                          description: Remote volume path
                          type: string
                        provider:
                          description: 'App Package Remote Store provider. For e.g.
                            aws, azure, minio, etc. Currently we are only supporting
                            aws. TODO: Support minio as well.'
                          type: string
                        secretRef:
                          description: Secret object name
                          type: string
                        storageType:
                          description: Remote Storage type.
                          type: string
                      type: object
                    type: array
                type: object
```

## Following is the table with the CRD and scope details 

| CRD Type | App Framework Config| Allowed Scopes |
| :--- | :--- | :--- |
| ClusterManager | Allowed | cluster, local |
| IndexerCluster | Not-Allowed | N/A |
| SearcHeadCluster | Allowed | cluster, local |
| Standalone | Allowed | local |
| LicenceMaster | Allowed  | local |

## Limitations