package repo

import (
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx"
	"log"
	"microblog/model"
	"microblog/utils"
	"os"
	"time"
)

import (
	"go.mongodb.org/mongo-driver/mongo"
)

var _ Repository = (*MongoDatabaseRepository)(nil)

type MongoDatabaseRepository struct {
	posts     *mongo.Collection
	feeds     *mongo.Collection
	following *mongo.Collection
	followed  *mongo.Collection
}

func NewMongoDatabaseRepository() Repository {
	dbUrl, ok := os.LookupEnv("MONGO_URL")
	if !ok {
		dbUrl = "mongodb://localhost:27017"
	}
	dbName, ok := os.LookupEnv("MONGO_DBNAME")
	if !ok {
		dbName = "system_design"
	}
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(dbUrl))
	if err != nil {
		panic(err)
	}

	posts := client.Database(dbName).Collection("posts")
	ensureIndexesForPosts(ctx, posts)
	feeds := client.Database(dbName).Collection("feeds")
	ensureIndexesForFeed(ctx, feeds)

	following := client.Database(dbName).Collection("following")
	ensureIndexesById(ctx, following)
	followed := client.Database(dbName).Collection("followed")
	ensureIndexesById(ctx, followed)

	return &MongoDatabaseRepository{posts: posts, feeds: feeds, followed: followed, following: following}
}

func ensureIndexesForPosts(ctx context.Context, collection *mongo.Collection) {
	indexModels := []mongo.IndexModel{
		{
			Keys: bsonx.Doc{
				{Key: "authorId", Value: bsonx.Int32(1)},
				{Key: "_id", Value: bsonx.Int32(-1)},
			},
		},
	}
	opts := options.CreateIndexes().SetMaxTime(10 * time.Second)

	_, err := collection.Indexes().CreateMany(ctx, indexModels, opts)
	if err != nil {
		panic(fmt.Errorf("failed to ensure indexes %w", err))
	}
}

func ensureIndexesForFeed(ctx context.Context, collection *mongo.Collection) {
	indexModels := []mongo.IndexModel{
		{
			Keys: bsonx.Doc{
				{Key: "userId", Value: bsonx.Int32(1)},
				{Key: "token", Value: bsonx.Int32(-1)},
			},
		},
	}
	opts := options.CreateIndexes().SetMaxTime(10 * time.Second)

	_, err := collection.Indexes().CreateMany(ctx, indexModels, opts)
	if err != nil {
		panic(fmt.Errorf("failed to ensure indexes %w", err))
	}
}

func ensureIndexesById(ctx context.Context, collection *mongo.Collection) {
	indexModels := []mongo.IndexModel{
		{
			Keys: bsonx.Doc{
				{Key: "_id", Value: bsonx.Int32(1)},
			},
		},
	}
	opts := options.CreateIndexes().SetMaxTime(10 * time.Second)

	_, err := collection.Indexes().CreateMany(ctx, indexModels, opts)
	if err != nil {
		panic(fmt.Errorf("failed to ensure indexes by id %w", err))
	}
}

func (storage *MongoDatabaseRepository) CreatePost(ctx context.Context, id model.UserId, post model.Post) (model.Post, error) {
	post.Token = primitive.NewObjectID()
	post.Id = model.PostId(post.Token.Hex())
	post.AuthorId = id

	now := utils.Now()
	post.CreatedAt = now
	post.LastModifiedAt = now

	_, err := storage.posts.InsertOne(ctx, post)

	if err != nil {
		log.Printf(err.Error())
		err = model.PostCreationFailed
	}

	return post, err
}

func (storage *MongoDatabaseRepository) EditPost(ctx context.Context, id model.UserId, post model.Post) (model.Post, error) {
	var result model.Post

	err := storage.posts.FindOneAndUpdate(ctx,
		bson.M{"id": post.Id},
		bson.D{
			{"$set",
				bson.D{
					{"text", post.Text},
					{"lastModifiedAt", utils.Now()},
				}},
		},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&result)

	if err != nil && errors.Is(err, mongo.ErrNoDocuments) {
		log.Printf(err.Error())
		err = model.PostNotFound
	}

	return result, err
}

func (storage *MongoDatabaseRepository) GetPostById(ctx context.Context, id model.PostId) (model.Post, error) {
	var result model.Post
	err := storage.posts.FindOne(ctx, bson.M{"id": id}).Decode(&result)
	if err != nil && errors.Is(err, mongo.ErrNoDocuments) {
		log.Printf(err.Error())
		err = model.PostNotFound
	}
	return result, err
}

func (storage *MongoDatabaseRepository) GetPosts(ctx context.Context, id model.UserId, page model.PageToken, size int) ([]model.Post, model.PageToken, error) {
	var result []model.Post
	newToken := model.PageToken("none")

	opts := options.Find().
		SetSort(bson.D{{"authorId", 1}, {"_id", -1}}).
		SetLimit(int64(size + 1))

	if page == model.EmptyPage {
		cursor, err := storage.posts.Find(ctx, bson.D{{"authorId", id}}, opts)

		if err != nil {
			return result, newToken, err
		}

		if err = cursor.All(ctx, &result); err != nil {
			return result, newToken, err
		}
	} else {
		token, err := primitive.ObjectIDFromHex(string(page))
		if err != nil {
			return result, model.EmptyPage, model.InvalidPageToken
		}

		// naive way to validate page token
		var r model.Post
		err = storage.posts.FindOne(ctx, bson.M{"_id": token}).Decode(&r)
		if err != nil || r.AuthorId != id {
			return result, model.EmptyPage, model.InvalidPageToken
		}

		cursor, err := storage.posts.Find(ctx,
			bson.D{{"authorId", id}, {"_id", bson.M{"$lt": token}}}, opts)

		if err != nil {
			return result, newToken, err
		}

		if err = cursor.All(ctx, &result); err != nil {
			return result, newToken, err
		}
	}

	if len(result) > 1 && len(result) == size+1 {
		newToken = model.PageToken(result[len(result)-2].Token.Hex())
		result = result[0 : len(result)-1]
	} else {
		newToken = model.EmptyPage
	}

	return result, newToken, nil
}

func (storage *MongoDatabaseRepository) Subscribe(ctx context.Context, subscriberId model.UserId, targetId model.UserId) error {
	if subscriberId == targetId {
		return fmt.Errorf("fromId == toId --> %s", subscriberId)
	}

	opts := options.Update().SetUpsert(true)

	update := bson.M{
		"$addToSet": bson.M{
			"ids": bson.M{"$each": []model.UserId{subscriberId}},
		},
	}

	result, err := storage.followed.UpdateOne(ctx, bson.M{"_id": targetId}, update, opts)

	if err != nil {
		return err
	}

	if result.ModifiedCount+result.UpsertedCount == 0 {
		return model.AlreadySubscribed
	}

	update = bson.M{
		"$addToSet": bson.M{
			"ids": bson.M{"$each": []model.UserId{targetId}},
		},
	}

	_, err = storage.following.UpdateOne(ctx, bson.M{"_id": subscriberId}, update, opts)

	if err != nil {
		return err
	}

	return nil
}

func (storage *MongoDatabaseRepository) GetSubscriptions(ctx context.Context, id model.UserId) ([]model.UserId, error) {
	var result []model.UserId
	var doc model.SubscriptionsDocument
	err := storage.following.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)

	if err == nil {
		result = doc.Ids
	}

	if err != nil && errors.Is(err, mongo.ErrNoDocuments) {
		log.Printf(err.Error())
		err = nil
		result = []model.UserId{}
	}

	return result, err
}

func (storage *MongoDatabaseRepository) GetSubscribers(ctx context.Context, id model.UserId) ([]model.UserId, error) {
	var result []model.UserId
	var doc model.SubscriptionsDocument
	err := storage.followed.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)

	if err == nil {
		result = doc.Ids
	}

	if err != nil && errors.Is(err, mongo.ErrNoDocuments) {
		log.Printf(err.Error())
		err = nil
		result = []model.UserId{}
	}

	return result, err
}

func (storage *MongoDatabaseRepository) GetFeed(ctx context.Context, id model.UserId, page model.PageToken, size int) ([]model.FeedMetadataDocument, model.PageToken, error) {
	var result []model.FeedMetadataDocument
	newToken := model.EmptyPage

	opts := options.Find().
		SetSort(bson.D{{"userId", 1}, {"token", -1}}).
		SetLimit(int64(size + 1))

	if page == model.EmptyPage {
		cursor, err := storage.feeds.Find(ctx, bson.D{{"userId", id}}, opts)
		if err != nil {
			return result, newToken, err
		}

		if err = cursor.All(ctx, &result); err != nil {
			return result, newToken, err
		}
	} else {
		token, err := primitive.ObjectIDFromHex(string(page))
		if err != nil {
			return result, "none", model.InvalidPageToken
		}

		var r model.FeedMetadataDocument
		err = storage.feeds.FindOne(ctx, bson.M{"token": token}).Decode(&r)
		if err != nil || r.UserId != id {
			return result, "none", model.InvalidPageToken
		}

		cursor, err := storage.feeds.Find(ctx,
			bson.D{{"userId", id}, {"token", bson.M{"$lt": token}}}, opts)

		if err != nil {
			return result, newToken, err
		}

		if err = cursor.All(ctx, &result); err != nil {
			return result, newToken, err
		}
	}

	if len(result) > 1 && len(result) == size+1 {
		newToken = model.PageToken(result[len(result)-2].Token.Hex())
		result = result[0 : len(result)-1]
	} else {
		newToken = model.EmptyPage
	}

	return result, newToken, nil
}

func (storage *MongoDatabaseRepository) AddPostToFeed(ctx context.Context, post model.FeedMetadataDocument) error {
	_, err := storage.feeds.InsertOne(ctx, post)

	return err
}
