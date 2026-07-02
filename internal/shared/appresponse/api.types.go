package appresponse

type (
	ErrorInfo struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
)
