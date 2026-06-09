package model

type CreateWorkflowRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=255"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

type UpdateWorkflowRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
}
