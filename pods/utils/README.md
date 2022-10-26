This is a small utility project for usage in various pods

It implements all the basic support methods that are required for pod implementations.
This methods are:
* InitUtils(job string, ns string) - initialize utility package. Should be called once before all other util methods are used
It also create HTTP and Kubernetes client for use by other methods
* SendReq(req *http.Request) implements logic for sending an HTTP request. Uses HTTP client, created by InitUtils
* GetConfigMap() - reads config map content using Kubernetes client created by InitUtils. The name of the map is based on job name
* UpdateConfigMap(cm *v1.ConfigMap, info map[string]string) - updates current config map with new values and writes it out using 
Kubernetes client created by InitUtils. The name of the map is based on job name
* ReadMountedFileContent(path string) reads mounted file content
* DownloadS3Data(bucket string, object string, data map[string]string) download S3 file from a given bucket/object based on configuration in data
* UploadS3Data(data map[string]string, info map[string]string, objects []UploadFile uploads a set of files to S3. 
Objects is an array of file names and content
