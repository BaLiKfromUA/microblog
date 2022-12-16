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

func (cache *RedisRepository) Subscribe(ctx context.Context, from model.UserId, to model.UserId) error {
	err := cache.persistentRepo.Subscribe(ctx, from, to)

	if err == nil {
		keyFrom := utils.CreateRedisKeyForSubscriptions(from)
		keyTo := utils.CreateRedisKeyForSubscribers(to)
		cache.client.Del(ctx, keyFrom)
		cache.client.Del(ctx, keyTo)
	}

	return err
}

func (cache *RedisRepository) GetSubscriptions(ctx context.Context, id model.UserId) ([]model.UserId, error) {
	key := utils.CreateRedisKeyForSubscriptions(id)
	result := cache.client.Get(ctx, key)

	switch serialized, err := result.Result(); {
	case err == redis.Nil:
		// continue execution
	case err != nil:
		return []model.UserId{}, fmt.Errorf("failed to get value from redis due to error %s", err)
	default:
		log.Printf("Successfully obtained ids from cache for key %s", key)
		var ids []model.UserId
		err = json.Unmarshal([]byte(serialized), &ids)
		return ids, err
	}

	ids, err := cache.persistentRepo.GetSubscriptions(ctx, id)

	if err == nil {
		serialized, _ := json.Marshal(ids)
		cache.client.Set(ctx, key, serialized, time.Hour)
	}

	return ids, err
}

func (cache *RedisRepository) GetSubscribers(ctx context.Context, id model.UserId) ([]model.UserId, error) {
	key := utils.CreateRedisKeyForSubscribers(id)
	result := cache.client.Get(ctx, key)

	switch serialized, err := result.Result(); {
	case err == redis.Nil:
		// continue execution
	case err != nil:
		return []model.UserId{}, fmt.Errorf("failed to get value from redis due to error %s", err)
	default:
		log.Printf("Successfully obtained ids from cache for key %s", key)
		var ids []model.UserId
		err = json.Unmarshal([]byte(serialized), &ids)
		return ids, err
	}

	ids, err := cache.persistentRepo.GetSubscribers(ctx, id)

	if err == nil {
		serialized, _ := json.Marshal(ids)
		cache.client.Set(ctx, key, serialized, time.Hour)
	}

	return ids, err
}

func (cache *RedisRepository) GetFeed(ctx context.Context, id model.UserId, page model.PageToken, size int) ([]model.FeedMetadataDocument, model.PageToken, error) {
	// we cache only first page for each user
	if page != model.EmptyPage {
		return cache.persistentRepo.GetFeed(ctx, id, page, size)
	}

	key := utils.CreateRedisKeyForFeedPage(id)
	result := cache.client.Get(ctx, key)

	switch serialized, err := result.Result(); {
	case err == redis.Nil:
		// continue execution
	case err != nil:
		return []model.FeedMetadataDocument{}, model.EmptyPage, fmt.Errorf("failed to get value from redis due to error %s", err)
	default:
		log.Printf("Successfully obtained first feed page from cache for key %s", key)
		var record model.FeedPageCacheRecord
		_ = json.Unmarshal([]byte(serialized), &record)

		if size == len(record.FeedMetadata) {
			return record.FeedMetadata, record.Page, nil
		}
		// continue execution
	}

	feed, newPage, err := cache.persistentRepo.GetFeed(ctx, id, page, size)

	if err == nil {
		record := model.FeedPageCacheRecord{FeedMetadata: feed, Page: newPage}
		serialized, _ := json.Marshal(record)
		cache.client.Set(ctx, key, serialized, time.Hour)
	}

	return feed, newPage, err
}

func (cache *RedisRepository) AddPostToFeed(ctx context.Context, post model.FeedMetadataDocument) error {
	err := cache.persistentRepo.AddPostToFeed(ctx, post)

	if err == nil {
		key := utils.CreateRedisKeyForFeedPage(post.UserId)
		cache.client.Del(ctx, key)
	}

	return err
}
