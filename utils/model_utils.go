package utils

import (
	"encoding/base64"
	"github.com/google/uuid"
	"microblog/model"
	"time"
)

func Now() model.ISOTimestamp {
	t := time.Now().UTC()
	return model.ISOTimestamp(t.Format(time.RFC3339))
}

func UUID() string {
	return uuid.New().String()
}

func CreateRandomPostId() model.PostId {
	return model.PostId(base64.URLEncoding.EncodeToString([]byte(UUID())))
}
