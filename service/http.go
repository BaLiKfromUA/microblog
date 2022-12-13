package service

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"microblog/model"
	"microblog/repo"
	"microblog/utils"
	"net/http"
	"os"
	"time"
)

type HTTPHandler struct {
	repo repo.Repository
}

type GetFeedResponse struct {
	Posts    []model.Post     `json:"posts"`
	NextPage *model.PageToken `json:"nextPage,omitempty"`
}

func NewHTTPHandler(repo repo.Repository) *HTTPHandler {
	return &HTTPHandler{
		repo: repo,
	}
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

	var respBody GetFeedResponse
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
	r.HandleFunc("/maintenance/ping", handler.Ping).Methods(http.MethodGet)

	return r
}

func NewServer(repo repo.Repository) *http.Server {
	handler := NewHTTPHandler(repo)

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

	return srv
}

func (h *HTTPHandler) Ping(rw http.ResponseWriter, _ *http.Request) {
	rw.WriteHeader(http.StatusOK)
}
