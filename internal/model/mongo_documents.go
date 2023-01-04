package model

import "go.mongodb.org/mongo-driver/bson/primitive"

type SubscriptionsDocument struct {
	TargetId UserId   `bson:"_id"`
	Ids      []UserId `bson:"ids"`
}

type FeedMetadataDocument struct {
	UserId UserId             `bson:"userId"`
	Token  primitive.ObjectID `bson:"token"`
	PostId PostId             `bson:"postId"`
}
