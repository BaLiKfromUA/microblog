package repo

import (
	"container/list"
	"context"
	"golang.org/x/exp/maps"
	"microblog/model"
	"microblog/utils"
	"sync"
)

var _ Repository = (*InMemoryRepository)(nil)

type InMemoryRepository struct {
	mu        *sync.RWMutex
	pagesMu   *sync.Mutex
	postById  map[model.PostId]model.Post
	userPosts map[model.UserId]*list.List
	userPages map[model.PageToken]*list.Element
}

func NewInMemoryRepository() Repository {
	return &InMemoryRepository{
		mu:        &sync.RWMutex{},
		pagesMu:   &sync.Mutex{},
		postById:  make(map[model.PostId]model.Post),
		userPosts: make(map[model.UserId]*list.List),
		userPages: make(map[model.PageToken]*list.Element),
	}
}

func (storage *InMemoryRepository) CreatePost(ctx context.Context, userId model.UserId, post model.Post) (model.Post, error) {
	post.Id = utils.CreateRandomPostId()
	post.AuthorId = userId

	now := utils.Now()
	post.CreatedAt = now
	post.LastModifiedAt = now

	ok := storage.tryToCreatePost(post)

	if !ok {
		return model.Post{}, model.PostCreationFailed
	} else {
		return post, nil
	}
}

func (storage *InMemoryRepository) EditPost(ctx context.Context, id model.UserId, post model.Post) (model.Post, error) {
	result := storage.tryToEditPost(post)

	if result != nil {
		return *result, nil
	} else {
		return model.Post{}, model.PostNotFound
	}
}

func (storage *InMemoryRepository) tryToCreatePost(post model.Post) bool {
	storage.mu.Lock()
	defer storage.mu.Unlock()

	if _, ok := storage.postById[post.Id]; ok {
		return false
	}

	storage.postById[post.Id] = post

	l, ok := storage.userPosts[post.AuthorId]
	if !ok {
		l = list.New()
		storage.userPosts[post.AuthorId] = l
	}
	l.PushBack(post.Id)

	return true
}

func (storage *InMemoryRepository) tryToEditPost(post model.Post) *model.Post {
	storage.mu.Lock()
	defer storage.mu.Unlock()

	if p, ok := storage.postById[post.Id]; ok {
		p.Text = post.Text
		p.LastModifiedAt = utils.Now()
		storage.postById[p.Id] = p
		return &p
	}

	return nil
}

func (storage *InMemoryRepository) GetPostById(ctx context.Context, id model.PostId) (model.Post, error) {
	storage.mu.RLock()
	defer storage.mu.RUnlock()

	post, ok := storage.postById[id]

	if !ok {
		return post, model.PostNotFound
	} else {
		return post, nil
	}
}

func (storage *InMemoryRepository) GetPosts(ctx context.Context, id model.UserId, page model.PageToken, size int) ([]model.Post, model.PageToken, error) {
	storage.mu.RLock()
	defer storage.mu.RUnlock()

	var posts = make([]model.Post, 0)
	var pageToken model.PageToken
	var err error

	var feedTail *list.Element

	if page != "none" {
		storage.pagesMu.Lock()
		var ok bool
		feedTail, ok = storage.userPages[page]
		storage.pagesMu.Unlock()
		if !ok {
			return posts, "none", model.InvalidPageToken
		}

		if feedTail == nil || storage.postById[feedTail.Value.(model.PostId)].AuthorId != id {
			return posts, "none", model.InvalidPageToken
		}
	} else {
		l, ok := storage.userPosts[id]
		if ok {
			feedTail = l.Back()
		}
	}

	for feedTail != nil && size > 0 {
		posts = append(posts, storage.postById[feedTail.Value.(model.PostId)])
		feedTail = feedTail.Prev()
		size -= 1
	}

	if feedTail != nil {
		pageToken = model.PageToken(utils.UUID())
		storage.pagesMu.Lock()
		storage.userPages[pageToken] = feedTail
		storage.pagesMu.Unlock()
	} else {
		pageToken = "none"
	}

	return posts, pageToken, err
}

func (storage *InMemoryRepository) Clear(ctx context.Context) error {
	storage.mu.Lock()
	storage.pagesMu.Lock()
	defer storage.mu.Unlock()
	defer storage.pagesMu.Unlock()

	maps.Clear(storage.postById)
	maps.Clear(storage.userPages)
	maps.Clear(storage.userPosts)
	return nil
}
