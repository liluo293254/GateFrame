package model

type CreateFileRequest struct {
	Filename      string `json:"filename" binding:"required,min=1,max=255"`
	ContentType   string `json:"content_type"`
	ContentBase64 string `json:"content_base64" binding:"required"`
}
