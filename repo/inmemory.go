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

func (storage *InMemoryRepository) CreatePost(_ context.Context, id model.UserId, post model.Post) (model.Post, error) {
	post.Id = utils.CreateRandomPostId()
	post.AuthorId = id
	post.CreatedAt = utils.Now()

	ok := storage.tryToCreatePost(post)

	if !ok {
		return post, model.PostCreationFailed
	} else {
		return post, nil
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
	l.PushBack(post)

	return true
}

func (storage *InMemoryRepository) GetPostById(_ context.Context, id model.PostId) (model.Post, error) {
	storage.mu.Lock()
	defer storage.mu.Unlock()
	post, ok := storage.postById[id]

	if !ok {
		return post, model.PostNotFound
	} else {
		return post, nil
	}
}

func (storage *InMemoryRepository) GetPosts(_ context.Context, id model.UserId, page model.PageToken, size int) ([]model.Post, model.PageToken, error) {
	storage.mu.RLock()
	defer storage.mu.RUnlock()

	var posts = make([]model.Post, 0)
	var pageToken model.PageToken
	var err error

	var feedTail *list.Element

	if page != model.EmptyPage {
		storage.pagesMu.Lock()
		var ok bool
		feedTail, ok = storage.userPages[page]
		storage.pagesMu.Unlock()

		if !ok || feedTail == nil || feedTail.Value.(model.Post).AuthorId != id {
			return posts, model.EmptyPage, model.InvalidPageToken
		}

	} else {
		l, ok := storage.userPosts[id]
		if ok {
			feedTail = l.Back()
		}
	}

	for feedTail != nil && size > 0 {
		posts = append(posts, feedTail.Value.(model.Post))
		feedTail = feedTail.Prev()
		size -= 1
	}

	if feedTail != nil {
		pageToken = model.PageToken(utils.UUID())
		storage.pagesMu.Lock()
		storage.userPages[pageToken] = feedTail
		storage.pagesMu.Unlock()
	} else {
		pageToken = model.EmptyPage
	}

	return posts, pageToken, err
}

func (storage *InMemoryRepository) clear(_ context.Context) {
	storage.mu.Lock()
	storage.pagesMu.Lock()
	defer storage.mu.Unlock()
	defer storage.pagesMu.Unlock()

	maps.Clear(storage.postById)
	maps.Clear(storage.userPages)
	maps.Clear(storage.userPosts)
}
