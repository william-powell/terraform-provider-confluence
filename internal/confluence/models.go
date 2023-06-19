package confluence

import "time"

type ContentDetail struct {
	Id                 int64                `json:"id"`
	Title              string               `json:"title"`
	Version            ContentDetailVersion `json:"version"`
	SpaceId            int64                `json:"spaceId"`
	CreatedAt          time.Time            `json:"createdAt"`
	Body               ContentOperationBody `json:"body"`
	ParentContentId    int64                `json:"parentId"`
	ResponseStatusCode int
	ResponseStatus     string
	ResponseBody       string
}

type ContentDetailVersion struct {
	Number    int64     `json:"number"`
	CreatedAt time.Time `json:"createdAt"`
}

type ContentUpdateOperationRequest struct {
	Id      int64                   `json:"id"`
	Status  string                  `json:"status"`
	Title   string                  `json:"title"`
	SpaceId int64                   `json:"spaceId"`
	Body    ContentOperationBody    `json:"body"`
	Version ContentOperationVersion `json:"version"`
}

type ContentNewOperationRequest struct {
	Status          string                  `json:"status"`
	Title           string                  `json:"title"`
	SpaceId         int64                   `json:"spaceId"`
	Body            ContentOperationBody    `json:"body"`
	Version         ContentOperationVersion `json:"version"`
	ParentContentId int64                   `json:"parentId"`
}

type ContentOperationVersion struct {
	Number int64 `json:"number"`
}

type ContentOperationBody struct {
	Storage ContentOperationBodyStorage `json:"storage"`
}

type ContentOperationBodyStorage struct {
	Value          string `json:"value"`
	Representation string `json:"representation"`
}
