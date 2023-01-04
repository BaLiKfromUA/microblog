package model

type PostPageCacheRecord struct {
	Posts []Post
	Page  PageToken
}

type FeedPageCacheRecord struct {
	FeedMetadata []FeedMetadataDocument
	Page         PageToken
}
