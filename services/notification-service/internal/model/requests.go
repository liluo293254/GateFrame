package model

type CreateNotificationRequest struct {
	Title  string `json:"title" binding:"required,min=1,max=255"`
	Body   string `json:"body"`
	UserID string `json:"user_id"`
}
