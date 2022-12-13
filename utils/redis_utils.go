package utils

import "microblog/model"

func CreateRedisKeyForPost(postId model.PostId) string {
	return "post:" + string(postId)
}

func CreateRedisKeyForPostPage(userId model.UserId) string {
	return "posts:" + string(userId)
}

func CreateRedisKeyForSubscribers(userId model.UserId) string {
	return "subscribers:" + string(userId)
}

func CreateRedisKeyForSubscriptions(userId model.UserId) string {
	return "subscriptions:" + string(userId)
}

func CreateRedisKeyForFeedPage(userId model.UserId) string {
	return "feeds:" + string(userId)
}
