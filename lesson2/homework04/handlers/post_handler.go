package handlers

import (
	"homework0401/models"
	"homework0401/services"

	"github.com/gin-gonic/gin"
)

type PostHandler struct {
	postService *services.PostService
	jwtSecret   []byte
}

func NewPostHandler(postService *services.PostService, jwtSecret []byte) *PostHandler {
	return &PostHandler{
		postService: postService,
		jwtSecret:   jwtSecret,
	}
}

func (h *PostHandler) CreatePost(c *gin.Context) {
	var req models.CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
}
