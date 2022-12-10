package model

import "errors"

var PostCreationFailed = errors.New("post_generation_failed")
var PostNotFound = errors.New("post_not_found")
var InvalidPageToken = errors.New("invalid_page_token")
