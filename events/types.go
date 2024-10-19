package events

type BaseResponse[T any] struct {
	Data    T      `json:"data"`
	Message string `json:"message"`
}

type InsertionResponse struct {
	Created int `json:"created"`
}

type UpdatesResponse struct {
	Updated int `json:"updated"`
}

type DeletesResponse struct {
	Deleted int `json:"deleted"`
}

type ListQuery struct {
	Page    int32  `query:"page"`
	Limit   int32  `query:"limit"`
	OrderBy string `query:"order_by"`
}
