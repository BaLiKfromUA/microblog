package repo

import (
	"context"
	"microblog/internal/model"
)

type Repository interface {
	CreatePost(ctx context.Context, id model.UserId, post model.Post) (model.Post, error)
	EditPost(ctx context.Context, id model.UserId, post model.Post) (model.Post, error)
	GetPostById(ctx context.Context, id model.PostId) (model.Post, error)
	GetPosts(ctx context.Context, id model.UserId, page model.PageToken, size int) ([]model.Post, model.PageToken, error)
	Subscribe(ctx context.Context, from model.UserId, to model.UserId) error
	GetSubscriptions(ctx context.Context, id model.UserId) ([]model.UserId, error)
	GetSubscribers(ctx context.Context, id model.UserId) ([]model.UserId, error)
	GetFeed(ctx context.Context, id model.UserId, page model.PageToken, size int) ([]model.FeedMetadataDocument, model.PageToken, error)
	AddPostToFeed(ctx context.Context, post model.FeedMetadataDocument) error
}
