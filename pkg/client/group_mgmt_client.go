// Copyright 2020 Hewlett Packard Enterprise Development LP

package client

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/hpe-storage/nimble-golang-sdk/pkg/client/v1/nimbleos"
	"github.com/hpe-storage/nimble-golang-sdk/pkg/param"
	"strings"
	"time"
)

const (
	groupURIFmt     = "https://%s:5392/%s"
	clientTimeout   = time.Second * 60 // 1 Minute
	maxLoginRetries = 5
	jobTimeout      = time.Second * 300 // 5 Minute
	jobPollInterval = 5 * time.Second   // Second
	smAsyncJobId    = "SM_async_job_id"
)

// GroupMgmtClient :
type GroupMgmtClient struct {
	URL          string
	Client       *resty.Client
	SessionToken string
	WaitOnJob    bool
	Username     string
	Password     string
}

// DataWrapper is used to represent a generic JSON API payload
type DataWrapper struct {
	Data          interface{} `json:"data"`
	StartRow      *int        `json:"startRow,omitempty"`
	EndRow        *int        `json:"endRow,omitempty"`
	PageSize      *int        `json:"pageSize,omitempty"`
	TotalRows     *int        `json:"totalRows,omitempty"`
	OperationType *string     `json:"operationType,omitempty"`
}

// ErrorResponse is a serializer struct for representing a valid JSON API errors payload.
type ErrorResponse struct {
	Messages []*Message `json:"messages"`
}

// Message is an `Error` implementation as well as an implementation of the JSON API error object.
type Message struct {
	Code      string   `json:"code,omitempty"`
	Text      string   `json:"text,omitempty"`
	Severity  string   `json:"severity,omitempty"`
	Arguments Argument `json:"arguments,omitempty"`
}

type Argument struct {
	JobId string `json:"job_id,omitempty"`
}

// NewClient instantiates a new client to communicate with the Nimble group
func NewClient(ipAddress, username, password, apiVersion string, waitOnJobs bool) (*GroupMgmtClient, error) {
	// Create GroupMgmt Client
	client := resty.New()
	client.SetTLSClientConfig(&tls.Config{
		InsecureSkipVerify: true,
	})
	groupMgmtClient := &GroupMgmtClient{
		URL:       fmt.Sprintf("https://%s:5392/%s", ipAddress, apiVersion),
		Client:    client,
		WaitOnJob: waitOnJobs,
		Username:  username,
		Password:  password,
	}

	// Get session token
	sessionToken, err := groupMgmtClient.login(username, password)
	if err != nil {
		return nil, err
	}
	groupMgmtClient.SessionToken = sessionToken
	return groupMgmtClient, nil
}

// EnableDebug : Enables debug logging of client request/response
func (client *GroupMgmtClient) EnableDebug() {
	client.Client.SetDebug(true)
}

// refreshSessionToken : refresh session token
func (client *GroupMgmtClient) refreshSessionToken() (bool, error) {
	// Invalid credential has empty token
	if client.SessionToken != "" {
		client.SessionToken = ""
		newSessionToken, err := client.login(client.Username, client.Password)
		if err != nil {
			return false, err
		}

		// set new auth token
		client.SessionToken = newSessionToken
		return true, nil
	}
	return false, nil
}

func (client *GroupMgmtClient) login(username, password string) (string, error) {
	// Construct Payload
	appName := "Go sdkv1 client"
	token := &nimbleos.Token{
		Username: &username,
		Password: &password,
		AppName:  &appName,
	}
	token, err := client.GetTokenObjectSet().CreateObject(token)
	if err != nil {
		return "", err
	}
	return *token.SessionToken, err
}

// Post :
func (client *GroupMgmtClient) Post(path string, payload interface{}, respHolder interface{}) (interface{}, error) {

	// build the url
	url := fmt.Sprintf("%s/%s", client.URL, path)
	// post it
	response, err := client.Client.R().
		SetHeader("X-Auth-Token", client.SessionToken).
		SetBody(&DataWrapper{
			Data: payload,
		}).
		Post(url)
	if err != nil {
		return nil, err
	}

	// Unauthorize access, retry POST if Auth token is expired
	if response.StatusCode() == 401 {
		// refresh session token
		isSessionRefreshed, err := client.refreshSessionToken()
		if err != nil {
			return nil, err
		}
		if isSessionRefreshed {
			// retry the ops
			return client.Post(path, payload, respHolder)
		}
		return nil,processError(response.StatusCode(), response.Body())
	}
	return processResponse(client, response, path, respHolder)
}

// Put
func (client *GroupMgmtClient) Put(path, id string, payload interface{}, respHolder interface{}) (interface{}, error) {
	// build the url
	url := fmt.Sprintf("%s/%s/%s", client.URL, path, id)
	// Put it
	response, err := client.Client.R().
		SetHeader("X-Auth-Token", client.SessionToken).
		SetBody(&DataWrapper{
			Data: payload,
		}).
		Put(url)

	if err != nil {
		return nil, err
	}
	// Unauthorize access, retry PUT if Auth token is expired
	if response.StatusCode() == 401 {
		// refresh session token
		isSessionRefreshed, err := client.refreshSessionToken()
		if err != nil {
			return nil, err
		}
		if isSessionRefreshed {
			// retry the ops
			return client.Put(path, id, payload, respHolder)
		}
		return nil,processError(response.StatusCode(), response.Body())
	}
	return processResponse(client, response, path, respHolder)
}

// Get : Only used to get a single object with the given ID
func (client *GroupMgmtClient) Get(path string, id string, respHolder interface{}) (interface{}, error) {
	// build the url
	url := fmt.Sprintf("%s/%s/%s", client.URL, path, id)

	// Get it
	response, err := client.Client.R().
		SetHeader("X-Auth-Token", client.SessionToken).
		Get(url)
	if err != nil {
		return nil, err
	}

	// convert a 404 (not found) to nil response
	if response.StatusCode() == 404 {
		return nil, nil
	}

	// Unauthorize access, retry Get if Auth token is expired
	if response.StatusCode() == 401 {
		// refresh session token
		isSessionRefreshed, err := client.refreshSessionToken()
		if err != nil {
			return nil, err
		}
		if isSessionRefreshed {
			// retry the ops
			return client.Get(path, id, respHolder)
		}
		return nil,processError(response.StatusCode(), response.Body())
	}
	return processResponse(client, response, path, respHolder)
}

// Delete :
func (client *GroupMgmtClient) Delete(path string, id string) error {
	// build the url
	url := fmt.Sprintf("%s/%s/%s", client.URL, path, id)
	// delete it
	response, err := client.Client.R().
		SetHeader("X-Auth-Token", client.SessionToken).
		Delete(url)
	if err != nil {
		return err
	}

	// Unauthorize access, retry Delete if Auth token is expired
	if response.StatusCode() == 401 {
		// refresh session token
		isSessionRefreshed, err := client.refreshSessionToken()
		if err != nil {
			return err
		}
		if isSessionRefreshed {
			// retry the ops
			return client.Delete(path, id)
		}
		return processError(response.StatusCode(), response.Body())
	}
	_, err = processResponse(client, response, path, nil)
	return err
}

// List without any params
func (client *GroupMgmtClient) List(path string) (interface{}, error) {
	listResp, err := client.ListFromParams(path, nil)
	if err != nil {
		return nil, err
	}
	return listResp, nil
}

// ListFromParams :
func (client *GroupMgmtClient) ListFromParams(path string, params *param.GetParams) (interface{}, error) {

	wrapper, err := client.listGetOrPost(path, params)
	if err != nil {
		return nil, err
	}
	return wrapper, nil
}

func (client *GroupMgmtClient) listGetOrPost(path string, params *param.GetParams) (interface{}, error) {
	if params == nil {
		return client.listGet(path, nil, nil)
	}

	// load the url query parameters
	queryParams, err := params.BuildQueryParts()
	if err != nil {
		return nil, err
	}

	// check if advanced criteria post is required
	if params.Filter != nil {
		simpleMap, _ := params.Filter.AsSimpleMap()
		if simpleMap == nil {
			fetch := "fetch"
			wrapper := &DataWrapper{
				Data:          params.Filter,
				StartRow:      params.Page.StartRow,
				EndRow:        params.Page.EndRow,
				OperationType: &fetch,
			}
			// complex filter, need to POST it
			postResp, err := client.listPost(path, wrapper, queryParams, params)
			if err != nil {
				return nil, err
			}
			return postResp, nil
		} else {
			// get request
			getResp, err := client.listGet(path, queryParams, params)
			if err != nil {
				return nil, err
			}
			return getResp, nil
		}
	} else {
		// get request
		getResp, err := client.listGet(path, queryParams, params)
		if err != nil {
			return nil, err
		}
		return getResp, nil
	}
}

// listPost uses a post request to get all objects on the path using an advanced criteria
func (client *GroupMgmtClient) listPost(
	path string,
	payload *DataWrapper,
	queryParams map[string]string,
	params *param.GetParams,
) (interface{}, error) {
	// build the url
	url := fmt.Sprintf("%s/%s/detail", client.URL, path)
	// Post it
	response, err := client.Client.R().
		SetQueryParams(queryParams).
		SetHeader("X-Auth-Token", client.SessionToken).
		SetBody(payload).
		Post(url)
	if err != nil {
		return nil, err
	}
	// Unauthorize access, retry listPost if Auth token is expired
	if response.StatusCode() == 401 {
		// refresh session token
		isSessionRefreshed, err := client.refreshSessionToken()
		if err != nil {
			return nil, err
		}
		if isSessionRefreshed {
			// retry the ops
			return client.listPost(path, payload, queryParams, params)
		}
		return nil,processError(response.StatusCode(), response.Body())
	}

	if params != nil && params.Page != nil {
		totalRows, err := getTotalRows(response.Body())
		if err != nil {
			return nil, err
		}
		params.Page.TotalRows = totalRows
	}
	return processResponse(client, response, path, nil)
}

// listGet uses a get request to get all objects on the path
func (client *GroupMgmtClient) listGet(
	path string,
	queryParams map[string]string,
	params *param.GetParams,
) (interface{}, error) {
	// build the url
	url := fmt.Sprintf("%s/%s/detail", client.URL, path)

	response, err := client.Client.R().
		SetQueryParams(queryParams).
		SetHeader("X-Auth-Token", client.SessionToken).
		Get(url)
	if err != nil {
		return nil, err
	}

	// Unauthorize access, retry listPost if Auth token is expired
	if response.StatusCode() == 401 {
		// refresh session token
		isSessionRefreshed, err := client.refreshSessionToken()
		if err != nil {
			return nil, err
		}
		if isSessionRefreshed {
			// retry the ops
			return client.listGet(path, queryParams, params)
		}
		return nil,processError(response.StatusCode(), response.Body())
	}

	if params != nil && params.Page != nil {
		totalRows, err := getTotalRows(response.Body())
		if err != nil {
			return nil, err
		}
		params.Page.TotalRows = totalRows
	}
	return processResponse(client, response, path, nil)
}

// unwrapData
func getTotalRows(body []byte) (*int, error) {
	// unmarshal the response
	wrapper := &DataWrapper{}
	err := json.Unmarshal(body, wrapper)
	if err != nil {
		return nil, err
	}
	// return it
	return wrapper.TotalRows, nil
}

// unwrapData
func unwrapData(body []byte, payload interface{}) (interface{}, error) {
	// unmarshal the response
	wrapper := &DataWrapper{
		Data: payload,
	}
	err := json.Unmarshal(body, wrapper)
	if err != nil {
		return nil, err
	}
	// return it
	return wrapper.Data, nil
}

//processResponse
func processResponse(client *GroupMgmtClient, response *resty.Response, path string, respHolder interface{}) (interface{}, error) {

	// process successfull response
	if response.IsSuccess() {
		// http code 202, handle async job
		if response.StatusCode() == 202 {

			id, err := processAsyncResponse(client, response.Body())
			if id != nil {
				// action rpc may contain different path.
				// extract Get uri path from original path.
				newPath := strings.Split(path, "/")
				return client.Get(newPath[0], id.(string), respHolder)
			} else {
				return nil, err
			}
		}
		// process success response

		return unwrapData(response.Body(), respHolder)

	} else {
		// error response
		return nil,processError(response.StatusCode(), response.Body())
	}
}

// process error response
func processError(httpCode int, body []byte) (error) {
	errResp := ""
	wrapper := &ErrorResponse{}
	err := json.Unmarshal(body, wrapper)
	if err != nil {
		return err
	}

	for _, emsg := range wrapper.Messages {
		errResp += fmt.Sprintf("%+v", *emsg)
	}
	return fmt.Errorf("error: http status(%d), messages: %v", httpCode, errResp)
}

//unwrap error response
func unwrapError(body []byte) (string, error) {
	errResp := ""
	wrapper := &ErrorResponse{}
	err := json.Unmarshal(body, wrapper)
	if err != nil {
		return errResp, err
	}

	for _, emsg := range wrapper.Messages {
		errResp += fmt.Sprintf("%+v", *emsg)
	}
	return errResp, nil
}

// processAsyncResponse: process http code 202 response
func processAsyncResponse(client *GroupMgmtClient, body []byte) (interface{}, error) {
	errResp, _ := unwrapError(body)
	if client.WaitOnJob { // check sync flag
		unwrapMessage := &ErrorResponse{}
		err := json.Unmarshal(body, unwrapMessage)
		if err != nil {
			return nil, err
		}
		var jobId string
		for _, msg := range unwrapMessage.Messages {
			if msg.Code == smAsyncJobId {
				jobId = msg.Arguments.JobId
			}
		}
		if len(jobId) == 0 {
			return nil, fmt.Errorf("http response error: status (202), failed to get the job id")
		}

		id, err := waitForJobResult(jobId, client)
		if err != nil {
			return nil, fmt.Errorf("http response error: status (202), messages: %v", err.Error())
		}
		return id, nil
	}
	return nil, fmt.Errorf("http response error: status (202), messages: %v", errResp)
}

//waitForJobResult : it monitors jobId periodically until job completion or timed out
func waitForJobResult(jobId string, client *GroupMgmtClient) (interface{}, error) {

	// Loop over job ids periodically unitl 300 sec timeout or unitl completion of jobs.
	intervalChan := time.Tick(jobPollInterval) // control the fequency of GetObject() API call.
	timeoutChan := time.After(jobTimeout)      // timeout setting, 300 Seconds
	for {
		select {
		case <-intervalChan:

			job, err := client.GetJobObjectSet().GetObject(jobId)
			if err != nil {
				fmt.Println("Warning : failed to %s jobId info, err : %s", jobId, err.Error())
			} else {
				var objectId = *job.ObjectId
				if string(*job.State) == string(*nimbleos.NsJobStatusDone) {
					return objectId, nil
				}
			}

		case <-timeoutChan:
			return nil, fmt.Errorf("waitForJobResult: job with ID %v timed out after %v seconds.", jobId, jobTimeout)
		}
	}
}
