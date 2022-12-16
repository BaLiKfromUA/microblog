package service

import (
	"encoding/json"
	"errors"
	"github.com/gorilla/mux"
	"microblog/model"
	"microblog/repo"
	"microblog/utils"
	"net/http"
	"os"
	"time"
)

type HTTPHandler struct {
	repo     repo.Repository
	producer Producer
}

type GetPostPageResponse struct {
	Posts    []model.Post     `json:"posts"`
	NextPage *model.PageToken `json:"nextPage,omitempty"`
}

type GetUsersResponse struct {
	Users []model.UserId `json:"users"`
}

func NewHTTPHandler(repo repo.Repository) (*HTTPHandler, error) {
	p, err := StartProducer(repo)

	if err != nil {
		return nil, err
	}

	return &HTTPHandler{
		repo:     repo,
		producer: p,
	}, nil
}

func (h *HTTPHandler) CreatePost(rw http.ResponseWriter, r *http.Request) {
	userId, err := utils.GetAuthorizedUserId(r)

	if err != nil {
		http.Error(rw, "Empty or Invalid User Id!", http.StatusUnauthorized)
		return
	}

	var post model.Post
	err = json.NewDecoder(r.Body).Decode(&post)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	post, err = h.repo.CreatePost(r.Context(), userId, post)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	err = h.producer.SendPostTask(r.Context(), post)

	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	utils.WriteResponseBody(rw, post)
}

func (h *HTTPHandler) EditPost(rw http.ResponseWriter, r *http.Request) {
	userId, err := utils.GetAuthorizedUserId(r)

	if err != nil {
		http.Error(rw, "Empty or Invalid User Id!", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	postId, ok := vars["postId"]

	if !ok {
		http.Error(rw, "Invalid post id in path", http.StatusNotFound)
		return
	}

	oldPost, err := h.repo.GetPostById(r.Context(), model.PostId(postId))

	if err != nil {
		http.Error(rw, "Invalid post id in path", http.StatusNotFound)
		return
	}

	if oldPost.AuthorId != userId {
		http.Error(rw, "Given user id is not a creator of requested post", http.StatusForbidden)
		return
	}

	var postToEdit model.Post
	err = json.NewDecoder(r.Body).Decode(&postToEdit)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	postToEdit.Id = model.PostId(postId)

	resultedPost, err := h.repo.EditPost(r.Context(), userId, postToEdit)

	if err != nil {
		http.Error(rw, "Invalid post id in path", http.StatusNotFound)
		return
	}

	utils.WriteResponseBody(rw, resultedPost)
}

func (h *HTTPHandler) GetPostById(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	postId, ok := vars["postId"]

	if !ok {
		http.Error(rw, "Invalid post id in path", http.StatusNotFound)
		return
	}

	post, err := h.repo.GetPostById(r.Context(), model.PostId(postId))

	if err != nil {
		http.Error(rw, err.Error(), http.StatusNotFound)
		return
	}

	utils.WriteResponseBody(rw, post)
}

func (h *HTTPHandler) GetPosts(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userId, ok := vars["userId"]

	if !ok {
		http.Error(rw, "Invalid user id in path", http.StatusBadRequest)
		return
	}

	pageToken, err := utils.GetPageToken(r)
	if err != nil {
		http.Error(rw, "Invalid Page Token", http.StatusUnauthorized)
		return
	}

	size, err := utils.GetSize(r)

	if err != nil {
		http.Error(rw, "Invalid size param", http.StatusBadRequest)
		return
	}

	posts, nextPageToken, err := h.repo.GetPosts(r.Context(), model.UserId(userId), pageToken, size)

	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	if posts == nil {
		posts = []model.Post{}
	}

	var respBody GetPostPageResponse
	respBody.Posts = posts

	if nextPageToken != model.EmptyPage {
		respBody.NextPage = &nextPageToken
	}

	utils.WriteResponseBody(rw, respBody)
}

func (h *HTTPHandler) Subscribe(rw http.ResponseWriter, r *http.Request) {
	fromUserId, err := utils.GetAuthorizedUserId(r)

	if err != nil {
		http.Error(rw, "Empty or Invalid User Id!", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	toUserId, ok := vars["userId"]

	if !ok {
		http.Error(rw, "Invalid user id in path", http.StatusBadRequest)
		return
	}

	err = h.repo.Subscribe(r.Context(), fromUserId, model.UserId(toUserId))

	if err != nil {
		if errors.Is(err, model.AlreadySubscribed) {
			rw.WriteHeader(http.StatusOK)
		} else {
			http.Error(rw, err.Error(), http.StatusBadRequest)
		}
		return
	}

	err = h.producer.SendFeedTask(r.Context(), fromUserId, model.UserId(toUserId))

	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	rw.WriteHeader(http.StatusOK)
}

func (h *HTTPHandler) GetSubscriptions(rw http.ResponseWriter, r *http.Request) {
	userId, err := utils.GetAuthorizedUserId(r)
	if err != nil {
		http.Error(rw, "Empty or Invalid User Id!", http.StatusUnauthorized)
		return
	}

	result, err := h.repo.GetSubscriptions(r.Context(), userId)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	if result == nil {
		result = []model.UserId{}
	}

	var respBody GetUsersResponse
	respBody.Users = result

	utils.WriteResponseBody(rw, respBody)
}

func (h *HTTPHandler) GetSubscribers(rw http.ResponseWriter, r *http.Request) {
	userId, err := utils.GetAuthorizedUserId(r)
	if err != nil {
		http.Error(rw, "Empty or Invalid User Id!", http.StatusUnauthorized)
		return
	}

	result, err := h.repo.GetSubscribers(r.Context(), userId)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	if result == nil {
		result = []model.UserId{}
	}

	var respBody GetUsersResponse
	respBody.Users = result

	utils.WriteResponseBody(rw, respBody)
}

func (h *HTTPHandler) GetFeed(rw http.ResponseWriter, r *http.Request) {
	userId, err := utils.GetAuthorizedUserId(r)
	if err != nil {
		http.Error(rw, "Empty or Invalid User Id!", http.StatusUnauthorized)
		return
	}

	pageToken, err := utils.GetPageToken(r)
	if err != nil {
		http.Error(rw, "Invalid Page Token", http.StatusUnauthorized)
		return
	}

	size, err := utils.GetSize(r)

	if err != nil {
		http.Error(rw, "Invalid size param", http.StatusBadRequest)
		return
	}

	feedMetadata, nextPageToken, err := h.repo.GetFeed(r.Context(), userId, pageToken, size)

	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	if feedMetadata == nil {
		feedMetadata = []model.FeedMetadataDocument{}
	}

	var posts []model.Post
	posts = []model.Post{}

	for _, metadata := range feedMetadata {
		post, err := h.repo.GetPostById(r.Context(), metadata.PostId)

		if err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}

		posts = append(posts, post)
	}

	var respBody GetPostPageResponse
	respBody.Posts = posts

	if nextPageToken != model.EmptyPage {
		respBody.NextPage = &nextPageToken
	}

	utils.WriteResponseBody(rw, respBody)
}

func createRouter(handler *HTTPHandler) *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc("/api/v1/posts", handler.CreatePost).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/posts/{postId:[A-Za-z0-9_\\-]+}", handler.EditPost).Methods(http.MethodPatch)
	r.HandleFunc("/api/v1/posts/{postId:[A-Za-z0-9_\\-]+}", handler.GetPostById).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/users/{userId:[0-9a-f]+}/posts", handler.GetPosts).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/users/{userId:[0-9a-f]+}/subscribe", handler.Subscribe).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/subscriptions", handler.GetSubscriptions).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/subscribers", handler.GetSubscribers).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/feed", handler.GetFeed).Methods(http.MethodGet)
	r.HandleFunc("/maintenance/ping", handler.Ping).Methods(http.MethodGet)

	return r
}

func NewServer(repo repo.Repository) (*http.Server, error) {
	handler, err := NewHTTPHandler(repo)

	if err != nil {
		return nil, err
	}

	port, ok := os.LookupEnv("SERVER_PORT")
	if !ok {
		port = "8080"
	}

	srv := &http.Server{
		Handler:      createRouter(handler),
		Addr:         "0.0.0.0:" + port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	return srv, nil
}

func (h *HTTPHandler) Ping(rw http.ResponseWriter, _ *http.Request) {
	rw.WriteHeader(http.StatusOK)
}
