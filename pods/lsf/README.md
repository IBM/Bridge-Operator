# LSF pod

The `lsf-pod` is an example of submitting and monitoring a job to an external system [IBM Spectrum LSF workload management package](https://www.ibm.com/products/hpc-workload-management) via the `Bridge Operator`.  The code built in (`pods/Dockerfile_lsf `) is intended to be used as the `Pod's` image. 

This implementation can serve as a template for job submission to other external systems which can accept HTTP(HTTPS) requests. 

## Information required by the Pod:
- environment variables
  - `NAMESPACE` the namespace where the `BridgeJob` was deployed
  - `JOBNAME` the `BridgeJob`'s name
- `ConfigMap`
  - includes:
    - information from `BridgeJob` (address of REST API, resource specification, etc.)
    - S3 information (`Secret` with credentials, other information for connection) if it was specified in `BridgeJob`

## Information gathered by the Pod:
  - `jobStatus` as reported by the workload manager on the external system.
    - **REQUIRED**
    - The Operator updates the `HPCJob Status`
    - The `HPCJob` is completed if `DONE/FAILED/KILLED/UNKNOWN`
  - `submitTime`
    - **OPTIONAL**
    - upon job completion, the Operator reads this and updates `hpcjob.Status.StartTime` 
  - `endTime`
     - **OPTIONAL**
    - upon job completion, the Operator reads this and updates `hpcjob.Status.CompletionTime` 
  - `message`
    - **OPTIONAL BUT HIGHLY RECOMMENDED**
    - upon job completion, the Operator reads this and updates `hpcjob.Status.Message`
    - it can provide valuable insight for the user such as location of output files, reason for failure etc.

Example ConfigMap:

```
kind: ConfigMap
apiVersion: v1
metadata:
  name: hpcjob-bridge-cm
data:
  resourceURL: http://mycluster.ibm.com:8080/platform/
  resourcesecret: mysecret
  imagepullpolicy: Always
  updateinterval: "20"
  jobdata.jobScript: "/home/batch.sh"
  jobdata.scriptLocation: remote

  jobproperties: |
    {"NodesNumber": "1", "Queue": "excl", "RunLimitHour": "1", "RunLimitMinute": "0", 
     "ErrorFileName": "sample.err", "OutputFileName": "sample.out",
     "inputfiledirectory": "/home/", "pastid": "497906"
    }
  #Download files from S3 and upload to cluster, inputfiledirectory and pastid must be in jobproperties
  jobdata.scriptExtraLocation: "s3"
  jobdata.additionalData: mybucket:test.txt
  #S3
  s3.endpoint: minio.endpoint.us-south.containers.appdomain.cloud #S3 endpoint
  s3.secure: "false"                                                        # S3 secure
  s3.secret: mysecret-s3  
```




## Testing

See `/samples/tutorials`. 
