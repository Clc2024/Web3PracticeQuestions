package models

import "gorm.io/gorm"

type Post struct {
	gorm.Model
	Title   string `gorm:"column:title;NOT NULL"`
	Desc    string `gorm:"column:desc"`
	Content string `gorm:"column:content;NOT NULL"`
	UserID  int    `gorm:"column:user_id;NOT NULL"`
	Tags    string `gorm:"column:tags"`
}

type CreatePostRequest struct {
	Title   string `gorm:"column:title;NOT NULL"`
	Desc    string `gorm:"column:desc"`
	Content string `gorm:"column:content;NOT NULL"`
	UserID  int    `gorm:"column:user_id;NOT NULL"`
	Tags    string `gorm:"column:tags"`
}

type UpdatePostRequest struct {
	Title   string `gorm:"column:title;NOT NULL"`
	Desc    string `gorm:"column:desc"`
	Content string `gorm:"column:content;NOT NULL"`
	UserID  int    `gorm:"column:user_id;NOT NULL"`
	Tags    string `gorm:"column:tags"`
}
