package podutils

import (
	"bytes"
	"context"
	e "errors"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"k8s.io/klog"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	CM_NAME = "-bridge-cm"
	S3_DIR  = "/s3credentials/"
)

// Global variables
var JOB_NAME string                 // Job name
var NAMESPACE string                // Namespace
var client http.Client              // HTTP client
var clientset *kubernetes.Clientset // Kubernetes clienset

// Upload file definition
type UploadFile struct {
	Name    string // Name
	Content string // content
}

// Upload file location definition
type UploadFileLocation struct {
	Name string // Name
	Path string // local path
}

func InitUtils(job string, ns string) {
	JOB_NAME = job
	NAMESPACE = ns

	// Create kubernets client
	config, err := rest.InClusterConfig()
	if err != nil {
		// Failed to get in cluster configuration
		klog.Exit(err.Error())
	}

	clients, err := kubernetes.NewForConfig(config)
	if err != nil {
		// Failed to create kubernetes client
		klog.Exit(err.Error())
	}
	clientset = clients

	// Create HTTP Client
	client = http.Client{Timeout: time.Duration(10) * time.Second}
}

// Send HTTP request
func SendReq(req *http.Request) ([]byte, int) {
	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		klog.Error("Error invoking HTTP client; error ", err)
		return nil, -1
	}
	defer resp.Body.Close()
	// Process response
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		klog.Error("Error reading HTTP result; error ", err)
		return nil, -1
	}
	return respBody, resp.StatusCode
}

// Get config map
func GetConfigMap() *v1.ConfigMap {
	cm, err := clientset.CoreV1().ConfigMaps(string(NAMESPACE)).Get(context.TODO(), JOB_NAME+CM_NAME, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		klog.Exit("ConfigMap ", NAMESPACE, "/", (JOB_NAME + CM_NAME), " not found.")
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		klog.Exit("Error getting ConfigMap ", NAMESPACE, "/", (JOB_NAME + CM_NAME), "; msg: ", statusError.ErrStatus.Message)
	} else if err != nil {
		klog.Exitln(err.Error())
	}
	return cm
}

// Update config map
func UpdateConfigMap(cm *v1.ConfigMap, info map[string]string) {
	change := false

	// Check if the information changed
	for k, v := range info {
		val := cm.Data[k]
		if val != v {
			klog.Info("Change in ConfigMap, key ", k, " from value ", val, " to value ", v)
			cm.Data[k] = v
			change = true
		}
	}

	// There was a change in info
	if change {
		// Update config map
		_, err := clientset.CoreV1().ConfigMaps(NAMESPACE).Update(context.Background(), cm, metav1.UpdateOptions{})
		if err != nil {
			klog.Error("Updating ConfigMap failed; msg ", err.Error())
		}
		klog.Info("ConfigMap updated.")
	}
}

// Ensure that the S3 bucket exists
func checkBucket(client *minio.Client, bucket string) error {
	found, err := client.BucketExists(context.Background(), bucket)
	if err != nil {
		klog.Infoln("Error checking bucket ", bucket, " err ", err.Error())
		return err
	}
	if !found {
		err = client.MakeBucket(context.Background(), bucket, minio.MakeBucketOptions{ObjectLocking: false})
		if err != nil {
			klog.Error("Error creating bucket ", bucket, " err ", err.Error())
			return err
		}
	}
	return nil
}

// Read from a mounted file
func ReadMountedFileContent(path string) string {
	if _, err := os.Stat(path); e.Is(err, os.ErrNotExist) {
		klog.Exit("Path ", path, " does not exist, exiting")
	}
	content, err := ioutil.ReadFile(path)
	if err != nil {
		klog.Exit("Failed to open path ", path)
	}
	return string(content)
}

// Obtain client for uploading data. We use Minio APIs, which are S3 compatible
func getMinioClient(endpoint string, secure bool) (*minio.Client, error) {
	accessKey := ReadMountedFileContent(S3_DIR + "accesskey")
	secretKey := ReadMountedFileContent(S3_DIR + "secretkey")
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: secure,
	})
	if err != nil {
		return nil, err
	}
	return minioClient, nil
}

// Download data from s3
func DownloadS3Data(bucket string, object string, data map[string]string) string {
	// Create Minio client
	secure, _ := strconv.ParseBool(data["s3.secure"])
	minioClient, err := getMinioClient(data["s3.endpoint"], secure)
	if err != nil {
		klog.Info("Error while getting S3 client; err ", err.Error())
		return ""
	}
	// Get object
	result, err := minioClient.GetObject(context.Background(), bucket, object, minio.GetObjectOptions{})
	if err != nil {
		klog.Info("Error downloading from S3 bucket ", bucket, " object ", object, " ; err ", err.Error())
		return ""
	}
	// Build and return result
	buf := new(bytes.Buffer)
	buf.ReadFrom(result)
	return buf.String()
}

// Download data from s3
func DownloadS3DataDisk(bucket string, object string, directory string, data map[string]string) error {
	// Create Minio client
	secure, _ := strconv.ParseBool(data["s3.secure"])
	minioClient, err := getMinioClient(data["s3.endpoint"], secure)
	if err != nil {
		klog.Info("Error while getting S3 client; err ", err.Error())
		return err
	}
	// Get object
	err = minioClient.FGetObject(context.Background(), bucket, object, directory, minio.GetObjectOptions{})
	if err != nil {
		klog.Info("Error downloading from S3 bucket ", bucket, " object ", object, " ; err ", err.Error())
		return err
	}
	return nil
}

func UploadS3Data(data map[string]string, info map[string]string, objects []UploadFile) {

	// Create client
	bucket := data["s3upload.bucket"]
	secure, _ := strconv.ParseBool(data["s3.secure"])
	minioClient, err := getMinioClient(data["s3.endpoint"], secure)
	if err != nil {
		klog.Info("Error while getting S3 client, not uploading to S3; err ", err.Error())
		info["status.message"] = "Failed to create S3 client. Data is not uploaded to S3"
		return
	}
	// check or create a bucket
	err = checkBucket(minioClient, bucket)
	if err != nil {
		klog.Info("Error while checking or creating bucket, not uploading to S3")
		info["status.message"] = "Failed to create S3 bucket. Data is not uploaded to S3"
		return
	}

	// Upload each object
	for _, object := range objects {
		if len(object.Content) > 0 {
			_, err = minioClient.PutObject(context.Background(), bucket, JOB_NAME+"/"+object.Name,
				strings.NewReader(object.Content), int64(len(object.Content)), minio.PutObjectOptions{})
			if err != nil {
				klog.Info("Error uploading object to S3; err ", err.Error())
			} else {
				klog.Info("Successfuly uploaded object to S3 at ", JOB_NAME+"/"+object.Name, " ", bucket)
			}
		}
	}
}

func UploadS3DataDisk(data map[string]string, objects []UploadFileLocation) error {

	// Create client
	bucket := data["s3upload.bucket"]
	secure, _ := strconv.ParseBool(data["s3.secure"])
	minioClient, err := getMinioClient(data["s3.endpoint"], secure)
	if err != nil {
		klog.Info("Error while getting S3 client, not uploading to S3; err ", err.Error())
		return err
	}
	// check or create a bucket
	err = checkBucket(minioClient, bucket)
	if err != nil {
		klog.Info("Error while checking or creating bucket, not uploading to S3")
		return err
	}

	// Upload each object
	for _, object := range objects {
		_, err = minioClient.FPutObject(context.Background(), bucket, object.Name, object.Path, minio.PutObjectOptions{})
		if err != nil {
			klog.Info("Error uploading object to S3; err ", err.Error())
			return err
		} else {
			klog.Info("Successfuly uploaded object to S3 at ", object.Name, " ", bucket)
		}
	}
	return nil
}
