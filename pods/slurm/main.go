//=============================================================================
// Code for managing HPC jobs deployed on SLURM
//=============================================================================

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ibm/bridge-operator/podutils"

	"k8s.io/klog"
)

const (
	MULTIPLE_ACCEPT_TYPE = "text/plain,application/xml,text/xml,multipart/mixed"
	TIME                 = "2006-01-02T15:04:05Z"

	SUBMITTED  = "SUBMITTED"
	PENDING    = "PENDING"
	RUNNING    = "RUNNING"
	COMPLETED  = "COMPLETED"
	COMPLETING = "COMPLETING"
	CANCELLED  = "CANCELLED"
	UNKNOWN    = "UNKNOWN"
	FAILED     = "FAILED"

	CREDS_DIR  = "/credentials/"
	SCRIPT_DIR = "/script/script"
	FILES_DIR  = "/downloads/"

	TOKEN_SLEEP = 3
)

var HPCURL string
var JOB_NAME string
var NAMESPACE string
var POLL int
var S3 string
var UPLOAD string
var DOWNLOAD string
var JobMap map[string]interface{}
var JobProp map[string]string

// HPC job resource definitions
var RESOURCES = map[string]string{
	"RunLimitHour":   "RUNLIMITHOUR",
	"RunLimitMinute": "RUNLIMITMINUTE",
	"Queue":          "QUEUE",
	"OutputFileName": "OUTPUT_FILE",
	"ErrorFileName":  "ERROR_FILE",
}

// Job ID
type JobId struct {
	Id int `json:"job_id"`
}

// HPC Job info
type JobInfo struct {
	Job []map[string]interface{} `json:"jobs"`
}

// Gets detailed job information for jobs that have the specified job IDs.
// If job is not return by call to all jobs, returns 404
func getJobInfo(slurmUsername string, slurmToken string, id string) *JobInfo {
	url := HPCURL + "/job/" + id
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		klog.Error("Error creating Job Info request; err ", err)
		return nil
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-SLURM-USER-NAME", slurmUsername)
	req.Header.Set("X-SLURM-USER-TOKEN", slurmToken)

	respBody, statusCode := podutils.SendReq(req)
	if statusCode != 200 {
		klog.Error("Retrieving job info not successful, status code ", statusCode)
		return nil
	}

	job := JobInfo{}
	err = json.Unmarshal(respBody, &job)
	JobMap = job.Job[0]
	return &job
}

func checkSlurmToken(slurmUsername string, slurmToken string) {
	url := HPCURL + "/ping"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		klog.Error("Error creating ping request; err ", err)
		os.Exit(1)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-SLURM-USER-NAME", slurmUsername)
	req.Header.Set("X-SLURM-USER-TOKEN", slurmToken)

	_, statusCode := podutils.SendReq(req)
	if statusCode != 200 {
		klog.Error("Ping to HPC cluster not successful, check SLURM Token ", statusCode)
		os.Exit(1)
	}

}

// Build body for the HPC job submission
func buildBody(script string) string {
	var xmlString string
	xmlPart := fmt.Sprintf("{\"job\":{\"partition\":\"%s\",\"tasks\":%s,\"name\":\"%s\",\"nodes\":%s,\"current_working_directory\":\"%s\",\"environment\":{\"PATH\":\"%s\",\"LD_LIBRARY_PATH\":\"%s\"}},\"script\":\"", JobProp["Queue"], JobProp["Tasks"], JobProp["slurmJobName"], JobProp["NodesNumber"], JobProp["currentWorkingDir"], JobProp["envPath"], JobProp["envLibPath"])
	xmlString += fmt.Sprintf("%s%s%s\"} ", xmlString, xmlPart, script)
	return xmlString
}

// Submit request for job execution
func submit(slurmUsername string, slurmToken string, data map[string]string) int {
	url := HPCURL + "/job/submit"
	jobscript := ""
	if data["jobdata.scriptLocation"] == "s3" {
		s3info := strings.Split(data["jobdata.jobScript"], ":")
		jobscript = podutils.DownloadS3Data(s3info[0], s3info[1], data)
	} else {
		jobscript = data["jobdata.jobScript"]
	}

	paramstr := data["jobproperties"]
	err := json.Unmarshal([]byte(paramstr), &JobProp)
	if err != nil {
		klog.Info("Error in JobProperties provided ", err)

	}
	str_body := buildBody(jobscript)

	req, err := http.NewRequest("POST", url, strings.NewReader(str_body))
	if err != nil {
		klog.Error("Failed to create http request to connect to HPC cluster ", err)
		return 0
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-SLURM-USER-NAME", slurmUsername)
	req.Header.Set("X-SLURM-USER-TOKEN", slurmToken)

	respBody, statusCode := podutils.SendReq(req)

	if statusCode != 200 {
		klog.Error("Submitting job not successful - status code ", statusCode, " err ", string(respBody))
		return 0
	}

	var id JobId
	json.Unmarshal(respBody, &id)
	if id.Id == 0 {
		klog.Error("Job submittion failed (id 0)")
		return 0
	}
	klog.Info("Successfully submitted a job with job id ", id.Id)
	return id.Id
}

// Kill HPC Job
func kill(slurmUsername string, slurmToken string, id string) string {
	url := HPCURL + "job" + id

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Sprintf("Failed to create job kill request, err: %s", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-SLURM-USER-NAME", slurmUsername)
	req.Header.Set("X-SLURM-USER-TOKEN", slurmToken)

	respBody, statusCode := podutils.SendReq(req)
	if statusCode != 200 {
		return fmt.Sprintf("Failed to execute job kill request, status %d, respBody %s", statusCode, string(respBody))
	}
	return ""
}

// Get login token
func getToken() (string, string) {
	username := podutils.ReadMountedFileContent(CREDS_DIR + "username")
	password := podutils.ReadMountedFileContent(CREDS_DIR + "password")
	return username, password
}

// Add additional information from HPC
func getAdditionalInfo(info map[string]string) {
	start := JobMap["start_time"]
	if start != 0 {
		info["startTime"] = fmt.Sprint(start)
	}
	sub := JobMap["submit_time"]
	if sub != 0 {
		info["submitTime"] = fmt.Sprint(sub)
	}
	end := JobMap["end_time"]
	if end != 0 {
		info["endTime"] = fmt.Sprint(end)
	}
}

// Kill the job
func killJob(slurmUsername string, slurmToken string, id string, state string, info map[string]string) {
	// Check if the job is still running
	running := state != CANCELLED && state != COMPLETED && state != FAILED
	if running {
		// Only kill jobs that are still running
		res := kill(slurmUsername, slurmToken, id)
		if len(res) == 0 {
			klog.Info("Job", id, "killed successfully.")
			info["jobStatus"] = CANCELLED
		} else {
			klog.Info("Job ", id, " is not killed; msg: ", res, ". Continue in monitoring, will try to kill again.")
		}
	} else {
		info["jobStatus"] = CANCELLED
		klog.Info("Job ", id, " is already in finished state ", state)
	}
}

// Monitoring job execution
// Method that runs constantly monitoring HPC job
func monitor(slurmUsername string, slurmToken string, info map[string]string) {
	id := info["id"]
	// Run forever
	for {
		// Sleep before next run
		time.Sleep(time.Duration(POLL) * time.Second)

		// Get current config map
		cm := podutils.GetConfigMap()

		// Get current execution status and update config map
		var state = ""
		job := getJobInfo(slurmUsername, slurmToken, id)
		if job != nil {
			jstate := JobMap["job_state"]
			state = fmt.Sprint(jstate)
			info["jobStatus"] = state
			if state == COMPLETED || state == COMPLETING || state == CANCELLED || state == FAILED {
				// Get additional info from HPC job
				getAdditionalInfo(info)
			} else {
				// Check for kill flag
				if cm.Data["kill"] == "true" {
					killJob(slurmUsername, slurmToken, id, info["jobStatus"], info)
				}
			}
			podutils.UpdateConfigMap(cm, info)
		}
		// Terminate if we are done
		if state == COMPLETED {
			os.Exit(0)
		}
		if state == CANCELLED || state == FAILED {
			os.Exit(1)
		}

	}
}

// Main method
func main() {

	// Get namespace and job name from environment
	NAMESPACE = os.Getenv("NAMESPACE")
	JOB_NAME = os.Getenv("JOBNAME")

	// Initialize utils
	podutils.InitUtils(JOB_NAME, NAMESPACE)

	// Get config map and its parameters
	cm := podutils.GetConfigMap()
	HPCURL = cm.Data["resourceURL"]
	POLL, _ = strconv.Atoi(cm.Data["updateInterval"])
	S3 = cm.Data["s3.secret"]

	// Get Access Username, Token for Slurm  cluster
	slurmUsername, slurmToken := getToken()
	checkSlurmToken(slurmUsername, slurmToken)

	if len(slurmToken) == 0 || len(slurmUsername) == 0 {
		// Failed to get credentials for HPC cluster
		klog.Exit("Failed to get access token for HPC cluster")
	}

	// Get ID from config map
	id := cm.Data["id"]

	// create info for keeping track of execution parameters
	info := make(map[string]string)
	info["startTime"] = ""
	info["endTime"] = ""
	info["message"] = ""

	// If an ID is present in the config map it means that that we have already started a job
	if len(id) == 0 {
		klog.Info("Slurm Job with name ", JOB_NAME, " does not exist. Submitting new job.")

		intId := submit(slurmUsername, slurmToken, cm.Data)
		id = fmt.Sprint(intId)

		if len(id) == 0 {
			// Failed to submit a job
			info["jobStatus"] = FAILED
			info["message"] = "Failed to submit a job to HPC"
		} else {
			info["id"] = id
			info["jobStatus"] = SUBMITTED
			info["startTime"] = time.Now().Format(TIME)
		}

		podutils.UpdateConfigMap(cm, info)

		// Start monitoring or exit
		if len(id) != 0 {
			monitor(slurmUsername, slurmToken, info)
		} else {
			klog.Exit("Failed to start HPC job")
		}
	} else {
		// Job is already running
		klog.Info("Slurm Job  has associated ID in ConfigMap. Handling state.")
		info["id"] = id
		monitor(slurmUsername, slurmToken, info)
	}
}
