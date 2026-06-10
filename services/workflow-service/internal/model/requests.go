package model

type CreateWorkflowRequest struct {
	Name            string `json:"name" binding:"required,min=1,max=255"`
	Description     string `json:"description"`
	Category        string `json:"category"`
	Priority        string `json:"priority"`
	RequesterLabel  string `json:"requester_label"`
}

type UpdateWorkflowRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Category    *string `json:"category"`
	Priority    *string `json:"priority"`
}

type ReviewWorkflowRequest struct {
	Comment     string `json:"comment"`
	ActorLabel  string `json:"actor_label"`
}
