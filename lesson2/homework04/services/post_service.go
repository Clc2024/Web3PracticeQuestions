package services

import (
	"homework0401/models"
	"homework0401/utils"

	"gorm.io/gorm"
)

type PostService struct {
	db *gorm.DB
}

func NewPostService(db *gorm.DB) *PostService {
	// 检查文章标题是否已存在
	return &PostService{db: db}
}

// CreatePostService
func (s *PostService) CreatePost(db *gorm.DB, req models.CreatePostRequest) (*models.Post, error) {
	var post models.Post

	// 检查文章标题是否已存在
	if err := db.Where("Title = ?", req.Title).First(&post).Error; err == nil {
		return nil, utils.NewAppError(409, "Title already exists")
	}

	posts := models.Post{
		Title:   req.Title,
		Desc:    req.Desc,
		Content: req.Content,
		UserID:  req.UserID,
		Tags:    req.Tags,
	}

	if err := db.Create(&posts).Error; err != nil {
		return nil, err
	}

	return &posts, nil
}

// GetPostsListService
func (s *PostService) GetPostsListService(db *gorm.DB) (posts []*models.Post) {
	db.Find(&posts)
	return posts
}

func (s *PostService) GetPostsByID(db *gorm.DB, id int) (*models.Post, error) {
	var post models.Post
	if err := db.Where("id = ?", id).First(&post); err != nil {
		return nil, utils.NewAppError(404, "User not found")
	}
	return &post, nil
}

// UpdatePostById
func (s *PostService) UpdatePostById(db *gorm.DB, post_id int, req models.UpdatePostRequest) (*models.Post, error) {

	post, err := s.GetPostsByID(db, post_id)
	if err != nil {
		return nil, err
	}
	// 如果更新邮箱，检查是否已存在
	if req.UserID == post.UserID {
		post.Title = req.Title
		post.Desc = req.Desc
		post.Content = req.Content
		post.Tags = req.Tags
	}

	if err := s.db.Save(post).Error; err != nil {
		return nil, err
	}

	return post, nil
}

// DeletePostById
func (s *PostService) DeletePostById(db *gorm.DB, id int) (flag int, err error) {
	//db := core.GetDB()

	err = db.First(&models.Post{}, id).Error
	if err != nil {
		return 0, nil
	}

	err = db.Delete(&models.Post{}, id).Error
	if err != nil {
		return 2, err
	}

	return 1, nil
}
