package repo

import (
	"context"
	"microblog/model"
)

type Repository interface {
	CreatePost(ctx context.Context, id model.UserId, post model.Post) (model.Post, error)
	EditPost(ctx context.Context, id model.UserId, post model.Post) (model.Post, error)
	GetPostById(ctx context.Context, id model.PostId) (model.Post, error)
	GetPosts(ctx context.Context, id model.UserId, page model.PageToken, size int) ([]model.Post, model.PageToken, error)
	Clear(ctx context.Context) error
}
