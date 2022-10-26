import time
import json
import argparse

from kfp_tekton import TektonClient

def main(
        host: str,
        resource_url: str,
        s3endpoint: str,
        s3uploadbucket: str,
        script: str,
        jobparams: str,
        scriptmd: str,
        scriptlocation: str = 's3',
        scriptextraloc: str = 's3',
        pagesize: int = 50,

):
    client = TektonClient(host = host)

    # List pipelines
    pipelines = client.list_pipelines(page_size=pagesize).pipelines

    pipeline = list(filter(lambda p: "bridge_pipeline" in p.name, pipelines))[0]

    # Get default experiment
    experiment = client.create_experiment('Default')

    # Start pipeline


    params = {
        "jobname": "quantumjob-kfp",
        "namespace": "kubeflow",
        "resourceURL": "https://us-east.quantum-computing.cloud.ibm.com/",
        "resourcesecret": "secret-quantum",
        "script": script,
        "scriptmd": scriptmd,
        "scriptlocation": "s3",
        "scriptextraloc": "s3",
        "s3secret" : "mysecret-s3",
        "s3endpoint": s3endpoint,
        "s3secure": "false",
        "s3uploadbucket": s3uploadbucket,
        "updateinterval": "20",
        "jobparams": jobparams,
        "docker": "quay.io/ibmdpdev/quantum-pod:v0.0.1",
        "arguments": "/quantum-pod",
        "imagepullpolicy": "Always"
}
    runID = client.run_pipeline(experiment_id = experiment.id, job_name = "test_quantum_invoker", pipeline_id = pipeline.id,params = params)
    print("Pipeline submitted")

    status = 'Running'

    while status.lower() not in ['succeeded', 'failed', 'completed', 'skipped', 'error']:
         time.sleep(10)
         run_state = client.get_run(run_id = runID.id)
         status = run_state.run.status

    print(f"Execution complete. Result status is {status}")


if __name__=="__main__":
    parser = argparse.ArgumentParser(description='quantum invoker for KFP Bridge pipeline')
    parser.add_argument('--kfphost',
                    type=str,
                    default='http://localhost:8081/pipeline',
                    help='KFP address')
    parser.add_argument('--resource_url',
                    type=str,
                    default='https://us-east.quantum-computing.cloud.ibm.com/',
                    help='QS address')
    parser.add_argument('--s3endpoint',
                    type=str,
                    default='minio-kubeflow.apps.adp-rosa-2.5wcf.p1.openshiftapps.com',
                    help='s3 endpoint')
    parser.add_argument('--s3uploadbucket',
                    type=str,
                    default='quantum',
                    help='s3 bucket name')
    parser.add_argument('--script',
                    type=str,
                    default='quantum:data/sample_vqe.py',
                    help='python script')
    parser.add_argument('--scriptmd',
                    type=str,
                    default='quantum:data/metadata.json',
                    help='metadata script')
    parser.add_argument('--jobparams',
                    type=str,
                    default='quantum:data/parameters.json',
                    help='parameter script')
    arg = parser.parse_args()
    main(host = arg.kfphost, resource_url=arg.resource_url,
          s3endpoint=arg.s3endpoint, s3uploadbucket=arg.s3uploadbucket,
          script=arg.script, scriptmd=arg.scriptmd, jobparams=arg.jobparams)
