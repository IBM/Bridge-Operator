package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"
	//	"os"
	"k8s.io/klog"
)

const (
	API_KEY     = [API key]
	SERVICE_CRN = [CRN]
	CLOUD_URL   = "https://us-east.quantum-computing.cloud.ibm.com/"

	DefaultProgramCost = 60 * 10 // 10 minutes

)

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

// Program definition
type ProgramDefinition struct {
	Name        string      `json:"name"`
	Data        string      `json:"data"`
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

// HTTP client used for interaction with HPC HTTP APIs
var client http.Client

// Send HTTP request
func sendReq(req *http.Request) ([]byte, int) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Service-CRN", SERVICE_CRN)
	req.Header.Set("Authorization", "apikey " + API_KEY)
	resp, err := client.Do(req)
	if err != nil {
		klog.Error("Error invoking HTTP client; error ", err)
		return nil, -1
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		klog.Error("Error reading HTTP result; error ", err)
		return nil, -1
	}
	return respBody, resp.StatusCode
}

// Get program(s) info
func getProgram(name string) *PaginatedProgramsResponse {
	url := CLOUD_URL + "programs"
	if len(name) > 0 {
		url = url + "?name=" + name
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		klog.Error("Error creating Get program request; err ", err)
		return nil
	}

	respBody, statusCode := sendReq(req)
	if statusCode != 200 {
		klog.Error("Retrieving program not successful, status code ", statusCode)
		return nil
	}

	pi := PaginatedProgramsResponse{}
	json.Unmarshal(respBody, &pi)
	return &pi
}

// Submit job
func submitJob(request JobRunParams) *JobSubmitResult {
	url := CLOUD_URL + "jobs"
	data, err := json.Marshal(request)
	if err != nil {
		klog.Error("Error marshalling job request; err ", err)
		return nil
	}
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		klog.Error("Error creating submit job request; err ", err)
		return nil
	}

	respBody, statusCode := sendReq(req)
	if statusCode != 200 {
		klog.Error("Submitting not successful, status code ", statusCode)
		return nil
	}

	res := JobSubmitResult{}
	json.Unmarshal(respBody, &res)
	return &res
}

// Get job status
func getJobState(jobID string) *JobStatusResult {

	url := CLOUD_URL + "jobs/" + jobID
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		klog.Error("Error creating Get job request; err ", err)
		return nil
	}

	respBody, statusCode := sendReq(req)
	if statusCode != 200 {
		klog.Error("Retrieving job not successful, status code ", statusCode)
		return nil
	}
	result := JobStatusResult{}
	json.Unmarshal(respBody, &result)
	return &result
}

// Get job results
func getJobResults(jobID string) string {

	url := CLOUD_URL + "jobs/" + jobID + "/results"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		klog.Error("Error creating Get job result request; err ", err)
		return ""
	}

	respBody, statusCode := sendReq(req)
	if statusCode != 200 {
		klog.Error("Retrieving job results is not successful, status code ", statusCode)
		return ""
	}

	return string(respBody)
}

// Get job results
func getJobInterimResults(jobID string) string {

	url := CLOUD_URL + "jobs/" + jobID + "/interim_results"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		klog.Error("Error creating Get job interim result request; err ", err)
		return ""
	}

	respBody, statusCode := sendReq(req)
	if statusCode != 200 {
		klog.Error("Retrieving job interim results is not successful, status code ", statusCode)
		return ""
	}

	return string(respBody)
}

// Get job results
func getJobLogs(jobID string) string {

	url := CLOUD_URL + "jobs/" + jobID + "/logs"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		klog.Error("Error creating Get job logs request; err ", err)
		return ""
	}

	respBody, statusCode := sendReq(req)
	if statusCode != 200 {
		klog.Error("Retrieving job logs is not successful, status code ", statusCode)
		return ""
	}

	return string(respBody)
}

// Delete job
func deleteJob(jobID string) {
	url := CLOUD_URL + "jobs/" + jobID
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		klog.Error("Error creating delete job request; err ", err)
		return
	}

	_, statusCode := sendReq(req)
	if statusCode != 204 {
		klog.Error("Deleting job not successful, status code ", statusCode)
		return
	}
}

// Delete program
func deleteProgram(programID string) {
	url := CLOUD_URL + "programs/" + programID
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		klog.Error("Error creating delete program request; err ", err)
		return
	}

	_, statusCode := sendReq(req)
	if statusCode != 204 {
		klog.Error("Deleting program not successful, status code ", statusCode)
		return
	}
}

// Add program
func addProgram(request ProgramSubmissionRequest) *Program {
	url := CLOUD_URL + "programs"
	data, err := json.Marshal(request)
	if err != nil {
		klog.Error("Error marshalling program request; err ", err)
		return nil
	}
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		klog.Error("Error creating create program request; err ", err)
		return nil
	}

	respBody, statusCode := sendReq(req)
	if statusCode != 201 {
		klog.Error("Creating program was not successful, status code ", statusCode)
		return nil
	}
	res := Program{}
	json.Unmarshal(respBody, &res)
	return &res
}

// Existing program
func existingProgram() {

	// Get program
	program := getProgram("hello-world")
	klog.Info("Retrieved ", len(program.Programs), " programs")
	for _, pm := range program.Programs {
		klog.Info("program ", pm.Name, " id:", pm.ID, " cost:", pm.Cost, " data:", pm.Data)
		klog.Info("Backend requirements ", pm.Spec.BackendRequirements)
		klog.Info("Parameters ", pm.Spec.Parameters.Properties, " schema ", pm.Spec.Parameters.Schema, " required ", pm.Spec.Parameters.Required)
		klog.Info("Return values ", pm.Spec.ReturnValues.Properties, " schema ", pm.Spec.ReturnValues.Schema)
		klog.Info("Interim results ", pm.Spec.InterimResults.Properties, " schema ", pm.Spec.InterimResults.Schema)
	}
	klog.Info("Program is ", string(program.Programs[0].Data))

	// Submit job
	jobRequest := JobRunParams{
		ProgramID: program.Programs[0].ID,
		Backend:   "",
		Params:    map[string]interface{}{"iterations": 5},
	}
	klog.Info("Job request", jobRequest)
	submissionResult := submitJob(jobRequest)
	klog.Info("Submitted job", submissionResult)

	// Get Job status
	status := "Queued"
	for status != "Completed" && status != "Cancelled" && status != "Cancelled - Ran too long" && status != "Failed" {
		statusResult := getJobState(submissionResult.ID)
		klog.Info("Job Status ID:", statusResult.ID, " backend:", statusResult.Backend, " runtime:", statusResult.Runtime,
			" program: ", statusResult.Program.ID, " created:", statusResult.Created, " status:", statusResult.Status)
		time.Sleep(10 * time.Second)
		status = statusResult.Status
	}

	// Get Job results
	jobResults := getJobResults(submissionResult.ID)
	klog.Info("Execution results ", jobResults)

	// Get Job interim results
	jobInterimResults := getJobInterimResults(submissionResult.ID)
	klog.Info("Interim execution results ", jobInterimResults)

	// Get Job results
	jobLogs := getJobLogs(submissionResult.ID)
	klog.Info("log results ", jobLogs)

	// Delete job
	deleteJob(submissionResult.ID)
}

// Upload program
func uploadProgram() {
	file, _ := ioutil.ReadFile("test/program.json")
	definedProgram := ProgramDefinition{}
	_ = json.Unmarshal([]byte(file), &definedProgram)
	klog.Info("Uploaded program ", definedProgram.Name, " with cost ", definedProgram.Cost)
	for key, pm := range definedProgram.Spec.Parameters.Properties {
		klog.Info("Parameter ", key, " description ", pm.Description, " type ", pm.Type, " default ", pm.Default)
	}
	for key, pm := range definedProgram.Spec.ReturnValues.Properties {
		klog.Info("Return value ", key, " description ", pm.Description, " type ", pm.Type, " default ", pm.Default)
	}
	for key, pm := range definedProgram.Spec.InterimResults.Properties {
		klog.Info("Intermediate result ", key, " description ", pm.Description, " type ", pm.Type, " default ", pm.Default)
	}

	file, _ = ioutil.ReadFile("test/parameters.json")
	parameters := ParameterDefinition{}
	_ = json.Unmarshal([]byte(file), &parameters)
	for key, pm := range parameters.Params {
		klog.Info("parameter ", key, " value ", pm)
	}

	// Create program
	submissionRequest := ProgramSubmissionRequest{
		Name:        definedProgram.Name,
		Data:        []byte(definedProgram.Data),
		Cost:        definedProgram.Cost,
		Description: definedProgram.Description,
		Spec:        definedProgram.Spec,
		IsPublic:    definedProgram.IsPublic,
	}
	program := addProgram(submissionRequest)
	klog.Info("Created ne program ", program.Name, " with ID ", program.ID)

	// Submit job
	jobRequest := JobRunParams{
		ProgramID: program.ID,
		Backend:   "",
		//		Params: map[string]interface{}{"hamiltonian": [][]interface{}{{1, "XX"}, {1, "YY"}, {1, "ZZ"}}, "optimizer_config": map[string]interface{}{"maxiter": 10}},
		Params: parameters.Params,
	}
	klog.Info("Job request", jobRequest)
	submissionResult := submitJob(jobRequest)
	klog.Info("Submitted job", submissionResult)

	// Get Job status
	status := ""
	for status != "Completed" && status != "Cancelled" && status != "Cancelled - Ran too long" && status != "Failed" {
		statusResult := getJobState(submissionResult.ID)
		klog.Info("Job Status ID:", statusResult.ID, " backend:", statusResult.Backend, " runtime:", statusResult.Runtime,
			" program: ", statusResult.Program.ID, " created:", statusResult.Created, " status:", statusResult.Status)
		time.Sleep(20 * time.Second)
		status = statusResult.Status
	}

	// Get Job results
	jobResults := getJobResults(submissionResult.ID)
	klog.Info("Execution results ", jobResults)

	// Get Job interim results
	jobInterimResults := getJobInterimResults(submissionResult.ID)
	klog.Info("Interim execution results ", jobInterimResults)

	// Get Job logs
	jobLogs := getJobLogs(submissionResult.ID)
	klog.Info("log results ", jobLogs)

	// Delete job
	deleteJob(submissionResult.ID)
	klog.Info("Job ", submissionResult.ID, " is deleted")

	// Delete program
	deleteProgram(program.ID)
	klog.Info("Program ", program.ID, " is deleted")

}

// Main method
func main() {

	// Create HTTP client
	client = http.Client{Timeout: time.Duration(5) * time.Second}
	//	uploadProgram()
	existingProgram()
}
