# Slurm pod

The `slurm-pod` is an example of submitting and monitoring a job to an external Slurm Cluster via the `Bridge Operator`.  The code built in (`pods/Dockerfile_slurm`) is intended to be used as the `Pod's` image.

This implementation can serve as a template for job submission to other external systems which can accept HTTP(HTTPS) requests. 
This code was developed to work with the  [Slurm API](https://slurm.schedmd.com/rest_api.html).

## Config Maps

Example ConfigMap:

```
kind: ConfigMap
apiVersion: v1
metadata:
  name: slurmjob-slurm-cm
data:
  # operator poll interval
  updateInterval: "20"                                                            # Poll time
  # job execution
  resourceURL: http://mycluster.ibm.com:6820/slurm/v0.0.36/          # URL for cluster
  resourcesecret: mysecret
  # execution script
  jobdata.jobScript: mybucket:slurmbatch.sh
  jobdata.scriptLocation: s3
  jobdata.jobParameters: |                                                  # parameters
    {
      "NodesNumber":"1", "Queue": "K20", "Tasks": "2", "slurmJobName": "test",
      "ErrorFileName": "slurmjob-sample.err",
      "OutputFileName": "slurmjob-sample.out"
    }
  #S3
  s3.endpoint: minio.endpoint.us-south.containers.appdomain.cloud #S3 endpoint
  s3.secure: "false"                                                        # S3 secure
  s3.secret: mysecret-s3                                                    # S3 secret
```





## Specifics for Slurm

Example body for job submission (slurmtest.txt)
````
{"job":{"partition":"K20","tasks":2,"name":"test","nodes":1,"current_working_directory":"/home/","environment":{"PATH":"/usr/mpi/gcc/bin","LD_LIBRARY_PATH":"/usr/mpi/gcc/"}},"script": "#!/bin/bash\n#SBATCH --job-name=test\n#SBATCH --output=test.out\n#SBATCH --error=test.err\n#SBATCH --nodes=1\n#SBATCH --ntasks=2\n#SBATCH --cpus-per-task=2\n#SBATCH --ntasks-per-node=2\n#SBATCH --partition=K20\n#SBATCH --time=00:05:00\nworkers=100000\necho $PATH\necho $LD_LIBRARY_PATH\nmodule load openmpi4\necho $PATH\necho $LD_LIBRARY_PATH\necho ${workers}\nmpirun -n $SLURM_NTASKS  echo ${PWD}"}
````

### Example curl requests

to submit job

````
curl -v -s -H "Content-Type: application/json" -H X-SLURM-USER-NAME:$USERNAME -H X-SLURM-USER-TOKEN:$SLURM_JWT -X POST http://plex05.watson.ibm.com:6820/slurm/v0.0.36/job/submit --data-binary @slurmtest.txt
````
to query status job id 39
````
curl -v -s -H "Content-Type: application/json" -H X-SLURM-USER-NAME:$USERNAME -H X-SLURM-USER-TOKEN:$SLURM_JWT -X GET http://plex05.watson.ibm.com:6820/slurm/v0.0.36/job/39
````
to delete job 39
````
curl -v -s -H "Content-Type: application/json" -H X-SLURM-USER-NAME:$USERNAME -H X-SLURM-USER-TOKEN:$SLURM_JWT -X DELETE http://plex05.watson.ibm.com:6820/slurm/v0.0.36/job/39
````


## Testing

See `/samples/tutorials/` 
