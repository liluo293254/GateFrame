package model

type CreateDocumentRequest struct {
	Title   string `json:"title" binding:"required,min=1,max=255"`
	Content string `json:"content"`
}
