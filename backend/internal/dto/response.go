package dto

type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}
type PageQuery struct {
	Page     int `form:"page"`
	PageSize int `form:"page_size"`
}
