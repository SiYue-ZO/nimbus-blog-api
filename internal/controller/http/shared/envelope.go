package shared

type Envelope struct {
	Code    string      `json:"code" example:"0000"`
	Message string      `json:"message" example:"ok"`
	Data    interface{} `json:"data,omitempty"`
}
