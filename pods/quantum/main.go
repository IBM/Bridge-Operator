package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"k8s.io/klog"

	"github.com/ibm/bridge-operator/podutils"
)

const (
	CREDS_DIR  = "/credentials/"
	SCRIPT_DIR = "/script/script"
	TIME       = "2006-01-02T15:04:05Z"
)

var CLOUD_URL string   // Cloud URL
var JOB_NAME string    // Job name
var NAMESPACE string   // Namespace
var POLL int           // Poll interval
var S3 string          // S3 secret - used to check whether we need S3 upload
var SERVICE_CRN string // Service CRN from secret
var API_KEY string     // API key from secret

// Structures for JSON conversion

// Program definition
type Program struct {
	ID           string      `json:"id" program:"id" job:"id"`
	Name         string      `json:"name" program:"name,omitempty" job:"-" binding:"required"`
	Cost         int         `json:"cost" program:"cost,omitempty" job:"-" db:"cost"`
	Description  string      `json:"description,omitempty" program:"description,omitempty" job:"-" db:"description"`
	Spec         ProgramSpec `json:"spec,omitempty" program:"spec,omitempty" job:"-" db:"spec"`
	Data         []byte      `json:"data,omitempty" program:"-" job:"-" db:"data"`
	CreationDate time.Time   `json:"creation_date,omitempty" program:"creation_date,omitempty" job:"-" db:"created_time"`
	UpdateDate   time.Time   `json:"update_date,omitempty" program:"update_date,omitempty" job:"-" db:"updated_time"`
	IsPublic     bool        `json:"is_public,omitempty" program:"is_public,omitempty" job:"-" db:"is_public"`
}

// Parameter definition used in Program spec
type Parameter struct {
	Description string `json:"description,omitempty"`
	Type        string `json:"type,omitempty"`
	Minimum     string `json:"minimum,omitempty"`
	Maximum     string `json:"maximum,omitempty"`
	Default     string `json:"default,omitempty"`
}

// List of parameters used in program spec
type ParameterList struct {
	Properties map[string]Parameter `json:"properties"`
	Schema     string               `json:"$schema"`
	Required   []string             `json:"required,omitempty"`
}

// Program sumission request
type ProgramSubmissionRequest struct {
	Name        string      `json:"name"`
	Data        []byte      `json:"data"`
	Cost        int         `json:"cost" program:"cost"`
	Description string      `json:"description,omitempty"`
	Spec        ProgramSpec `json:"spec"`
	IsPublic    bool        `json:"is_public,omitempty"`
}

// Program metadata definition
type ProgramMetadataDefinition struct {
	Name        string      `json:"name"`
	Cost        int         `json:"cost" program:"cost"`
	Description string      `json:"description,omitempty"`
	Spec        ProgramSpec `json:"spec"`
	IsPublic    bool        `json:"is_public,omitempty"`
}

// Parameters definition
type ParameterDefinition struct {
	Params map[string]interface{} `json:"params,omitempty" `
}

// ProgramSpec defines fields in a program that are not used for execution. The fields
// here are meant to contain a JSON object for the various inputs and outputs that a running
// program can handle. Parameters, ReturnValues, InterimResults should be JSON schemas.
type ProgramSpec struct {
	BackendRequirements map[string]string `json:"backend_requirements,omitempty"`
	Parameters          ParameterList     `json:"parameters,omitempty"`
	ReturnValues        ParameterList     `json:"return_values,omitempty"`
	InterimResults      ParameterList     `json:"interim_results,omitempty"`
}

// List programs response
type PaginatedProgramsResponse struct {
	Programs []Program `json:"programs"`
	Limit    int       `json:"limit"`
	Offset   int       `json:"offset"`
}

// Job submission request
type JobRunParams struct {
	ProgramID string                 `json:"program_id"`
	Backend   string                 `json:"backend"`
	Params    map[string]interface{} `json:"params,omitempty" `
}

// Job submission result
type JobSubmitResult struct {
	ID string `json:"id"`
}

// Job status result
type JobStatusResult struct {
	ID      string                 `json:"id"`
	Backend string                 `json:"backend"`
	Status  string                 `json:"status"`
	Params  map[string]interface{} `json:"params"`
	Program JobSubmitResult        `json:"program"`
	Created string                 `json:"created"`
	Runtime string                 `json:"runtime"`
}

// Set request headers
func setHeaders(req *http.Request) *http.Request {
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Service-CRN", SERVICE_CRN)
	req.Header.Set("Authorization", "apikey "+API_KEY)
	return req
}

// Get program(s) info
func getProgram(name string) *PaginatedProgramsResponse {
	// Build URL
	url := CLOUD_URL + "programs"
	if len(name) > 0 {
		url = url + "?name=" + name
	}
	// Create request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		klog.Error("Error creating Get program request; err ", err)
		return nil
	}
	// Execute
	respBody, statusCode := podutils.SendReq(setHeaders(req))
	if statusCode != 200 {
		klog.Error("Retrieving program not successful, status code ", statusCode)
		return nil
	}
	// Unmarshal result
	pi := PaginatedProgramsResponse{}
	json.Unmarshal(respBody, &pi)
	return &pi
}

// Submit job
func submitJob(request JobRunParams) *JobSubmitResult {
	// Build URL
	url := CLOUD_URL + "jobs"
	// Marshall input data
	data, err := json.Marshal(request)
	if err != nil {
		klog.Error("Error marshalling job request; err ", err)
		return nil
	}
	// Build request
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		klog.Error("Error creating submit job request; err ", err)
		return nil
	}
	// execute
	respBody, statusCode := podutils.SendReq(setHeaders(req))
	if statusCode != 200 {
		klog.Error("Submitting not successful, status code ", statusCode)
		return nil
	}
	// Unmarshal result
	res := JobSubmitResult{}
	json.Unmarshal(respBody, &res)
	return &res
}

// Get job status
func getJobState(jobID string) *JobStatusResult {
	// Build URL
	url := CLOUD_URL + "jobs/" + jobID
	// Build request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		klog.Error("Error creating Get job request; err ", err)
		return nil
	}
	// Execute
	respBody, statusCode := podutils.SendReq(setHeaders(req))
	if statusCode != 200 {
		klog.Error("Retrieving job not successful, status code ", statusCode)
		return nil
	}
	// Unmarshal result
	result := JobStatusResult{}
	json.Unmarshal(respBody, &result)
	return &result
}

// Get job results
func getJobResults(jobID string) string {
	// Build URL
	url := CLOUD_URL + "jobs/" + jobID + "/results"
	// Create request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		klog.Error("Error creating Get job result request; err ", err)
		return ""
	}
	// Execute
	respBody, statusCode := podutils.SendReq(setHeaders(req))
	if statusCode != 200 {
		klog.Error("Retrieving job results is not successful, status code ", statusCode)
		return ""
	}
	// Return results
	return string(respBody)
}

// Get job results
func getJobInterimResults(jobID string) string {
	// Build URL
	url := CLOUD_URL + "jobs/" + jobID + "/interim_results"
	// Create request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		klog.Error("Error creating Get job interim result request; err ", err)
		return ""
	}
	// Execute
	respBody, statusCode := podutils.SendReq(setHeaders(req))
	if statusCode != 200 {
		klog.Error("Retrieving job interim results is not successful, status code ", statusCode)
		return ""
	}
	// Return result
	return string(respBody)
}

// Get job results
func getJobLogs(jobID string) string {
	// Build URL
	url := CLOUD_URL + "jobs/" + jobID + "/logs"
	// Create request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		klog.Error("Error creating Get job logs request; err ", err)
		return ""
	}
	// Execute
	respBody, statusCode := podutils.SendReq(setHeaders(req))
	if statusCode != 200 {
		klog.Error("Retrieving job logs is not successful, status code ", statusCode)
		return ""
	}
	// Return result
	return string(respBody)
}

// Delete job
func deleteJob(jobID string) {
	// Build URL
	url := CLOUD_URL + "jobs/" + jobID
	// Create request
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		klog.Error("Error creating delete job request; err ", err)
		return
	}
	// Execute
	_, statusCode := podutils.SendReq(setHeaders(req))
	if statusCode != 204 {
		klog.Error("Deleting job not successful, status code ", statusCode)
		return
	}
}

// Delete program
func deleteProgram(programID string) {
	// Build URL
	url := CLOUD_URL + "programs/" + programID
	// Create request
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		klog.Error("Error creating delete program request; err ", err)
		return
	}
	// Execute
	_, statusCode := podutils.SendReq(setHeaders(req))
	if statusCode != 204 {
		klog.Error("Deleting program not successful, status code ", statusCode)
		return
	}
}

// Add program
func addProgram(request ProgramSubmissionRequest) *Program {
	// Build URL
	url := CLOUD_URL + "programs"
	// Marshall data
	data, err := json.Marshal(request)
	if err != nil {
		klog.Error("Error marshalling program request; err ", err)
		return nil
	}
	// Create request
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		klog.Error("Error creating create program request; err ", err)
		return nil
	}
	// Execute
	respBody, statusCode := podutils.SendReq(setHeaders(req))
	if statusCode != 201 {
		klog.Error("Creating program was not successful, status code ", statusCode)
		return nil
	}
	// Unmarshal result
	res := Program{}
	json.Unmarshal(respBody, &res)
	return &res
}

// Kill job
func CancelJob(jobID string) bool {
	// Create URL
	url := CLOUD_URL + "jobs/" + jobID + "/cancel"
	// Build request
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		klog.Error("Error creating kill job request; err ", err)
		return false
	}
	// Execute
	_, statusCode := podutils.SendReq(setHeaders(req))
	if statusCode != 204 {
		klog.Error("Killing job not successful, status code ", statusCode)
		return false
	}
	return true
}

// Add additional information from quqntum job
func getAdditionalInfo(job *JobStatusResult, info map[string]string) {

	info["status.submitTime"] = job.Created
	info["status.endTime"] = time.Now().Format(TIME)
}

// Kill the job
func killJob(state string, info map[string]string) {
	// Check if the job is queued or running
	if state == "Queued" || state == "Running" {
		// Only kill jobs that are queued or running
		res := CancelJob(info["id"])
		if res {
			klog.Info("Job ", info["id"], " killed successfully.")
		} else {
			klog.Info("Job ", info["id"], " is not killed; msg: ", res, ". Continue in monitoring, will try to kill again.")
		}
	} else {
		klog.Info("Job ", info["id"], " is already in finished state ", state)
	}
}

// Submit job for execution
func submit(data map[string]string) string {

	var programID string
	script_location := data["jobdata.scriptLocation"]
	script_extlocation := data["jobdata.scriptExtraLocation"]

	if script_location == "remote" {
		// We have an uploaded program
		program := getProgram(data["jobdata.jobScript"])
		if program == nil || len(program.Programs) != 1 {
			klog.Info("Failed to find program ", data["jobdata.jobScript"])
			return ""
		}
		programID = program.Programs[0].ID
	} else {
		// Upload our program itself
		var program_data string
		if script_location == "inline" {
			program_data = data["jobdata.jobScript"]
		} else {
			bucketobj := strings.Split(data["jobdata.jobScript"], ":")
			program_data = podutils.DownloadS3Data(bucketobj[0], bucketobj[1], data)
		}
		if len(program_data) == 0 {
			klog.Info("Failed to load program ", data["jobdata.jobScript"])
			return ""
		}
		var program_metadata string
		if script_extlocation == "inline" {
			program_metadata = data["jobdata.scriptMetadata"]
		} else {
			bucketobj := strings.Split(data["jobdata.scriptMetadata"], ":")
			program_metadata = podutils.DownloadS3Data(bucketobj[0], bucketobj[1], data)
		}

		programMetadata := ProgramMetadataDefinition{}
		_ = json.Unmarshal([]byte(program_metadata), &programMetadata)

		submissionRequest := ProgramSubmissionRequest{
			Name:        programMetadata.Name,
			Data:        []byte(program_data),
			Cost:        programMetadata.Cost,
			Description: programMetadata.Description,
			Spec:        programMetadata.Spec,
			IsPublic:    programMetadata.IsPublic,
		}
		program := addProgram(submissionRequest)
		if program == nil {
			klog.Info("Failed to upload program ")
			return ""
		}
		programID = program.ID
	}

	// Get parameters
	var params_string string
	if script_extlocation == "inline" {
		params_string = data["jobdata.jobParameters"]
	} else {
		bucketobj := strings.Split(data["jobdata.jobParameters"], ":")
		params_string = podutils.DownloadS3Data(bucketobj[0], bucketobj[1], data)
	}

	parameters := ParameterDefinition{}
	_ = json.Unmarshal([]byte(params_string), &parameters)

	// Submit job
	jobRequest := JobRunParams{
		ProgramID: programID,
		Backend:   "",
		Params:    parameters.Params,
	}
	submissionResult := submitJob(jobRequest)
	if submissionResult == nil {
		klog.Info("Failed to submit program ")
		return ""
	}

	return submissionResult.ID
}

// Monitoring job execution
// Method that runs constantly monitoring quantum job
func monitor(info map[string]string) {
	id := info["id"]
	// Run forever
	for {
		// Sleep before next run
		time.Sleep(time.Duration(POLL) * time.Second)

		// Get current config map
		cm := podutils.GetConfigMap()

		// Get current execution status and update config map
		var state = ""
		job := getJobState(id)
		if job != nil {
			state = job.Status
			info["status.jobStatus"] = strings.ToUpper(state)
			if state == "Completed" || state == "Cancelled" || state == "Cancelled - Ran too long" || state == "Failed" {
				// Upload outputs to S3

				podutils.UploadS3Data(cm.Data, info, []podutils.UploadFile{
					podutils.UploadFile{Name: "results", Content: getJobResults(id)},
					podutils.UploadFile{Name: "intermediateresults", Content: getJobInterimResults(id)},
					podutils.UploadFile{Name: "logs", Content: getJobLogs(id)},
				})
				// Get additional info from Quantum and upload it
				getAdditionalInfo(job, info)
				// Adjust status
				if state == "Cancelled - Ran too long" {
					info["status.jobStatus"] = "FAILED"
					info["status.message"] = "Job execution takes too long, aborted by runtime"
				} else if state == "Completed" {
					info["status.jobStatus"] = "SUCCEEDED"
				} else if state == "Cancelled" {
					info["status.jobStatus"] = "KILL"
				}
			} else {
				// Check for kill flag
				if cm.Data["kill"] == "true" {
					killJob(state, info)
				}
			}

			podutils.UpdateConfigMap(cm, info)
		}

		// Terminate if we are done
		if state == "Completed" {
			os.Exit(0)
		}
		if state == "Cancelled" || state == "Cancelled - Ran too long" || state == "Failed" {
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
	S3 = cm.Data["s3.secret"]
	POLL, _ = strconv.Atoi(cm.Data["updateInterval"])
	CLOUD_URL = cm.Data["resourceURL"]
	SERVICE_CRN = podutils.ReadMountedFileContent(CREDS_DIR + "username")
	API_KEY = podutils.ReadMountedFileContent(CREDS_DIR + "password")

	// Get ID from config map
	id := cm.Data["id"]

	// create info for keeping track of execution parameters
	info := make(map[string]string)
	info["status.startTime"] = ""
	info["status.submitTime"] = ""
	info["status.endTime"] = ""
	info["status.message"] = ""

	// If an ID is present in the config map it means that that we have already started a job
	if len(id) == 0 {
		klog.Info("Quantum Job with name ", JOB_NAME, " does not exist. Submitting new job.")
		// Trying to submit a job. Here we are trying several times to successfully submit a job
		id := submit(cm.Data)

		// Update execution state in config map
		if len(id) == 0 {
			// Failed to submit a job
			info["status.jobStatus"] = "FAILED"
			info["message"] = "Failed to submit a job to Quantum"
		} else {
			info["id"] = id
			info["status.jobStatus"] = "SUBMITTED"
			info["status.startTime"] = time.Now().Format(TIME)
		}
		podutils.UpdateConfigMap(cm, info)
		// Start monitoring or exit
		if len(id) != 0 {
			monitor(info)
		} else {
			klog.Exit("Failed to start quantum job")
		}
	} else {
		// Job is already running
		klog.Info("Quantom Job with name ", JOB_NAME, " has associated ID in ConfigMap. Handling state.")
		info["id"] = id
		monitor(info)
	}
}
