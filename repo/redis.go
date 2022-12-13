package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"log"
	"microblog/model"
	"microblog/utils"
	"os"
	"time"
)

var _ Repository = (*RedisRepository)(nil)

type RedisRepository struct {
	client         *redis.Client
	persistentRepo Repository
}

func NewRedisRepository(repo Repository) Repository {
	url, ok := os.LookupEnv("REDIS_URL")

	if !ok {
		url = "127.0.0.1:6379"
	}

	return &RedisRepository{
		client:         redis.NewClient(&redis.Options{Addr: url}),
		persistentRepo: repo,
	}
}

func (cache *RedisRepository) CreatePost(ctx context.Context, id model.UserId, post model.Post) (model.Post, error) {
	result, err := cache.persistentRepo.CreatePost(ctx, id, post)
	if err == nil {
		serialized, _ := json.Marshal(result)
		cache.client.Set(ctx, utils.CreateRedisKeyForPost(result.Id), serialized, time.Hour)
		cache.client.Del(ctx, utils.CreateRedisKeyForPostPage(result.AuthorId)) // invalidate post page cache
	}

	return result, err
}

func (cache *RedisRepository) EditPost(ctx context.Context, id model.UserId, post model.Post) (model.Post, error) {
	result, err := cache.persistentRepo.EditPost(ctx, id, post)
	if err == nil {
		serialized, _ := json.Marshal(result)
		cache.client.Set(ctx, utils.CreateRedisKeyForPost(result.Id), serialized, time.Hour)
	}

	return result, err
}

func (cache *RedisRepository) GetPostById(ctx context.Context, id model.PostId) (model.Post, error) {
	key := utils.CreateRedisKeyForPost(id)
	result := cache.client.Get(ctx, key)

	switch serialized, err := result.Result(); {
	case err == redis.Nil:
		// continue execution
	case err != nil:
		return model.Post{}, fmt.Errorf("failed to get value from redis due to error %s", err)
	default:
		log.Printf("Successfully obtained post from cache for key %s", key)
		var post model.Post
		err = json.Unmarshal([]byte(serialized), &post)
		return post, err
	}

	post, err := cache.persistentRepo.GetPostById(ctx, id)
	if err == nil {
		serialized, _ := json.Marshal(post)
		cache.client.Set(ctx, utils.CreateRedisKeyForPost(post.Id), serialized, time.Hour)
	}
	return post, err
}

func (cache *RedisRepository) GetPosts(ctx context.Context, id model.UserId, page model.PageToken, size int) ([]model.Post, model.PageToken, error) {
	// cache only first page for each user
	if page != model.EmptyPage {
		return cache.persistentRepo.GetPosts(ctx, id, page, size)
	}

	key := utils.CreateRedisKeyForPostPage(id)
	result := cache.client.Get(ctx, key)

	switch serialized, err := result.Result(); {
	case err == redis.Nil:
		// continue execution
	case err != nil:
		return []model.Post{}, model.EmptyPage, fmt.Errorf("failed to get value from redis due to error %s", err)
	default:
		log.Printf("Successfully obtained first page from cache for key %s", key)
		var record model.PostPageCacheRecord
		_ = json.Unmarshal([]byte(serialized), &record)

		if size == len(record.Posts) {
			return record.Posts, record.Page, nil
		}
		// continue execution
	}

	posts, newPage, err := cache.persistentRepo.GetPosts(ctx, id, page, size)

	if err == nil {
		record := model.PostPageCacheRecord{Posts: posts, Page: newPage}
		serialized, _ := json.Marshal(record)
		cache.client.Set(ctx, key, serialized, time.Hour)
	}

	return posts, newPage, err
}

func (cache *RedisRepository) Clear(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}
