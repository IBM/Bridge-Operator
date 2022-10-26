import time
import json
import argparse
from kfp_tekton import TektonClient

def main(
        host: str,
        resource_url: str,
        s3endpoint: str,
        s3_secret: str,
        resource_secret: str,
        pagesize: int = 50,
    ):
    client = TektonClient(host = host)
    # List pipelines
    pipelines = client.list_pipelines(page_size=pagesize).pipelines

    pipeline = list(filter(lambda p: "bridge_pipeline" in p.name, pipelines))[0]

    # Get default experiment
    experiment = client.create_experiment('Default')

    # Start pipeline

    script = \
    """#!/bin/bash
    #SBATCH --job-name=test
    #SBATCH --output=test.out
    module load intelmpi
    echo $PWD
    """

    jobprop = { "NodesNumber":"1", "Queue": "skylake", "Tasks": "1", "slurmJobName": "test",
      "currentWorkingDir": "/home/ec2-user/test/",
      "envPath": "/usr/mpi/gcc/openmpi-4.0.3rc4/bin:/usr/local/bin:/usr/bin:/usr/local/sbin:/usr/sbin:/opt/ibutils/bin",
      "envLibPath": "/usr/mpi/gcc/openmpi-4.0.3rc4/lib",
      "ErrorFileName": "slurmjob-sample.err",
      "OutputFileName": "slurmjob-sample.out"}
    json_str = json.dumps(jobprop)

    params = {
        "jobname": "slurmjob-kfp",
        "namespace": "kubeflow",
        "resourceURL": resource_url,
        "resourcesecret": resource_secret,
        "script": script,
        "scriptlocation": "inline",
        "s3secret" : s3_secret,
        "s3endpoint": s3endpoint,
        "s3secure": "false",
        "updateinterval": "20",
        "jobproperties": json_str,
        "docker": "quay.io/ibmdpdev/slurm-pod:v0.0.1",
        "arguments": "/slurm-pod",
    }

    runID = client.run_pipeline(experiment_id = experiment.id, job_name = "slurm_hpc_invoker", pipeline_id = pipeline.id,
                            params = params)

    print("Pipeline submitted")

    status = 'Running'

    while status.lower() not in ['succeeded', 'failed', 'completed', 'skipped', 'error']:
        time.sleep(10)
        run_state = client.get_run(run_id = runID.id)
        status = run_state.run.status

    print(f"Execution complete. Result status is {status}")


if __name__=="__main__":
    parser = argparse.ArgumentParser(description='slurm invoker for KFP Bridge pipeline')
    parser.add_argument('--kfphost',
                    type=str,
                    default='http://localhost:8081/pipeline',
                    help='KFP address')
    parser.add_argument('--resource_url',
                    type=str,
                    default='http://ec2-3-139-236-142.us-east-2.compute.amazonaws.com:8082/slurm/v0.0.37/',
                    help='slurm cluster address')
    parser.add_argument('--s3_secret',
                    type=str,
                    default='mysecret-s3',
                    help='s3 secret name')
    parser.add_argument('--resource_secret',
                    type=str,
                    default='secret-slurm',
                    help='resource secret name')
    parser.add_argument('--s3endpoint',
                    type=str,
                    default='minio-kubeflow.apps.adp-rosa-2.5wcf.p1.openshiftapps.com',
                    help='s3 endpoint')
    arg = parser.parse_args()
    main(host = arg.kfphost, resource_url=arg.resource_url,
          s3endpoint=arg.s3endpoint, 
          s3_secret=arg.s3_secret, resource_secret=arg.resource_secret)
