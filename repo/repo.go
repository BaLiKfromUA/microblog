package repo

import "microblog/model"

type Repository interface {
	CreatePost(id model.UserId, post model.Post) (model.Post, error)
	GetPostById(id model.PostId) (model.Post, error)
	GetPosts(id model.UserId, page model.PageToken, size int) ([]model.Post, model.PageToken, error)
	clear()
}
