# Bridge Kubeflow Pipeline

The Bridge pipeline is a three component workflow consisting of:

- setup_op: A configmap is created which contains the job parameters.
- invokeop: The relevant Bridge pod (e.g. Slurm, LSF, Ray or Quantum) is created with the resource and S3 credentials mounted.
- cleanup_op: The configmap is deleted.


Specific implementations of running this pipeline for Slurm, LSF, Ray or Quantum are provided in `/implementations` and example tutorials are in `/samples/tutorials`.


# Setup for running locally with Kubeflow on kind

## Install kind with PVC and NGNIX

Following document [here](https://kind.sigs.k8s.io/docs/user/ingress/#ingress-nginx)
````
kind create cluster --config /Users/boris/Projects/kubeflowWithKind/kfpWIthKind/deployment/kind-local.yaml 
````

##Install NGNIX and verify its up:
````
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
kubectl get pods -n ingress-nginx 
````

## Install KFP

If you want to install KFP without tekton use 
[this](https://www.kubeflow.org/docs/components/pipelines/installation/standalone-deployment/#deploying-kubeflow-pipelines), 
otherwise follow install KFP with Tekton

````
export PIPELINE_VERSION=1.8.1
kubectl apply -k "github.com/kubeflow/pipelines/manifests/kustomize/cluster-scoped-resources?ref=$PIPELINE_VERSION"
kubectl wait --for condition=established --timeout=60s crd/applications.app.k8s.io
kubectl apply -k "github.com/kubeflow/pipelines/manifests/kustomize/env/dev?ref=$PIPELINE_VERSION"
````
Once installation is complete change `containerRuntimeExecutor` from `docker` to `k8sapi`

````
kubectl edit configmap workflow-controller-configmap -n kubeflow
````

### OpenShift

In the case of OpenShift, after KFP install, you need to run 2 additional commands:

````
oc adm policy add-scc-to-user anyuid -z application -n kubeflow
oc adm policy add-scc-to-user anyuid -z pipeline-runner -n kubeflow
````

## Install KFP with Tekton

### Install Tekton

For openshift, start with
````
oc new-project tekton-pipelines --display-name='Tekton Pipelines'
oc adm policy add-scc-to-user anyuid -z tekton-pipelines-controller -n tekton-pipelines
oc adm policy add-scc-to-user anyuid -z tekton-pipelines-webhook -n tekton-pipelines
````
Install Tekton

````
kubectl apply --filename https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml
kubectl get pods -n tekton-pipelines
kubectl patch cm feature-flags -n tekton-pipelines \
      -p '{"data":{"enable-custom-tasks": "true", "enable-api-fields": "alpha"}}'
````

### Install KFP 

````
kubectl apply --selector kubeflow/crd-install=true -f https://raw.githubusercontent.com/kubeflow/kfp-tekton/master/install/v1.2.0/kfp-tekton.yaml
kubectl apply  -f https://raw.githubusercontent.com/kubeflow/kfp-tekton/master/install/v1.2.0/kfp-tekton.yaml
kubectl get pods -n kubeflow
````

Make sure that all the pods are running (`kfp-csi-s3-...` failed is ok). For failed pods, delete them (they will restart normally)

KFP pods are running with `serviceAccount: pipeline-runner`, which is using role `pipeline-runner`. This role provides permissions 
to `get watch list` for config map, while we additionally need `create`, `delete` and `update`. We need to patch this role:

For Openshift add

````
oc adm policy add-scc-to-user privileged -z pipeline-runner -n kubeflow
````


## Patch cluster for role creation

````
kubectl apply -f /Users/boris/go/gosrc/hpc-pod/kubeflow/yaml/role-patch.yaml
````

## Enable HTTP access

To enable http access to the kfp run:

````
kubectl apply -f /Users/boris/go/gosrc/hpc-pod/kubeflow/yaml/ingress.yaml
````
now you can access kfp UI go to `localhost:8081`; to access minio go to `localhost:8081/minio`

## Create the pipeline & run a lsf job

```
python /kubeflow/bridge_pipeline.py
```
Upload the yaml file to http://localhost:8081/#/pipelines

Add the secrets to the kubeflow namespace
```
kubectl apply -f test/yaml/secrets/lsfsecret.yaml -n kubeflow
kubectl apply -f test/yaml/secrets/s3secret.yaml -n kubeflow
```
Run the job ```python /kubeflow/implementations/lsf_invoker.py``` and inspect the run at http://localhost:8081/#/runs

Note if running the Ray example we need to apply the following patch first
```kubectl apply -f yaml/patch_serviceaccount.yaml -n kubeflow ```

## Cleanup
````
kind delete cluster
````
