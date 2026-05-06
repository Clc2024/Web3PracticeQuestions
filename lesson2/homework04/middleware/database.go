package middleware

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// WithGormDB 将 gorm.DB 注入到 gin.Context
// handler 中通过 c.MustGet("blog").(*gorm.DB) 来使用
func WithGormDB(db *gorm.DB) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Set("blog", db)
		ctx.Next()
	}
}
