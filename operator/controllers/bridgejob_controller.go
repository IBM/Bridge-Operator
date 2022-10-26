/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	e "errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	bridgeoperatorv1alpha1 "github.com/ibm/bridge-operator/api/v1alpha1"
	apiv1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// BridgeJobReconciler reconciles a BridgeJob object
type BridgeJobReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const (
	POD_NAME       = "-bridge-pod"
	CONTAINER_NAME = "-bridge-cont"
	CM_NAME        = "-bridge-cm"
	SA_NAME        = "bridge-cm-viewer"
	ROLE_NAME      = "bridge-cm-role"
	ROLEB_NAME     = "bridge-cm-binding"
	//	PULL_SEC_NAME  = "artifactory"

	PENDING   = "PENDING"
	RUNNING   = "RUNNING"
	DONE      = "DONE"
	FAILED    = "FAILED"
	SUSP      = "SUSPENDED"
	KILL      = "KILL"
	SUCCEEDED = "SUCCEEDED"
	COMPLETED = "COMPLETED"
	UNKNOWN   = "UNKNOWN"

	TIME = "2006-01-02T15:04:05Z"
)

//+kubebuilder:rbac:groups=bridgejob.ibm.com,resources=bridgejobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=bridgejob.ibm.com,resources=bridgejobs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=bridgejob.ibm.com,resources=bridgejobs/finalizers,verbs=update

// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;watch;list
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch;create
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=get;list;watch;create
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the BridgeJob object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *BridgeJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	klog.Info("New reconciliation")

	// Get CR
	var bridgejob bridgeoperatorv1alpha1.BridgeJob

	if err := r.Get(ctx, req.NamespacedName, &bridgejob); err != nil {
		klog.Errorf("Unable to fetch BridgeJob with name %s; namespace %s", req.Name, req.Namespace)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// If we are done - just return
	if bridgejob.Status.JobStatus == DONE || bridgejob.Status.JobStatus == KILL || bridgejob.Status.JobStatus == FAILED || bridgejob.Status.JobStatus == UNKNOWN {
		return ctrl.Result{}, nil
	}

	// Get config map
	cm := &apiv1.ConfigMap{}
	cmErr := r.Get(ctx, types.NamespacedName{Name: bridgejob.Name + CM_NAME, Namespace: bridgejob.Namespace}, cm)

	if cmErr != nil {
		if errors.IsNotFound(cmErr) {
			// Validate the parameters
			S3used := bridgejob.Spec.JobData.ScriptLocation == "s3" || bridgejob.Spec.JobData.ScriptExtraLocation == "s3" ||
				len(bridgejob.Spec.JobData.AdditionalData) > 0 || len(bridgejob.Spec.S3Upload.Bucket) > 0

			if len(bridgejob.Spec.S3Storage.S3Secret) == 0 && S3used {
				// S3 access is not defined but used
				return ctrl.Result{}, r.failCR(ctx, &bridgejob, bridgejob.Name, e.New("S3 access is not defined but used in configuration"))
			}
			// Config map does not exist - create it. Only if we are not done
			// Create definition
			cm, cmErr := r.newConfigMapDefinition(ctx, &bridgejob)
			if cmErr != nil && cm == nil {
				klog.Errorf("Error creating ConfigMap definition; err %s", cmErr.Error())
				return ctrl.Result{}, cmErr
			}
			// Actually create the map
			err := r.Create(ctx, cm)
			if err != nil {
				klog.Errorf("Error creating ConfigMap; err %s", err.Error())
				return ctrl.Result{}, err
			}
			klog.Infof("ConfigMap for BridgeJob %s created.", bridgejob.Name)
		} else {
			// Error getting the map
			return ctrl.Result{}, cmErr
		}
	}

	// Get pod image name
	ptype := getPodType(&bridgejob)

	// Get pod
	pod := &apiv1.Pod{}
	podErr := r.Get(ctx, types.NamespacedName{Name: bridgejob.Name + POD_NAME, Namespace: bridgejob.Namespace}, pod)

	if podErr != nil {
		// Pod does not exist
		if errors.IsNotFound(podErr) {
			// First validate preconditions
			klog.Infoln("Checking Secrets for Pod.")
			err := r.checkCredsSecret(ctx, &bridgejob, bridgejob.Spec.ResourceSecret, "username", "password")
			if err != nil {
				return ctrl.Result{}, err
			}
			// Check for S3 ticket (if used)
			if len(bridgejob.Spec.S3Storage.S3Secret) != 0 {
				err := r.checkCredsSecret(ctx, &bridgejob, bridgejob.Spec.S3Storage.S3Secret, "accesskey", "secretkey")
				if err != nil {
					return ctrl.Result{}, err
				}
			}

			// Check or create RBAC for pod
			klog.Infoln("Checking RBAC for Pod.")
			err = r.checkRBAC(ctx, &bridgejob)
			if err != nil {
				return ctrl.Result{}, err
			}

			// Create pod definition
			pod, podErr := r.newPodDefinition(ctx, &bridgejob)
			if podErr != nil {
				klog.Errorf("Error creating Pod definition; err %s", podErr.Error())
				return ctrl.Result{}, podErr
			}

			// Actually create pod
			err = r.Create(ctx, pod)
			if err != nil {
				klog.Errorf("Error creating Pod; err %s", err.Error())
				return ctrl.Result{}, err
			}
			klog.Infof("Pod for BridgeJob %s created.", bridgejob.Name)
			// Report usage
			podscreated.Inc()
			if ptype != UNKNOWN_POD {
				counters[ptype][POD_CREATED].Inc()
			}

			// Return
			return ctrl.Result{}, nil

		} else {
			return ctrl.Result{}, podErr
		}
	} else {
		// Pod exists, make sure it has not failed
		if pod.Status.Phase == apiv1.PodFailed {
			// Report usage
			podsfailed.Inc()
			if ptype != UNKNOWN_POD {
				counters[ptype][POD_FAILED].Inc()
			}

			// Oops, pod is in a failed state
			jobStatus := cm.Data["status.jobStatus"]
			if jobStatus != KILL && jobStatus != FAILED {
				// Pod failed not because of resource failure or being killed
				msg := fmt.Sprintf("Pod for BridgeJob %s in error state while running, see logs for ", bridgejob.Name)
				klog.Errorf("Pod for BridgeJob %s encountered error while running. Failing CR, see Pod's logs.", bridgejob.Name)
				return ctrl.Result{}, r.failCR(ctx, &bridgejob, bridgejob.Name+POD_NAME, e.New(msg))
			}
		}
	}

	// Check for kill flag
	if bridgejob.Spec.JobKill {
		// CR has a kill flag
		if cm.Data["kill"] != "true" {
			// Report usage
			podskilled.Inc()
			if ptype != UNKNOWN_POD {
				counters[ptype][POD_KILLED].Inc()
			}

			// If the kill flag is not set on config map - set it
			err := r.updateConfigMap(ctx, &bridgejob, bridgejob.Name+CM_NAME, "kill", "true")
			if err != nil {
				klog.Errorf("Updating ConfigMap with kill flag not successful; err %s", err.Error())
				return ctrl.Result{}, err
			}
			klog.Infof("Updating ConfigMap %s with kill flag successful", bridgejob.Name+CM_NAME)
			return ctrl.Result{}, nil
		}
	}

	// Get execution status and update it, if it has changed
	jobStatus := cm.Data["status.jobStatus"]
	updated := updateCondition(&bridgejob, jobStatus, cm)
	if updated {
		err := r.Status().Update(context.Background(), &bridgejob)
		if err != nil {
			klog.Infof("Error updating CR status; msg: %s", err.Error())
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil

}

// Create a new configmap
func (r *BridgeJobReconciler) newConfigMapDefinition(ctx context.Context, bridgejob *bridgeoperatorv1alpha1.BridgeJob) (*apiv1.ConfigMap, error) {

	// Set default for resources
	var cmData = map[string]string{}

	// Main parameters
	cmData["updateInterval"] = strconv.Itoa(bridgejob.Spec.UpdateInterval)
	cmData["resourceURL"] = bridgejob.Spec.ResourceURL
	cmData["jobproperties"] = bridgejob.Spec.JobProperties

	// Job data
	cmData["jobdata.jobScript"] = bridgejob.Spec.JobData.JobScript
	cmData["jobdata.scriptLocation"] = bridgejob.Spec.JobData.ScriptLocation
	cmData["jobdata.scriptMetadata"] = bridgejob.Spec.JobData.ScriptMetadata
	cmData["jobdata.jobParameters"] = bridgejob.Spec.JobData.JobParameters
	cmData["jobdata.scriptExtraLocation"] = bridgejob.Spec.JobData.ScriptExtraLocation
	cmData["jobdata.additionalData"] = bridgejob.Spec.JobData.AdditionalData

	// Set S3, if defined
	if len(bridgejob.Spec.S3Storage.S3Secret) > 0 {
		s3Err := r.addS3Data(ctx, bridgejob, cmData)
		if s3Err != nil {
			return nil, s3Err
		}
	}

	// Upload files
	if len(bridgejob.Spec.S3Upload.Bucket) > 0 {
		cmData["s3upload.bucket"] = bridgejob.Spec.S3Upload.Bucket
		cmData["s3upload.files"] = bridgejob.Spec.S3Upload.Files
	}

	// There is already status
	if len(bridgejob.Status.JobStatus) > 0 {
		cmData["status.startTime"] = bridgejob.Status.StartTime
		cmData["status.endTime"] = bridgejob.Status.CompletionTime
		cmData["status.message"] = bridgejob.Status.Message
	}

	// Define cm map
	cm := &apiv1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      bridgejob.Name + CM_NAME,
			Namespace: bridgejob.Namespace,
		},
		Data: cmData,
	}

	// Set owner reference
	if err := controllerutil.SetControllerReference(bridgejob, cm, r.Scheme); err != nil {
		return nil, err
	}

	return cm, nil
}

// if secret defined , bucket and endpoint must be defined as well
// FAIL whole CR otherwise because we miss data - won't repair by itself
func (r *BridgeJobReconciler) addS3Data(ctx context.Context, bridgejob *bridgeoperatorv1alpha1.BridgeJob, cmData map[string]string) error {

	if len(bridgejob.Spec.S3Storage.Endpoint) < 1 {
		return e.New("bridgejob spec missing S3 endpoint")
	}

	cmData["s3.secret"] = bridgejob.Spec.S3Storage.S3Secret
	cmData["s3.endpoint"] = bridgejob.Spec.S3Storage.Endpoint
	cmData["s3.secure"] = strconv.FormatBool(bridgejob.Spec.S3Storage.Secure)
	return nil
}

// Ensure that required secret exists and formatted properly
func (r *BridgeJobReconciler) checkCredsSecret(ctx context.Context, bridgejob *bridgeoperatorv1alpha1.BridgeJob, secretname, u, p string) error {
	secret := &apiv1.Secret{}
	secretErr := r.Get(ctx, types.NamespacedName{Name: secretname, Namespace: bridgejob.Namespace}, secret)

	if secretErr != nil {
		return r.failCR(ctx, bridgejob, secretname, secretErr)
	} else {
		secErr := checkSecretContent(secret, u, p)
		if secErr != nil {
			return r.failCR(ctx, bridgejob, secretname, secErr)
		}
	}
	return nil
}

// Validate secret content
func checkSecretContent(secret *apiv1.Secret, u, p string) error {
	username := secret.Data[u]
	password := secret.Data[p]
	if len(string(username)) == 0 || len(string(password)) == 0 {
		return fmt.Errorf("secret %s with credentials missing data", secret.Name)
	}
	return nil
}

// Ensure that RBAC for Pod execution exists
func (r *BridgeJobReconciler) checkRBAC(ctx context.Context, bridgejob *bridgeoperatorv1alpha1.BridgeJob) error {
	// Service account
	sa := &apiv1.ServiceAccount{}
	err := r.Get(ctx, types.NamespacedName{Name: SA_NAME, Namespace: bridgejob.Namespace}, sa)
	if err != nil {
		if errors.IsNotFound(err) {
			sa.ObjectMeta = metav1.ObjectMeta{Name: SA_NAME, Namespace: bridgejob.Namespace}
			err = r.Create(ctx, sa)
			if err != nil {
				klog.Errorf("ServiceAccount for Pod not created; err %s", err.Error())
				return err
			}
		} else {
			klog.Errorf("Can not get ServiceAccount for Pod; err %s", err.Error())
			return err
		}
	}

	// Role
	role := &rbacv1.Role{}
	err = r.Get(ctx, types.NamespacedName{Name: ROLE_NAME, Namespace: bridgejob.Namespace}, role)
	if err != nil {
		if errors.IsNotFound(err) {
			role.ObjectMeta = metav1.ObjectMeta{Name: ROLE_NAME, Namespace: bridgejob.Namespace}
			role.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"configmaps"},
					Verbs:     []string{"get", "watch", "list", "update", "patch"},
				},
			}
			err = r.Create(ctx, role)
			if err != nil {
				klog.Errorf("Role %s not created; err %s", ROLE_NAME, err.Error())
				return err
			}
		} else {
			klog.Errorf("Can not get Role for Pod; err %s", err.Error())
			return err
		}
	}

	// Role binding
	roleb := &rbacv1.RoleBinding{}
	err = r.Get(ctx, types.NamespacedName{Name: ROLEB_NAME, Namespace: bridgejob.Namespace}, roleb)
	if err != nil {
		if errors.IsNotFound(err) {
			roleb.ObjectMeta = metav1.ObjectMeta{Name: ROLEB_NAME, Namespace: bridgejob.Namespace}
			roleb.Subjects = []rbacv1.Subject{{Kind: "ServiceAccount", Name: SA_NAME}}
			roleb.RoleRef = rbacv1.RoleRef{Kind: "Role", Name: ROLE_NAME, APIGroup: "rbac.authorization.k8s.io"}
			err = r.Create(ctx, roleb)
			if err != nil {
				klog.Errorf("RoleBinding %s not created; err %s", ROLEB_NAME, err.Error())
				return err
			}
		} else {
			klog.Errorf("Can not get RoleBinding for Pod; err %s", err.Error())
			return err
		}
	}
	return nil
}

// Create a new pod definition
func (r *BridgeJobReconciler) newPodDefinition(ctx context.Context, bridgejob *bridgeoperatorv1alpha1.BridgeJob) (*apiv1.Pod, error) {
	//	autoMount := bool(true)
	pod := &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      bridgejob.Name + POD_NAME,
			Namespace: bridgejob.Namespace,
		},
		Spec: apiv1.PodSpec{
			//			AutomountServiceAccountToken: &autoMount,
			ServiceAccountName: SA_NAME,
			//			ImagePullSecrets: []apiv1.LocalObjectReference{
			//				{
			//					Name: PULL_SEC_NAME,
			//				},
			//			},
			RestartPolicy:  apiv1.RestartPolicyNever,
			InitContainers: []apiv1.Container{},
			Containers: []apiv1.Container{
				{
					Name:            bridgejob.Name + CONTAINER_NAME,
					Image:           bridgejob.Spec.Image,
					ImagePullPolicy: bridgejob.Spec.ImagePullPolicy,
					Resources: apiv1.ResourceRequirements{
						Requests: apiv1.ResourceList{
							apiv1.ResourceCPU:    resource.MustParse("250m"),
							apiv1.ResourceMemory: resource.MustParse("100Mi"),
						},
						Limits: apiv1.ResourceList{
							apiv1.ResourceCPU:    resource.MustParse("500m"),
							apiv1.ResourceMemory: resource.MustParse("100Mi"),
						},
					},
					VolumeMounts: []apiv1.VolumeMount{
						{
							Name:      "credentials",
							MountPath: "/credentials",
							ReadOnly:  true,
						},
					},
					Env: []apiv1.EnvVar{
						{
							Name:  "NAMESPACE",
							Value: bridgejob.Namespace,
						},
						{
							Name:  "JOBNAME",
							Value: bridgejob.Name,
						},
					},
				},
			},
			Volumes: []apiv1.Volume{
				{
					Name: "credentials",
					VolumeSource: apiv1.VolumeSource{
						Secret: &apiv1.SecretVolumeSource{
							SecretName: bridgejob.Spec.ResourceSecret,
						},
					},
				},
			},
		},
	}

	// See if we need to mount S3 credentials
	if len(bridgejob.Spec.S3Storage.S3Secret) != 0 {
		r.mountS3Creds(ctx, bridgejob, pod)
	}

	// Set owner reference
	if err := controllerutil.SetControllerReference(bridgejob, pod, r.Scheme); err != nil {
		return nil, err
	}
	return pod, nil
}

// Mount S3 credentials
func (r *BridgeJobReconciler) mountS3Creds(ctx context.Context, bridgejob *bridgeoperatorv1alpha1.BridgeJob, pod *apiv1.Pod) {

	volume := apiv1.Volume{
		Name: "s3credentials",
		VolumeSource: apiv1.VolumeSource{
			Secret: &apiv1.SecretVolumeSource{
				SecretName: bridgejob.Spec.S3Storage.S3Secret,
			},
		},
	}
	volumeMount := apiv1.VolumeMount{
		Name:      "s3credentials",
		MountPath: "/s3credentials",
		ReadOnly:  true,
	}
	pod.Spec.Volumes = append(pod.Spec.Volumes, volume)
	pod.Spec.Containers[0].VolumeMounts = append(pod.Spec.Containers[0].VolumeMounts, volumeMount)
}

// Update status from cm
func updateCondition(bridgejob *bridgeoperatorv1alpha1.BridgeJob, status string, cm *apiv1.ConfigMap) bool {

	// Check if status has changed
	if len(status) == 0 {
		return false
	}
	if bridgejob.Status.JobStatus == status {
		return false
	}

	// Update status
	bridgejob.Status.JobStatus = status

	if (cm != nil) && (status == DONE || status == KILL || status == FAILED || status == UNKNOWN || status == SUCCEEDED) {
		bridgejob.Status.StartTime = cm.Data["status.submitTime"]
		bridgejob.Status.CompletionTime = cm.Data["status.endTime"]
		bridgejob.Status.Message = cm.Data["status.message"]

		// Get pod type
		ptype := getPodType(bridgejob)

		// Report usage
		if status == DONE || status == SUCCEEDED {

			// Remote job duration (sec)
			start_time, err := time.Parse(TIME, bridgejob.Status.StartTime)
			if err != nil {
				klog.Errorf("Error parsing start time %s; error %s", bridgejob.Status.StartTime, err.Error())
				return true
			}
			end_time, err := time.Parse(TIME, bridgejob.Status.CompletionTime)
			if err != nil {
				klog.Errorf("Error parsing start time %s; error %s", bridgejob.Status.CompletionTime, err.Error())
				return true
			}
			execution := end_time.Sub(start_time).Seconds() / 60.

			podsjobcompleted.Inc()
			podsjobduration.Add(execution)
			if ptype != UNKNOWN_POD {
				counters[ptype][POD_JOBCOMPLETED].Inc()
				gauges[ptype].Add(execution)

			}
		}
		if status == FAILED || status == UNKNOWN {
			podsfailed.Inc()
			if ptype != UNKNOWN_POD {
				counters[ptype][POD_JOBFAILED].Inc()
			}
		}
	}
	return true
}

// Fail CR for kubernetes issues
func (r *BridgeJobReconciler) failCR(ctx context.Context, bridgejob *bridgeoperatorv1alpha1.BridgeJob, objectname string, e error) error {
	_ = updateCondition(bridgejob, FAILED, nil)
	err := r.Status().Update(context.Background(), bridgejob)
	if err != nil {
		klog.Errorf("Error updating CR status; msg: %s", err.Error())
		return err
	}
	msg := fmt.Sprintf("Error in Object %s for job %s, failing BridgeJob; err: %s", objectname, bridgejob.Name, e.Error())
	err = r.updateConfigMap(ctx, bridgejob, objectname, "message", msg)
	if err != nil {
		klog.Errorf("Error updating CM message; msg: %s", err.Error())
		return err
	}
	klog.Infof("Error in Object %s for job %s, failing BridgeJob", objectname, bridgejob.Name)
	return e
}

// Update config map
func (r *BridgeJobReconciler) updateConfigMap(ctx context.Context, bridgejob *bridgeoperatorv1alpha1.BridgeJob, objectname, key, value string) error {
	// Reread cm to ensure its current
	cm := &apiv1.ConfigMap{}
	cmErr := r.Get(ctx, types.NamespacedName{Name: bridgejob.Name + CM_NAME, Namespace: bridgejob.Namespace}, cm)
	if cmErr != nil {
		return cmErr
	}
	// And update it
	cm.Data[key] = value
	err := r.Update(context.Background(), cm)
	if err != nil {
		klog.Errorf("Error updating CM message; msg: %s", err.Error())
		return err
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BridgeJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&bridgeoperatorv1alpha1.BridgeJob{}).
		Owns(&apiv1.Pod{}).
		Owns(&apiv1.ConfigMap{}).
		Complete(r)
}

// Get pod type
func getPodType(bridgejob *bridgeoperatorv1alpha1.BridgeJob) string {

	// Get pod image name
	pod_image := bridgejob.Spec.Image

	if strings.Contains(pod_image, LSF_POD) {
		return LSF_POD
	}
	if strings.Contains(pod_image, SLURM_POD) {
		return SLURM_POD
	}
	if strings.Contains(pod_image, RAY_POD) {
		return RAY_POD
	}
	if strings.Contains(pod_image, QUANTUM_POD) {
		return QUANTUM_POD
	}
	return UNKNOWN_POD
}
