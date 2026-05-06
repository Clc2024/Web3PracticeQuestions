package models

import "gorm.io/gorm"

type Comment struct {
	gorm.Model
	PostID         int    `gorm:"column:post_id;NOT NULL"`
	CommentContent string `gorm:"column:comment_content;NOT NULL"`
	Email          string `gorm:"column:email;NOT NULL"`
}
