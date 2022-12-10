package utils

import (
	"encoding/json"
	"fmt"
	"microblog/model"
	"net/http"
	"regexp"
	"strconv"
)

func GetAuthorizedUserId(r *http.Request) (model.UserId, error) {
	userId := model.UserId(r.Header.Get("System-Design-User-Id"))
	matched, err := regexp.Match(`[0-9a-f]+`, []byte(userId))

	if !matched || err != nil {
		return userId, fmt.Errorf("empty or invalid user id")
	}

	return userId, nil
}

func GetPageToken(r *http.Request) (model.PageToken, error) {
	pageToken := model.PageToken(r.URL.Query().Get("page"))

	if pageToken == "" {
		pageToken = model.EmptyPage
	} else {
		matched, err := regexp.Match(`[A-Za-z0-9_\-]+`, []byte(pageToken))

		if !matched || err != nil {
			return pageToken, fmt.Errorf("invalid page token")
		}
	}

	return pageToken, nil
}

func GetSize(r *http.Request) (int, error) {
	rSize := r.URL.Query().Get("size")
	var size int

	if rSize == "" {
		size = 10
	} else {
		var err error
		size, err = strconv.Atoi(rSize)

		if err != nil || size < 1 || size > 100 {
			return size, fmt.Errorf("invalid size param")
		}
	}

	return size, nil
}

func WriteResponseBody(rw http.ResponseWriter, body any) {
	rawResponse, _ := json.Marshal(body)

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	_, _ = rw.Write(rawResponse)
}
