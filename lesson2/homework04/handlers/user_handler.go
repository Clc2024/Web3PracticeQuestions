package handlers

import (
	"homework0401/models"
	"homework0401/services"
	"homework0401/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserHandler struct {
	userService *services.UserService
	jwtSecret   []byte
}

func NewUserHandler(userService *services.UserService, jwtSecret []byte) *UserHandler {
	return &UserHandler{
		userService: userService,
		jwtSecret:   jwtSecret,
	}
}

func (h *UserHandler) Register(c *gin.Context) {
	db := c.MustGet("blog").(*gorm.DB)
	var req models.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, parseValidationErrors(err))
		return
	}

	user, err := h.userService.CreateUser(db, req)
	if err != nil {
		utils.HandleError(c, err)
		return
	}

	utils.Success(c, models.UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	})
}

func (h *UserHandler) Login(c *gin.Context) {
	db := c.MustGet("blog").(*gorm.DB)
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, parseValidationErrors(err))
		return
	}

	user, err := h.userService.Authenticate(db, req.Username, req.Password)
	if err != nil {
		utils.HandleError(c, err)
		return
	}

	token, err := utils.GenerateToken(h.jwtSecret, user.ID, user.Username)
	if err != nil {
		utils.HandleError(c, err)
		return
	}

	utils.Success(c, gin.H{
		"token": token,
		"user": models.UserResponse{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			CreatedAt: user.CreatedAt,
		},
	})
}

func (h *UserHandler) GetProfile(c *gin.Context) {
	db := c.MustGet("blog").(*gorm.DB)
	userID, exists := c.Get("userID")
	if !exists {
		utils.Error(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	user, err := h.userService.GetUserByID(db, userID.(uint))
	if err != nil {
		utils.HandleError(c, err)
		return
	}

	utils.Success(c, models.UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	})
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	db := c.MustGet("blog").(*gorm.DB)
	userID, exists := c.Get("userID")
	if !exists {
		utils.Error(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, parseValidationErrors(err))
		return
	}

	user, err := h.userService.UpdateUser(db, userID.(uint), req)
	if err != nil {
		utils.HandleError(c, err)
		return
	}

	utils.Success(c, models.UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	})
}

func parseValidationErrors(err error) map[string]string {
	errors := make(map[string]string)
	// 简化处理，实际应该解析 binding 错误
	errors["general"] = err.Error()
	return errors
}
