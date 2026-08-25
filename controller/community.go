package controller

import (
	"net/http"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

type CreatePostRequest struct {
	Content  string `json:"content"`
	ImageUrl string `json:"image_url"`
	MaskId   string `json:"mask_id"`
	ParentId *int   `json:"parent_id"`
}

func GetPosts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	maskId := c.Query("mask_id")
	userIdStr := c.Query("user_id")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}

	var posts []model.Post
	query := model.DB.Where("status = 1 AND parent_id IS NULL")

	if maskId != "" {
		query = query.Where("mask_id = ?", maskId)
	}
	if userIdStr != "" {
		query = query.Where("user_id = ?", userIdStr)
	}

	var total int64
	query.Model(&model.Post{}).Count(&total)

	err := query.Preload("User").
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&posts).Error
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items":     posts,
			"page":      page,
			"page_size": pageSize,
			"total":     total,
		},
	})
}

func CreatePost(c *gin.Context) {
	var req CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	if len(req.Content) == 0 || len(req.Content) > 2000 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "内容长度需在1-2000字之间"})
		return
	}

	userId := c.GetInt("id")

	post := model.Post{
		UserId:   userId,
		Content:  req.Content,
		ImageUrl: req.ImageUrl,
		MaskId:   req.MaskId,
		ParentId: req.ParentId,
		Status:   1,
	}

	if req.ParentId != nil {
		var parent model.Post
		if err := model.DB.First(&parent, *req.ParentId).Error; err == nil {
			if parent.RootId != nil {
				post.RootId = parent.RootId
			} else {
				post.RootId = req.ParentId
			}
			model.DB.Model(&parent).UpdateColumn("reply_count", parent.ReplyCount+1)
		}
	}

	if err := model.DB.Create(&post).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	model.DB.Preload("User").First(&post, post.Id)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": post})
}

func GetPost(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的ID"})
		return
	}

	var post model.Post
	if err := model.DB.Preload("User").First(&post, id).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "帖子不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": post})
}

func DeletePost(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的ID"})
		return
	}

	userId := c.GetInt("id")
	var post model.Post
	if err := model.DB.First(&post, id).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "帖子不存在"})
		return
	}
	if post.UserId != userId && c.GetInt("role") < common.RoleAdminUser {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无权删除"})
		return
	}

	model.DB.Delete(&post)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "已删除"})
}

func GetPostReplies(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的ID"})
		return
	}

	var replies []model.Post
	model.DB.Where("parent_id = ? AND status = 1", id).
		Preload("User").
		Order("created_at ASC").
		Find(&replies)

	c.JSON(http.StatusOK, gin.H{"success": true, "data": replies})
}

func ToggleLike(c *gin.Context) {
	postId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的ID"})
		return
	}

	userId := c.GetInt("id")
	var existing model.Like

	if model.DB.Where("user_id = ? AND post_id = ?", userId, postId).First(&existing).Error == nil {
		model.DB.Delete(&existing)
		var post model.Post
		model.DB.First(&post, postId)
		newCount := post.LikeCount - 1
		if newCount < 0 {
			newCount = 0
		}
		model.DB.Model(&post).UpdateColumn("like_count", newCount)
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"liked": false, "like_count": newCount}})
		return
	}

	like := model.Like{UserId: userId, PostId: postId, CreatedAt: time.Now()}
	model.DB.Create(&like)
	var post model.Post
	model.DB.First(&post, postId)
	model.DB.Model(&post).UpdateColumn("like_count", post.LikeCount+1)

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"liked": true, "like_count": post.LikeCount + 1}})
}

func GetUserProfile(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的ID"})
		return
	}

	var user model.User
	if err := model.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "用户不存在"})
		return
	}

	var postCount int64
	model.DB.Model(&model.Post{}).Where("user_id = ? AND status = 1", id).Count(&postCount)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"id":           user.Id,
			"username":     user.Username,
			"display_name": user.DisplayName,
			"role":         user.Role,
			"post_count":   postCount,
		},
	})
}

func GetUserPosts(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的ID"})
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}

	var posts []model.Post
	model.DB.Where("user_id = ? AND status = 1 AND parent_id IS NULL", id).
		Preload("User").
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&posts)

	c.JSON(http.StatusOK, gin.H{"success": true, "data": posts})
}

var builtinMasks = []model.CommunityMask{
	{
		Id:     "huashan",
		Name:   "⚔️ 大模型华山论剑群",
		Avatar: "2694-fe0f",
		Model:  "deepseek-reasoner",
		Lang:   "cn",
	},
	{
		Id:     "suanming",
		Name:   "🔮 赛博周易推演天师",
		Avatar: "1f52e",
		Model:  "deepseek-reasoner",
		Lang:   "cn",
	},
	{
		Id:     "lunwen",
		Name:   "📝 论文三联复式调优专家",
		Avatar: "1f4dd",
		Model:  "deepseek-reasoner",
		Lang:   "cn",
	},
	{
		Id:     "wenan",
		Name:   "📢 情境自适应爆款文案精排器",
		Avatar: "1f4e2",
		Model:  "deepseek-reasoner",
		Lang:   "cn",
	},
}

func GetCommunityMasks(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true, "data": builtinMasks})
}

func GetMaskPosts(c *gin.Context) {
	maskId := c.Param("id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}

	var posts []model.Post
	model.DB.Where("mask_id = ? AND status = 1", maskId).
		Preload("User").
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&posts)

	c.JSON(http.StatusOK, gin.H{"success": true, "data": posts})
}
