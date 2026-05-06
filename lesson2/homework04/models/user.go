package models

import (
	"time"

	"gorm.io/gorm"
)

// User
type User struct {
	gorm.Model
	Username string `gorm:"column:username;NOT NULL"`
	Password string `gorm:"column:password;NOT NULL"`
	Email    string `gorm:"column:email"`
}

type CreateUserRequest struct {
	Username string `json:"username" binding:"required,min=3,max=20"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type UpdateUserRequest struct {
	Email string `json:"email" binding:"omitempty,email"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type UserResponse struct {
	ID        uint      `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}
