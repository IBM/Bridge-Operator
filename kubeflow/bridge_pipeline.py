import kfp.components as comp
from kfp_tekton.compiler import TektonCompiler
from kfp.compiler import Compiler
import kfp.dsl as dsl
from kubernetes import client as k8s_client

CMPREFIX = "-bridge-cm"

# Create parameters config map
def create_config_map(job_name: str,                # job name
                      namespace: str,               # execution namespace
                      resourceURL: str,             # external resource address - url
                      resourcesecret: str,          # resource credentials
                      script: str,                  # script name or content
                      scriptlocation: str,          # inline, s3 or remote
                      scriptmd: str,                # bucket:file
                      additionaldata: str,          # extra files required
		              scriptextraloc: str,          # s3, inline
		              jobproperties: str,           # dict of job properties
		              jobparams: str,               # dict of job parameters
                      s3secret: str,                # secret with S3 credentials
                      s3endpoint: str,              # S3 URL
                      s3secure: str,                # is S3 secure?
                      s3uploadfiles: str,           # files to upload to S3
                      s3uploadbucket: str,          # bucket in S3
                      updateinterval: str           # resource poll interval
                      ):



    #import
    from kubernetes import client as k8s_client, config

    CMPREFIX = "-bridge-cm"
    config.load_incluster_config()
    api_instance = k8s_client.CoreV1Api()
    cmap = k8s_client.V1ConfigMap()
    cmap.metadata = k8s_client.V1ObjectMeta(name=(job_name + CMPREFIX))
    # populate s3 data
    cmap.data = {}
    # poll
    cmap.data["updateInterval"] = updateinterval
    # HPC cluster
    cmap.data["resourceURL"] = resourceURL
    cmap.data["resourcesecret"] = resourcesecret
    cmap.data["jobproperties"] =  jobproperties
    cmap.data["jobdata.additionalData"] = additionaldata
    cmap.data["jobdata.scriptMetadata"] = scriptmd
    cmap.data["jobdata.jobParameters"] = jobparams
    cmap.data["jobdata.scriptExtraLocation"] = scriptextraloc
    # execution script
    cmap.data["jobdata.jobScript"] = script
    cmap.data["jobdata.scriptLocation"] = scriptlocation
    #S3
    cmap.data["s3.endpoint"] = s3endpoint
    cmap.data["s3.secure"] = s3secure
    cmap.data["s3.secret"] = s3secret
    cmap.data["s3upload.files"] = s3uploadfiles
    cmap.data["s3upload.bucket"] = s3uploadbucket
    # create config map
    api_instance.create_namespaced_config_map(namespace=namespace, body=cmap)

    return

# Delete parameters config map
def delete_config_map(job_name: str,                        # job name
                      namespace: str                        # execution namespace
                      ):
    #import
    from kubernetes import client as k8s_client, config

    CMPREFIX = "-bridge-cm"
    config.load_incluster_config()
    api_instance = k8s_client.CoreV1Api()
    api_instance.delete_namespaced_config_map(name=job_name + CMPREFIX, namespace=namespace)
    return

# components
setup_op = comp.func_to_container_op(
    func=create_config_map,
    packages_to_install=['kubernetes']
)

cleanup_op = comp.func_to_container_op(
    func=delete_config_map,
    packages_to_install=['kubernetes']
)

# Pipeline to invoke execution on remote resource
@dsl.pipeline(
    name='bridge-pipeline',
    description='Pipeline to invoke execution on external resource'
)
def bridge_pipeline(jobname: str,               # job name
                 namespace: str,                # execution namespace
	             resourceURL: str,              # resource address - url
                 resourcesecret: str,           # resource credentials
                 script: str,                   # script name or content
                 scriptlocation: str,           # script location
                 docker: str,                   # docker pod name
                 arguments: str,                # Arguments for docker command
                 scriptmd: str = "",            # script metadata
                 scriptextraloc: str = "",      # location for script extra components
                 additionaldata: str = "",      # extra files required
                 jobproperties: str = "",       # dict of job properties
                 jobparams: str = "",           # dict of job parameters
                 s3secret: str = "",            # secret with S3 credentials
                 s3endpoint: str = "",          # S3 URL
                 s3secure: str = "",            # is S3 secure?
                 s3uploadfiles: str = "",       # files to upload to S3
                 s3uploadbucket: str = "",      # bucket in S3
                 updateinterval: str = "20",    #  poll interval
		         imagepullpolicy: str = "IfNotPresent"
                ):

    createop = setup_op(jobname, namespace, resourceURL, resourcesecret, script, scriptlocation,scriptmd, additionaldata, scriptextraloc, jobproperties, jobparams, \
                    s3secret, s3endpoint, s3secure, s3uploadfiles, s3uploadbucket, updateinterval)
    createop.execution_options.caching_strategy.max_cache_staleness = "P0D"

    invokeop = dsl.ContainerOp(
        name = "bridge-pod",
        image=docker,
        command=["sh", "-c"],
        arguments=[f"{arguments}"]
    ) \
        .add_volume(k8s_client.V1Volume(name='credentials',
                                        secret=k8s_client.V1SecretVolumeSource(secret_name=resourcesecret))) \
        .add_volume_mount(k8s_client.V1VolumeMount(mount_path='/credentials', name='credentials')) \
        .add_env_variable(k8s_client.V1EnvVar(name='NAMESPACE', value=namespace)) \
        .add_env_variable(k8s_client.V1EnvVar(name='JOBNAME', value=jobname)) \
        .after(createop)
    invokeop.container.set_image_pull_policy(imagepullpolicy)
    # Disable caching
    invokeop.execution_options.caching_strategy.max_cache_staleness = "P0D"
    if s3secret != "":
        # Using S3 - mount S3 secret
        invokeop \
            .add_volume(k8s_client.V1Volume(name='s3credentials',
                                            secret=k8s_client.V1SecretVolumeSource(secret_name=s3secret))) \
            .add_volume_mount(k8s_client.V1VolumeMount(mount_path='/s3credentials', name='s3credentials'))
        invokeop \
            .add_volume(k8s_client.V1Volume(name='downloads')) \
            .add_volume_mount(k8s_client.V1VolumeMount(mount_path='/downloads', name='downloads'))

    cleanop = cleanup_op(jobname, namespace).after(invokeop)
    cleanop.execution_options.caching_strategy.max_cache_staleness = "P0D"

if __name__ == '__main__':
    # Compiling the pipeline
#    Compiler().compile(bridge_pipeline, __file__.replace('.py', '.yaml'))

    TektonCompiler().compile(bridge_pipeline, __file__.replace('.py', '.yaml'))
