from kubernetes import client as k8s_client, config

from ray.dashboard.modules.job.sdk import JobSubmissionClient
from ray.dashboard.modules.job.common import JobStatus

from minio import Minio

import sys
import time
import os
import io
import json
from datetime import datetime


CMPREFIX = "-bridge-cm"
NAMESPACE = os.environ['NAMESPACE']
JOBNAME = os.environ['JOBNAME']

POLL = 20
ADDRESS = ''
PIPUPLOADS = {}
ENVIRONMENT = {}
PARAMETERS = {}
S3URL = ""
S3SECRET = ''
S3BUCKET = ''
S3SECURE = False
JOB_ID = ''
RAYCLIENT = None


# upload config map
def upload_config_map() -> k8s_client.V1ConfigMap:
    return api_instance.read_namespaced_config_map(JOBNAME + CMPREFIX, NAMESPACE)

# update config map
def update_config_map(current: dict, cmap: k8s_client.V1ConfigMap):

    updated = False
    for key, value in current.items():
        mvalue = cmap.data.get(key, '')
        if value != mvalue:
            updated = True
            cmap.data[key] = value
            print(f"Updating config map for key {key} changing from {mvalue} to {value}")

    if updated:
        api_instance.patch_namespaced_config_map(JOBNAME + CMPREFIX, NAMESPACE, cmap)

# Submit job
def submit_job() -> str:
    # build entrypoint string
    entrypoint = 'python script.py'
    print(f"building entrypoint {entrypoint}")
    if len(S3SECRET) > 1 or len(PARAMETERS) > 0:
        entrypoint = entrypoint + ' --kwargs '
        if len(S3SECRET) > 1:
            entrypoint = entrypoint + 's3_secret=' + S3SECRET + ', s3_bucket=' + S3BUCKET + ', s3_prefix=' + JOBNAME + '/, s3_secure=' + str(S3SECURE)
        for key, value in PARAMETERS.items():
            entrypoint = f"{entrypoint}, {key}={value}"
# build runtime environment
    runtime = {"working_dir": "/downloads"}
    if len(PIPUPLOADS) > 0:
        pipilist = []
        for key, value in PIPUPLOADS.items():
            pipilist.append(f"{key}=={value}")
        runtime["pip"] = pipilist
    if len(ENVIRONMENT) > 0:
        runtime["env_vars"] = ENVIRONMENT

    print(f"Submitting Ray job, entrypoint {entrypoint} runtime {runtime}")
    job_id = RAYCLIENT.submit_job(
        # Entrypoint shell command to execute
        entrypoint=entrypoint,
        # Working dir
        runtime_env=runtime
    )
    return job_id

# Get S3 client
def get_S3_client() -> Minio:
    try:
        with open('/s3credentials/accesskey', 'r') as secret_file:
            accesskey = secret_file.read()
        with open('/s3credentials/secretkey', 'r') as secret_file:
            secretkey = secret_file.read()
        return Minio(
            endpoint=S3URL,
            access_key=accesskey,
            secret_key=secretkey,
            secure=S3SECURE,
        )
    except Exception as e:
        print(f"Failed to connect to S3, error {e}")
        return None

# Upload log to S3
def upload_job_log(current: dict):
    # get the log
    try:
        logs = RAYCLIENT.get_job_logs(JOB_ID)
    except Exception as e:
        print(f"Failed to get current job log, error {e}")
        current["status.message"] = "Failed to upload log"
        return
    if len(logs) < 1:
        print("Log for current job is empty")
        current["status.message"] = "Log for current job is empty"
        return

    try:
        # Create bucket if it does not exist
        s3client = get_S3_client()
        if not s3client.bucket_exists(S3BUCKET):
            s3client.make_bucket(S3BUCKET)
        # Upload
        s3client.put_object(
            bucket_name=S3BUCKET,
            object_name=f"{JOBNAME}/logs",
            data= io.BytesIO(bytes(logs, 'utf-8')),
            length= len(logs),
        )
        current["status.message"] = "Execution log can be found at S3 location specified"
        return
    except Exception as e:
        print(f"Failed to upload current job log to S3, error {e}")
        current["status.message"] = "Failed to upload log"
        return

# Get object from Minio
def get_s3_object(bucket: str, object: str) ->str:
    resp = None
    try:
        s3client = get_S3_client()
        resp = s3client.get_object(bucket, object)
        return resp.data.decode("utf-8")
    except Exception as e:
        print(f"Failed to load file {bucket}:{object} from S3, error {e}")
        return ""
    finally:
        if resp != None:
            resp.close()
            resp.release_conn()

# Monitor running job
def monitor_job(current: dict):
    while True:
        # Sleep a bit
        time.sleep(POLL)
        # Get current config map
        params = upload_config_map()
        # Get job status
        try:
            status = RAYCLIENT.get_job_status(JOB_ID)
        except Exception as e:
            print(f"Failed to get current job status, error {e} failing the job")
            status = JobStatus.FAILED
        print(f"Monitoring Ray job, status {status}")
        current["status.jobStatus"] = status
        if status in {JobStatus.SUCCEEDED, JobStatus.STOPPED, JobStatus.FAILED}:
            current["status.endTime"] = datetime.now().strftime("%d/%m/%Y %H:%M:%S")
            if len(S3SECRET) > 1:
                # upload results to S3
                upload_job_log(current)
            else:
                current["status.message"] = "Output, intermediate results and log can be found at S3 location specified"
            if status == JobStatus.STOPPED:
                current["status.jobStatus"] = "KILL"
        else:
            if params.data.get('kill', '') == "true":
                try:
                    print("Killing Ray job")
                    RAYCLIENT.stop_job(JOB_ID)
                except Exception as e:
                    print(f"Failed to kill running job, error {e}")
        update_config_map(current, params)

        # Terminate if we are done
        if status == JobStatus.SUCCEEDED:
            sys.exit(0)
        if status in {JobStatus.STOPPED, JobStatus.FAILED}:
            sys.exit(1)

# Load config map
try:
    config.load_incluster_config()
    api_instance = k8s_client.CoreV1Api()
    cmap = upload_config_map()
    params = cmap.data
except Exception as e:
    print(f'Failed to get config map, error {e}')
    sys.exit(1)
if len(params) == 0:
    print('Config map is empty')
    sys.exit(1)
print("Loaded config map")
# Load parameters
JOB_ID = params.get('id','')
POLL = int(params.get('updateInterval'), 20)
ADDRESS = params.get('resourceURL','')
# Add default port (80) if not in address
if not ":" in ADDRESS[7:]:
    ADDRESS = ADDRESS + ":80"
# S3 parameters
S3URL = params.get('s3.endpoint','')
S3SECRET = params.get('s3.secret','')
if params.get('s3.secure','') == "true":
    S3SECURE = True
if len(S3SECRET) > 0:
    if get_S3_client() == None:
        sys.exit(1)
# Script and parameters
script_location = params.get('jobdata.scriptLocation', '')
script_string = params.get('jobdata.jobScript','')
if script_location == 'inline':
    with open("/downloads/script.py", "w") as file:
        file.writelines(script_string)
elif script_location == "s3":
    loc = script_string.split(":")
    with open("/downloads/script.py", "w") as file:
        file.writelines(get_s3_object(loc[0], loc[1]))
else:
    print(f'Unknown script location {script_location}')
    sys.exit(1)
script_extra_location = params.get('jobdata.scriptExtraLocation', '')
metadata_string = params.get('jobdata.scriptMetadata', '')
param_string = params.get('jobdata.jobParameters', '')
if script_extra_location == 'inline':
    metadata = json.loads(metadata_string)
    PARAMETERS = json.loads(param_string)
elif script_extra_location == "s3":
    loc = metadata_string.split(":")
    metadata = json.loads(get_s3_object(loc[0], loc[1]))
    loc = param_string.split(":")
    PARAMETERS = json.loads(get_s3_object(loc[0], loc[1]))
else:
    print(f'Unknown extra parameters location {script_extra_location}')
    sys.exit(1)

pips = metadata.get("pip", {})
if len(pips) > 0:
    PIPUPLOADS = pips
envs = metadata.get("env", {})
if len(envs) > 0:
    ENVIRONMENT = envs

S3BUCKET = params.get('s3upload.bucket','')

# Populate running execution data
current = {}

# Create Ray client
try:
    RAYCLIENT = JobSubmissionClient(ADDRESS)
except Exception as e:
    print(f'Failed to create Ray client, error {e}')
    sys.exit(1)
print("Ray client created")

# see if the job is already running
if len(JOB_ID) < 1:
    # Create job submissition client
    print("Creating a new Ray job")
    # Submit job
    try:
        JOB_ID = submit_job()
    except Exception as e:
        print(f'Failed to submit Ray job, error {e}')
        current["status.jobStatus"] = "FAILED"
        current["status.message"] = "Failed to submit a job to Ray"
        update_config_map(current, cmap)
        sys.exit(1)
    current["id"] = JOB_ID
    current["status.jobStatus"] = "SUBMITTED"
    current["status.startTime"] = datetime.now().strftime("%d/%m/%Y %H:%M:%S")
    current["status.submitTime"] = current["status.startTime"]
    update_config_map(current, cmap)
    monitor_job(current)
else:
    print(f"Continuing with Ray job {JOB_ID}")
    current["id"] = JOB_ID
    monitor_job(current)