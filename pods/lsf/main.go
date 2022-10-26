//=============================================================================
// Code for managing HPC jobs deployed on LSF
// Some parts are tailored for Application Centre installed on LSF with Cluster Systems Manager
// If Application Centre installed on different infrastructure, additional changes might be needed
// Application Centre versions prior to 9.1.5
//
// IBM Dublin Research Lab
//
//=============================================================================

package main

import (
	"encoding/json"
	e "errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	mxj "github.com/clbanning/mxj/v2"

	"github.com/google/uuid"
	"k8s.io/klog"

	"github.com/ibm/bridge-operator/podutils"
)

const (
	MULTIPLE_ACCEPT_TYPE = "text/plain,application/xml,text/xml,multipart/mixed"
	TIME                 = "2006-01-02T15:04:05Z"

	SUBMITTED = "SUBMITTED"
	PENDING   = "PENDING"
	RUNNING   = "RUNNING"
	DONE      = "DONE"
	EXIT      = "EXIT"
	KILL      = "KILL"
	UNKNOWN   = "UNKNOWN"
	FAILED    = "FAILED"

	CREDS_DIR    = "/credentials/"
	BATCH_SCRIPT = "/downloads/script"
	FILES_DIR    = "/downloads/"

	TOKEN_SLEEP = 3
)

var AC string
var JOB_NAME string
var NAMESPACE string
var POLL int
var S3 string
var JobProp map[string]string
var FilePath string

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
	Id int `json:"id"`
}

// HPC Job info
type JobInfo struct {
	Total string                 `json:"@total"`
	Job   map[string]interface{} `json:"job"`
}

// Login to HPC system
func login(username, pass string) string {
	url := AC + "ws/logon"
	strBody := fmt.Sprintf("<User><name>%s</name> <pass>%s</pass> </User>", username, pass)

	req, err := http.NewRequest("POST", url, strings.NewReader(strBody))
	if err != nil {
		klog.Error("Error creating login request; error ", err)
		return ""
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/xml")

	respBody, statusCode := podutils.SendReq(req)

	if statusCode != 200 {
		klog.Error("Login is not successful, status code ", statusCode, " err ", string(respBody))
		return ""
	}

	var unmarshaled map[string]string
	json.Unmarshal(respBody, &unmarshaled)
	return unmarshaled["token"]
}

// Gets detailed job information for jobs that have the specified job IDs.
// If job is not return by call to all jobs, returns 404
func getJobInfo(token, id string) *JobInfo {
	url := AC + "ws/jobs/" + id
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		klog.Error("Error creating Job Info request; err ", err)
		return nil
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Cookie", token)

	respBody, statusCode := podutils.SendReq(req)
	if statusCode != 200 {
		klog.Error("Retrieving job info not successful, status code ", statusCode)
		return nil
	}

	job := JobInfo{}
	json.Unmarshal(respBody, &job)
	return &job
}

// EJ need to test this once we have working AC access
func getOldJobId(token, id string) string {
	url := AC + "/platform/ws/jobhistory?ids=*"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		klog.Error("Error retrieving job history; err ", err)
		return ""
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Cookie", token)

	_, statusCode := podutils.SendReq(req)
	if statusCode != 200 {
		klog.Error("Retrieving job info not successful, status code ", statusCode)
		return ""
	}

	return ""

}

// Save file to the local drive
func saveDownload(response, filename, dst string) error {
	content := strings.Split(response, filename+">")[1]
	if len(dst) > 0 && dst[len(dst)-1] != '/' {
		dst = fmt.Sprintf("%s/", dst)
	}
	err := os.WriteFile(dst+filename, []byte(content), 0644)
	return err
}

// Download file from HPC
func downloadFile(token, filename, id string) ([]byte, error) {
	url := AC + "webservice/pacclient/file/" + id

	// Create a new download request
	req, err := http.NewRequest("GET", url, strings.NewReader(filename))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Cookie", token)
	req.Header.Set("Accept", MULTIPLE_ACCEPT_TYPE)
	req.Header.Set("Content-Type", "text/plain")

	// Download
	respBody, statusCode := podutils.SendReq(req)
	if statusCode != 200 {
		return nil, fmt.Errorf("downloading file from server not successful (%d)", statusCode)
	}
	return respBody, nil
}

// Build name area for body for the HPC job submission
func nameArea(boundary, appName string) string {
	return fmt.Sprintf("--%s\r\nContent-Disposition: form-data; name=\"AppName\"\r\nContent-ID: <AppName>\r\n\r\n%s\r\n", boundary, appName)
}

// Build parameters (in XML) for body for the HPC job submission
func paramsXml(boundary2 string, params map[string]string) string {
	var xmlString string
	for pName, pVal := range params {
		pHead := fmt.Sprintf("--%s\r\nContent-Disposition: form-data; name=\"%s\"\r\nContent-Type: application/xml; charset=UTF-8\r\nContent-Transfer-Encoding: 8bit\r\nAccept-Language: en\r\n\r\n", boundary2, pName)
		pDesc := fmt.Sprintf("<AppParam><id>%s</id><value>%s</value><type></type></AppParam>\r\n", pName, pVal)
		xmlString += pHead + pDesc
	}
	return xmlString
}

// Build parameter files (in XML) for body for the HPC job submission
func paramsFilesXml(boundary2 string, files map[string]string) string {
	var xmlString string
	for fPath, fAction := range files {
		name := strings.Split(fPath, "/")
		fHead := fmt.Sprintf("--%s\r\nContent-Disposition: form-data; name=\"%s\"\r\nContent-Type: application/xml; charset=UTF-8\r\nContent-Transfer-Encoding: 8bit\r\nAccept-Language: en\r\n\r\n", boundary2, name[len(name)-1])
		fDesc := fmt.Sprintf("<AppParam><id>%s</id><value>%s,%s</value><type>file</type></AppParam>\r\n", name[len(name)-1], name[len(name)-1], fAction)
		xmlString += fHead + fDesc
	}
	return xmlString
}

// Build params area for body for the HPC job submission
func paramsArea(boundary string, params, files map[string]string) string {
	boundary2 := uuid.New().String()
	pHead := fmt.Sprintf("--%s\r\nContent-Disposition: form-data; name=\"data\"\r\nContent-ID: <data>\r\nAccept-Language: en-us\r\nContent-Type: multipart/mixed; boundary=%s\r\n\r\n", boundary, boundary2)
	pBody := paramsXml(boundary2, params)
	pFilesBody := paramsFilesXml(boundary, files)
	xmlString := fmt.Sprintf("%s%s%s--%s--\r\n", pHead, pBody, pFilesBody, boundary2)
	return xmlString
}

// Build files area for body for the HPC job submission
func filesArea(boundary string, files map[string]string) string {
	var xmlString string
	for fPath := range files {
		name := strings.Split(fPath, "/")
		fHead := fmt.Sprintf("--%s\r\nContent-Disposition: form-data; name=\"%s\"\r\nContent-Type: application/octet-stream\r\nContent-ID: <%s>\r\nContent-Transfer-Encoding: 8bit\r\n", boundary, name[len(name)-1], name[len(name)-1])
		fContent, err := ioutil.ReadFile(fPath)
		if err != nil {
			klog.Error("Failed to load file ", fPath, " error ", err)
			return ""
		}
		klog.Info("Successfully loaded file ", fPath)
		xmlPart := fmt.Sprintf("%s%s\r\n%v\r\n", xmlString, fHead, string(fContent))
		xmlString += xmlPart
	}
	return xmlString
}

// Build body for the HPC job submission
func buildBody(boundary string, params, files map[string]string) string {
	nArea := nameArea(boundary, "generic")
	pArea := paramsArea(boundary, params, files)
	fArea := filesArea(boundary, files)
	body := fmt.Sprintf("%s%s%s--%s--", nArea, pArea, fArea, boundary)
	//klog.Info("Submit request body ", body)
	return body
}

// Build job spec for the HPC job submission
func buildJobParams(data map[string]string) map[string]string {
	jobSpec := make(map[string]string)
	jobSpec["JOB_NAME"] = JOB_NAME
	if data["jobdata.scriptLocation"] == "remote" {
		jobSpec["COMMANDTORUN"] = data["jobdata.jobScript"]
	} else {
		jobSpec["COMMANDTORUN"] = "chmod 755 `pwd`/script;sed -i -e 's/\\r$//' `pwd`/script;`pwd`/script"
	}

	for k, v := range JobProp {
		if k == "numnodes" {
			jobSpec["EXTRA_PARAMS"] = fmt.Sprintf("-nnodes %s", v)
		} else {
			lsfEqK := RESOURCES[k]
			if len(lsfEqK) != 0 {
				jobSpec[lsfEqK] = v
			}
		}
	}
	return jobSpec
}

func saveInlineScript(scriptcontents string, files map[string]string) map[string]string {
	err := os.WriteFile(BATCH_SCRIPT, []byte(scriptcontents), 0666)
	if err != nil {
		klog.Info("Error saving file ", BATCH_SCRIPT, err)
	}
	files[BATCH_SCRIPT] = "upload"
	return files
}

// Submit request for job execution
func submit(token string, data map[string]string) int {
	url := AC + "ws/jobs/submit"
	boundary := uuid.New().String()

	files := make(map[string]string)
	if data["jobdata.scriptLocation"] == "inline" {
		scriptcontents := data["jobdata.jobScript"]
		files = saveInlineScript(string(scriptcontents), files)
	} else if data["jobdata.scriptLocation"] == "s3" {
		//download to BATCH_SCRIPT
		downloadScript(data)
		files[BATCH_SCRIPT] = "upload"
	}

	job_spec := buildJobParams(data)
	str_body := buildBody(boundary, job_spec, files)

	req, err := http.NewRequest("POST", url, strings.NewReader(str_body))
	if err != nil {
		klog.Error("Failed to create http request to connect to HPC cluster ", err)
		return 0
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Cookie", token)
	req.Header.Set("Content-Type", "multipart/mixed; boundary="+boundary)
	req.Header.Set("Content-Length", fmt.Sprint(len(str_body)))

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
func kill(token, id string) string {
	url := AC + "ws/userCmd"
	strBody := fmt.Sprintf("<UserCmd><cmd>bkill %s</cmd></UserCmd>", id)

	req, err := http.NewRequest("POST", url, strings.NewReader(strBody))
	if err != nil {
		return fmt.Sprintf("Failed to create job kill request, err: %s", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/xml")
	req.Header.Set("Cookie", token)

	respBody, statusCode := podutils.SendReq(req)
	if statusCode != 200 {
		return fmt.Sprintf("Failed to execute job kill request, status %d, respBody %s", statusCode, string(respBody))
	}
	return ""
}

// Get time from login token
func parseTimeFromToken(token string) time.Time {
	t := strings.Split(token, "#quote#")[1]
	ti, err := time.Parse(TIME, t)
	if err != nil {
		klog.Error("Error parsing token time value ", err)
		return time.Now().Add(time.Duration(-3) * time.Hour)
	}
	return ti
}

// Get login token
func getToken() string {
	username := podutils.ReadMountedFileContent(CREDS_DIR + "username")
	password := podutils.ReadMountedFileContent(CREDS_DIR + "password")
	token := login(username, password)
	if len(token) == 0 {
		return ""
	}
	token = strings.Replace(token, "\"", "#quote#", -1)
	return "platform_token=" + token
}

// Add additional information from HPC
func getAdditionalInfo(job *JobInfo, info map[string]string) {
	start := job.Job["startTime"]
	if start != nil {
		info["startTime"] = fmt.Sprint(start)
	}
	end := job.Job["endTime"]
	if end != nil {
		info["endTime"] = fmt.Sprint(end)
	}
	cwd := job.Job["cwd"]
	if cwd != nil {
		if len(S3) == 0 {
			info["message"] = fmt.Sprintf("Output and error files can be found in your home directory, path %s", fmt.Sprint(cwd))
		} else {
			info["message"] = fmt.Sprintf("Output, error and additional downloaded files can be found at S3 location specified or your home directory (path %s)", fmt.Sprint(cwd))
		}

	}
}

// Kill the job
func killJob(token, id, state string, info map[string]string) {
	// Check if the job is still running
	running := state != KILL && state != DONE && state != EXIT && state != FAILED
	if running {
		// Only kill jobs that are still running
		res := kill(token, id)
		if len(res) == 0 {
			klog.Info("Job", id, "killed successfully.")
			info["jobStatus"] = KILL
		} else {
			klog.Info("Job ", id, " is not killed; msg: ", res, ". Continue in monitoring, will try to kill again.")
		}
	} else {
		info["jobStatus"] = KILL
		klog.Info("Job ", id, " is already in finished state ", state)
	}
}

// Monitoring job execution
// Method that runs constantly monitoring HPC job
func monitor(tokenTime time.Time, token string, info map[string]string) {
	id := info["id"]
	// Run forever
	for {
		// Sleep before next run
		time.Sleep(time.Duration(POLL) * time.Second)

		// Check and update token. We are here assuming that a token is valifd for at least 2 hours,
		// So we reaqqire it every 2 hours
		if time.Since(tokenTime).Hours() >= 2 {
			token = getToken()
			tokenTime = parseTimeFromToken(token)
		}

		// Get current config map
		cm := podutils.GetConfigMap()

		// Get current execution status and update config map
		var state = ""
		job := getJobInfo(token, id)
		if job != nil {
			jstate := job.Job["jobStatus"]
			if jstate != nil {
				state = fmt.Sprint(jstate)
				info["jobStatus"] = state
				if state == DONE || state == EXIT || state == KILL || state == FAILED {
					// If specified upload outputs to S3
					if cm.Data["s3upload.files"] != "" {
						uploadOutputs(token, id, cm.Data)
					}
					// Get additional info from HPC
					getAdditionalInfo(job, info)
				} else {
					// Check for kill flag
					if cm.Data["kill"] == "true" {
						killJob(token, id, info["jobStatus"], info)
					}
				}
				podutils.UpdateConfigMap(cm, info)
			}
		}

		// Check for kill flag
		if JobProp["kill"] == "true" {
			killJob(token, id, info["jobStatus"], info)
		}

		// Terminate if we are done
		if state == DONE {
			os.Exit(0)
		}
		if state == EXIT || state == KILL || state == FAILED {
			os.Exit(1)
		}
	}
}

// Upload outputs to S3
func uploadOutputs(token string, id string, data map[string]string) {
	// Skip if S3 info is not provided
	if len(S3) == 0 {
		return
	}

	// Get the list of files to upload
	toUpload := strings.Split(data["s3upload.files"], ",")

	// Add execution script, if not already on s3
	if data["jobdata.jobScript"] != "s3" {
		err := podutils.UploadS3DataDisk(data, []podutils.UploadFileLocation{
			podutils.UploadFileLocation{Name: JOB_NAME + "/" + "script", Path: BATCH_SCRIPT}})
		if err == nil {
			toUpload = append(toUpload, BATCH_SCRIPT)
		}
	}

	// Build S3 file prefix
	prefix := JOB_NAME + "/"

	// For every file to upload
	var objects []podutils.UploadFileLocation
	for _, f := range toUpload {
		// Read file content
		content, err := downloadFile(token, f, id)
		if err != nil {
			klog.Info("Error uploading file ", f, " this file won't be uploaded to S3; err ", err.Error())
			continue
		}
		// Create file locally
		fnames := strings.Split(f, "/")
		shortname := fnames[len(fnames)-1]
		err = saveDownload(string(content), shortname, FILES_DIR)
		if err != nil {
			klog.Info("Error saving file ", f, ", this file won't be uploaded to S3; err ", err.Error())
			continue
		}
		objects = append(objects, podutils.UploadFileLocation{Name: prefix + shortname, Path: FILES_DIR + shortname})
	}
	// Load to S3
	podutils.UploadS3DataDisk(data, objects)

}

// Download files from S3 before job submission
func downloadInputs(token string, data map[string]string) {
	// Skip if S3 info is not provided
	if len(S3) == 0 {
		return
	}

	// Get the list of files to download
	toDownload := strings.Split(data["jobdata.additionalData"], ", ")
	klog.Info("List of files to download: ", toDownload)

	// For every file to download
	for _, f := range toDownload {
		pairs := strings.Split(f, ":")
		bucket := pairs[0]
		scriptPath := pairs[1]
		scriptPathPairs := strings.Split(scriptPath, "/")
		script := scriptPathPairs[len(scriptPathPairs)-1]
		FilePath = scriptPath
		klog.Info("Bucket:", bucket, " File:", scriptPath)

		// Download object to file
		err := podutils.DownloadS3DataDisk(bucket, scriptPath, FILES_DIR+script, data)
		if err != nil {
			klog.Info("Error downloading file ", scriptPath, " this file won't be downloaded from S3; err ", err.Error())
		} else {
			klog.Info("Successfuly downloaded file ", scriptPath, " from S3 to ", FILES_DIR+script)
			// Load file to HPC cluster
			err := uploadInputFile(token, script, data)
			if err != nil {
				klog.Info("Error uploading file ", script, " to S3")
			}
		}
	}
}

func downloadScript(data map[string]string) {
	// Skip if S3 info is not provided
	if len(S3) == 0 {
		return
	}

	pairs := strings.Split(data["jobdata.jobScript"], ":")
	bucket := pairs[0]
	script := pairs[1]
	klog.Info("Attempting to download file ", script)
	// download file content or file from S3
	err := podutils.DownloadS3DataDisk(bucket, script, BATCH_SCRIPT, data)
	if err != nil {
		klog.Info("Error downloading file ", script, " this file won't be downloaded from S3; err ", err.Error())
	} else {
		klog.Info("Successfuly downloaded file ", script, " from S3 at ", BATCH_SCRIPT)
	}

}

// Build directory area for body for upload to HPC cluster
func directoryAreaUpload(boundary string, dirname string) string {
	var xmlString string
	xmlPart := fmt.Sprintf("--%s\r\nContent-Disposition: form-data; name=\"dir\"\r\n\r\n", boundary)
	xmlString += fmt.Sprintf("%s%s%s\r\n", xmlString, xmlPart, dirname)
	return xmlString
}

// Build files area for body for upload to HPC cluster
func filesAreaUpload(boundary string, filename string) string {
	var xmlString string
	filePath := FILES_DIR + filename
	fHead := fmt.Sprintf("--%s\r\nContent-Disposition: form-data; name=\"%s\"\r\nContent-Type: application/octet-stream\r\nContent-ID: <%s>\r\n", boundary, filename, filename)
	fContent, err := ioutil.ReadFile(filePath)
	if err != nil {
		klog.Error("Failed to load file ", filePath, " error ", err)
		return ""
	}
	xmlString += fmt.Sprintf("%s%s\r\n%v\r\n", xmlString, fHead, string(fContent))
	return xmlString
}

func buildBodyUpload(boundary string, filename string, dirname string) string {
	dArea := directoryAreaUpload(boundary, dirname)
	fArea := filesAreaUpload(boundary, filename)
	body := fmt.Sprintf("%s%s--%s--", dArea, fArea, boundary)
	return body
}

// upload file to HPC https://www.ibm.com/docs/en/slac/10.1.0?topic=915-upload-filesdeprecated
func uploadInputFile(token, filename string, data map[string]string) error {
	// Inputfiles moved to HPC cluster before job submission, therefore we need a job id to move the file, take an job id
	//GET request to http://c699wrk01.pok.stglabs.ibm.com:8080/platform/ws/jobhistory?ids=*
	//id := getOldJobId(token, id string)

	klog.Info("Attempting to upload file ", filename, " to cluster")

	//The above doesn't work on WSC CSM (so using fixed job id for now), need to test on working LSF cluster
	id := JobProp["pastid"]
	url := AC + "ws/jobfiles/upload/" + id
	boundary := uuid.New().String()

	inputdir := JobProp["inputfiledirectory"]
	if len(inputdir) > 0 {
		str_body := buildBodyUpload(boundary, filename, inputdir)

		req, err := http.NewRequest("POST", url, strings.NewReader(str_body))
		if err != nil {
			klog.Error("Failed to create http request to connect to HPC cluster ", err)
			return err
		}
		req.Header.Set("Accept", "application/xml")
		req.Header.Set("Cookie", token)
		req.Header.Set("Content-Type", "multipart/mixed; boundary="+boundary)
		//req.Header.Set("Content-Length", fmt.Sprint(len(str_body)))

		respBody, statusCode := podutils.SendReq(req)

		if statusCode != 200 {
			klog.Error("Uploading files to cluster not successful - status code ", statusCode, " err ", string(respBody))
			return err
		} else {
			klog.Info("Successfully uploaded file ", filename, " to cluster")
		}

		return err
	} else {
		klog.Info("No input file directory specified to upload to cluster!")
		return e.New("No input file directory specified to upload to cluster!")

	}
}

// Get job fom history. Used if we can't find it by ID
func getJobFromHistory(token, id string) []interface{} {
	url := AC + "ws/jobhistory?ids=" + id
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		klog.Fatal("Creating request for retrieving job from history for id ", id, " failed; err ", err)
	}
	req.Header.Set("Content-Type", "application/xml")
	req.Header.Set("Accept", MULTIPLE_ACCEPT_TYPE)
	req.Header.Set("Cookie", token)

	respBody, statusCode := podutils.SendReq(req)

	if statusCode != 200 {
		klog.Fatal("Retrieving job history for id ", id, " not successful ", statusCode, "; err ", string(respBody))
	}

	var mv mxj.Map
	mv, err = mxj.NewMapXml(respBody)
	if err != nil {
		klog.Fatal("Unmarshaling job history to XML not successful; err ", err.Error())
	}

	valForKey, err := mv.ValuesForPath("jobHistory.history")
	if err != nil {
		klog.Fatal("Error getting job from history; err ", err.Error())
	}
	return valForKey
}

// Extract state from history
func stateFromHistory(job []interface{}) string {
	jk := reflect.ValueOf(job[0])
	val := jk.MapIndex(reflect.ValueOf("content"))
	content := val.Interface().(string)
	split := strings.Split(content, " ")
	last := split[len(split)-1]
	if strings.Contains(last, "CSM_ALLOCATION_ID=") {
		return RUNNING
	}
	if strings.Contains(content, "Done successfully.") {
		return DONE
	}
	if strings.Contains(content, "Completed <exit>") {
		if strings.Contains(content, "<KILL>") {
			return KILL
		}
		return EXIT
	}
	return UNKNOWN
}

// Extract time from history
func timeFromHistory(job []interface{}) string {
	jk := reflect.ValueOf(job[0])
	ts := jk.MapIndex(reflect.ValueOf("timeSummary"))
	toc := ts.Elem().MapIndex(reflect.ValueOf("timeOfCalculation"))
	return fmt.Sprint(toc)
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
	AC = cm.Data["resourceURL"]
	S3 = cm.Data["s3.secret"]

	paramstr := cm.Data["jobproperties"]
	err := json.Unmarshal([]byte(paramstr), &JobProp)
	if err != nil {
		klog.Info("Error in JobProperties provided ", err)

	}

	POLL, _ = strconv.Atoi(cm.Data["updateinterval"])

	// Get Access Token for HPC cluster
	token := getToken()
	if len(token) == 0 {
		// Failed to get token from HPC cluster
		klog.Exit("Failed to get access token from HPC cluster")
	}
	tokenTime := parseTimeFromToken(token)

	// Get ID from config map
	id := cm.Data["id"]

	// create info for keeping track of execution parameters
	info := make(map[string]string)
	info["startTime"] = ""
	info["endTime"] = ""
	info["message"] = ""

	// If an ID is present in the config map it means that that we have already started a job
	if len(id) == 0 {
		klog.Info("LSF Job with name ", JOB_NAME, " does not exist. Submitting new job.")

		//Check if downloading of files from S3 to HPC cluster is required first
		// id here is placeholder and can be provided in configmap 'pastid'

		if cm.Data["jobdata.scriptExtraLocation"] == "s3" {
			downloadInputs(token, cm.Data)
		}

		sid := ""
		// Trying to submit a job. Here we are trying several times to successfully submit a job
		id := submit(token, cm.Data)

		// Update execution state in config map
		if id == 0 {
			// Failed to submit a job
			info["jobStatus"] = FAILED
			info["message"] = "Failed to submit a job to HPC"
		} else {
			sid = fmt.Sprint(id)
			info["id"] = sid
			info["jobStatus"] = SUBMITTED
		}
		podutils.UpdateConfigMap(cm, info)
		// Start monitoring or exit
		if id != 0 {
			monitor(tokenTime, token, info)
		} else {
			klog.Exit("Failed to start HPC job")
		}
	} else {
		// Job is already running
		klog.Info("LSF Job with name ", JOB_NAME, " has associated ID in ConfigMap. Handling state.")
		info["id"] = id

		// Get Job info from HPC by ID first
		var state string
		jobInfo := getJobInfo(token, id)

		if jobInfo != nil {
			state = fmt.Sprint(jobInfo.Job["jobStatus"])
			info["jobStatus"] = state

			// Continue processing
			if state == EXIT || state == DONE || state == KILL {
				podutils.UpdateConfigMap(cm, info)
			} else {
				monitor(tokenTime, token, info)
			}
		} else {
			// Get job from history. Here we assume that the job is completed, so just update the state
			job := getJobFromHistory(token, id)
			state = stateFromHistory(job)
			info["jobStatus"] = state
			info["submitTime"] = timeFromHistory(job)
			info["history"] = "true"
			info["message"] = fmt.Sprintf("Job with id %s found in history with state %s. Can't retrieve more information", id, state)
			podutils.UpdateConfigMap(cm, info)
		}
	}
}
