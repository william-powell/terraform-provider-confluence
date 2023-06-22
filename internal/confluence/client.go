package confluence

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"golang.org/x/net/html"
)

const (
	contentDetailBaseUrlFormat string = "%s/wiki/api/v2/pages/%d?body-format=storage"
	// Delete Version not Support in v1 API yet.
	contentVersionBaseUrlFormat string = "%s/wiki/rest/api/content/%d/version/1"
	updateDeleteContentBaseUrl  string = "%s/wiki/api/v2/pages/%d"
	newContentBaseUrlFormat     string = "%s/wiki/api/v2/pages"
)

type Config struct {
	baseUrl  string
	userName string
	apiKey   string
}

func NewConfig(baseUrl string, userName string, apiKey string) *Config {
	return &Config{baseUrl: baseUrl, userName: userName, apiKey: apiKey}
}

func CreateNewPage(config Config, parentContentId int64, title string, body string) (ContentDetail, error) {
	parentContent, err := GetContentDetailById(config, parentContentId)

	if err != nil {
		return ContentDetail{}, err
	}

	newPageRequest, err := NewNewOperationRequest(title, parentContent.SpaceId, body, parentContentId)

	if err != nil {
		return ContentDetail{}, err
	}

	newPageRequestJson, err := json.Marshal(newPageRequest)

	if err != nil {
		return ContentDetail{}, err
	}

	bodyReader := bytes.NewReader(newPageRequestJson)

	requestUrl := fmt.Sprintf(newContentBaseUrlFormat, config.baseUrl)

	auth := basicAuth(config.userName, config.apiKey)

	client := &http.Client{}

	newReq, err := http.NewRequest("POST", requestUrl, bodyReader)

	if err != nil {
		return ContentDetail{}, err
	}

	newReq.Header.Add("Authorization", "Basic "+auth)
	newReq.Header.Add("Content-Type", "application/json")
	newResp, err := client.Do(newReq)

	if err != nil {
		return ContentDetail{}, err
	}

	if newResp.StatusCode != 200 {
		body, err := ioutil.ReadAll(newResp.Body)

		_ = err

		return ContentDetail{}, fmt.Errorf("Error Updating content: Status: %d, Reason: %s - Body: %s", newResp.StatusCode, newResp.Status, body)
	}

	responseData, err := ioutil.ReadAll(newResp.Body)

	if err != nil {
		return ContentDetail{}, err
	}

	var contentDetail ContentDetail
	err = json.Unmarshal(responseData, &contentDetail)

	if err != nil {
		return ContentDetail{}, err
	}

	return GetContentDetailById(config, contentDetail.Id)
}

func NewNewOperationRequest(title string, spaceId int64, body string, parentContentId int64) (ContentNewOperationRequest, error) {
	htmlErr := isValidHTML(body)

	if htmlErr != nil {
		return ContentNewOperationRequest{}, htmlErr
	}

	request := ContentNewOperationRequest{}

	request.Status = "current"
	request.Title = title
	request.SpaceId = spaceId
	request.Body.Storage.Representation = "storage"
	request.Body.Storage.Value = body
	request.ParentContentId = parentContentId

	return request, nil
}

func GetContentDetailById(config Config, contentId int64) (ContentDetail, error) {
	auth := basicAuth(config.userName, config.apiKey)

	requestUrl := fmt.Sprintf(contentDetailBaseUrlFormat, config.baseUrl, contentId)

	client := &http.Client{}

	req, err := http.NewRequest("GET", requestUrl, nil)
	if err != nil {
		return ContentDetail{}, err
	}

	req.Header.Add("Authorization", "Basic "+auth)
	resp, err := client.Do(req)

	if err != nil {
		return ContentDetail{}, err
	}

	responseData, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return ContentDetail{}, err
	}

	var contentDetail ContentDetail
	err = json.Unmarshal(responseData, &contentDetail)

	if err != nil {
		log.Fatal(err)
		return ContentDetail{}, err
	}

	contentDetail.ResponseStatusCode = resp.StatusCode
	contentDetail.ResponseStatus = resp.Status

	return contentDetail, err
}

func UpdateContentById(config Config, contentId int64, body string, removePreviousVersions bool) (ContentDetail, error) {
	contentDetail, err := GetContentDetailById(config, contentId)

	if err != nil {
		log.Fatal(err)
		return ContentDetail{}, err
	}

	updateRequest, err := NewUpdateOperationRequest(contentDetail, body)

	if err != nil {
		log.Fatal(err)
		return ContentDetail{}, err
	}

	updateRequestJson, err := json.Marshal(updateRequest)

	if err != nil {
		log.Fatal(err)
		return ContentDetail{}, err
	}

	bodyReader := bytes.NewReader(updateRequestJson)

	requestUrl := fmt.Sprintf(updateDeleteContentBaseUrl, config.baseUrl, contentId)

	auth := basicAuth(config.userName, config.apiKey)

	client := &http.Client{}

	upReq, err := http.NewRequest("PUT", requestUrl, bodyReader)

	if err != nil {
		return ContentDetail{}, err
	}

	upReq.Header.Add("Authorization", "Basic "+auth)
	upReq.Header.Add("Content-Type", "application/json")
	upResp, err := client.Do(upReq)

	if err != nil {
		return ContentDetail{}, err
	}

	if upResp.StatusCode != 200 {
		return ContentDetail{}, fmt.Errorf("Error Updating content: Status: %d, Reason: %s", upResp.StatusCode, upResp.Status)
	}

	if removePreviousVersions {
		err = RemovePreviousVersions(config, contentId, 1)
		if err != nil {
			log.Fatal(err)
		}
	}

	return GetContentDetailById(config, contentId)
}

func NewUpdateOperationRequest(detail ContentDetail, body string) (ContentUpdateOperationRequest, error) {
	htmlErr := isValidHTML(body)

	if htmlErr != nil {
		return ContentUpdateOperationRequest{}, htmlErr
	}

	request := ContentUpdateOperationRequest{}

	request.Id = detail.Id
	request.Status = "current"
	request.Title = detail.Title
	request.SpaceId = detail.SpaceId
	request.Body.Storage.Representation = "storage"
	request.Body.Storage.Value = body
	nextVersion := detail.Version.Number + 1
	request.Version.Number = nextVersion

	return request, nil
}

func RemovePreviousVersions(config Config, contentId int64, numberOfVersionsToKeep int64) error {
	if numberOfVersionsToKeep < 1 {
		fmt.Println("Must keep at least 1 version")
		os.Exit(1)
	}

	contentDetail, err := GetContentDetailById(config, contentId)

	if err != nil {
		log.Fatal(err)
		return err
	}

	versionsToDelete := contentDetail.Version.Number - numberOfVersionsToKeep
	deleteRequestUrl := fmt.Sprintf(contentVersionBaseUrlFormat, config.baseUrl, contentId)
	auth := basicAuth(config.userName, config.apiKey)

	for {
		if versionsToDelete <= 0 {
			break
		}

		fmt.Printf("Deleting version: %d - %s\n", versionsToDelete, deleteRequestUrl)
		client := &http.Client{}

		deleteReq, err := http.NewRequest("DELETE", deleteRequestUrl, nil)

		if err != nil {
			log.Fatal(err)
		}

		deleteReq.Header.Add("Authorization", "Basic "+auth)
		deleteResponse, err := client.Do(deleteReq)

		if err != nil {
			log.Fatal(err)
		}

		if deleteResponse.StatusCode != 204 {
			log.Printf("Unable to delete version. - Code: %d - Reason: %s\n", deleteResponse.StatusCode, deleteResponse.Status)
		}

		versionsToDelete = versionsToDelete - 1
	}

	return nil
}

func DeleteContentById(config Config, contentId int64) (http.Response, error) {
	requestUrl := fmt.Sprintf(updateDeleteContentBaseUrl, config.baseUrl, contentId)

	auth := basicAuth(config.userName, config.apiKey)

	client := &http.Client{}

	upReq, err := http.NewRequest("DELETE", requestUrl, nil)

	if err != nil {
		log.Fatal(err)
	}

	upReq.Header.Add("Authorization", "Basic "+auth)
	upReq.Header.Add("Content-Type", "application/json")
	upResp, err := client.Do(upReq)

	if err != nil {
		return *upResp, err
	}

	if upResp.StatusCode != 200 {
		return *upResp, fmt.Errorf("Error Deleting content: Status: %d, Reason: %s", upResp.StatusCode, upResp.Status)
	}

	return *upResp, nil
}

func isValidHTML(htmlStr string) error {
	r := strings.NewReader(htmlStr)
	z := html.NewTokenizer(r)
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			err := z.Err()
			if err == io.EOF {
				// Not an error, we're done and it's valid!
				return nil
			}
			return err
		}
	}
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}
