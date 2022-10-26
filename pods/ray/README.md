# Ray pod

The `ray-pod` allows a user to submit and monitor a job to an external 
[Ray](https://www.ray.io/) cluster via the Bridge Operator.  

The implementation is based on the  [Ray job SDK](https://docs.ray.io/en/latest/ray-job-submission/overview.html#ray-job-sdk),
that provides 3 main APIs:
* Submit_job
* Get_job_status
* Get_job_log
* Stop_job

In order for these APIs to be used, Ray's head node's port `8265` should be accessible either within a same cluster
or via Ingress/Route/LB

The pod assumes that the python script which will be run on the Ray cluster contains the following
parser for parameters in the form of key/value pairs, something like:

````
class ParseKwargs(argparse.Action):
    def __call__(self, parser, namespace, values, option_string=None):
        setattr(namespace, self.dest, dict())
        for value in values:
            key, value = value.split('=')
            getattr(namespace, self.dest)[key] = value

parser = argparse.ArgumentParser()
parser.add_argument('-k', '--kwargs', nargs='*', action=ParseKwargs)
args = parser.parse_args()
````
With this in place, an individual parameter can be accessed via `args.kwargs["<parameter name>"]`.
The parameters that are always submitted to a Ray application are:
* a S3 secret, `s3_secret` and the value of the secret containing S3 credentials
* a S3 bucket, `s3_bucket` and the value of the S3 bucket used for reading/writing data from Ray
* a S3 object, `s3_prefix` and the value of the S3 prefix used for reading/writing data from Ray
* a S3 security flag, `s3_secure` and the flag specifying whether S3 communications are secure (https)

## Security using Ray

At the moment Ray job SDK does not provide any security (all communications are in HTTP with no credentials). This means
that the current implementation should be used only if the Ray cluster:
* Runs in the same Kubernetes cluster
* Runs outside of this kubernetes cluster, but within the the same VPC as the cluster

## ray-pod and S3

Ray APis only support uploading the execution log, which the pod will upload to S3 after the execution is complete.
We are also assuming that a Ray based implementation can directly communicate with S3 and download/upload data.

Example ConfigMap:

```
data:
  jobdata.jobParameters: 'mybucket:ray/parameters.json'
  jobdata.scriptMetadata: 'mybucket:ray/metadata.json'
  resourceURL: 'http://ray-ray-head.ray.svc.cluster.local:8265'
  status.submitTime: '19/04/2022 21:32:40'
  jobdata.jobScript: 'mybucket:ray/code.py'
  s3.endpoint: <S3_URL>
  s3.secure: 'false'
  jobdata.scriptLocation: s3
  s3upload.files: ''
  updateInterval: '20'
  jobproperties: ''
  jobdata.scriptExtraLocation: s3
  status.jobStatus: SUBMITTED
  status.startTime: '19/04/2022 21:32:40'
  s3upload.bucket:  mybucket
  jobdata.additionalData: ''
  id: raysubmit_eFea9na6iKcuzKQk
  resourcesecret: mysecret
  s3.secret: mysecret-s3
```


## Building Docker image

To build an image make sure that you are at the root and then run:
````
export IMAGE_TAG_BASE=<MY_IMAGE_TAG_BASE>
export VERSION=<MY_VERSION>
make docker-build
make docker-push
````

## Testing

See `/samples/tutorials/`.
 
