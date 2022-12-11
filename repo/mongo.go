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
	posts *mongo.Collection
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

	return &MongoDatabaseRepository{posts: posts}
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

func (storage *MongoDatabaseRepository) CreatePost(ctx context.Context, id model.UserId, post model.Post) (model.Post, error) {
	post.Token = primitive.NewObjectID()
	post.Id = model.PostId(post.Token.Hex())
	post.AuthorId = id

	now := utils.Now()
	post.CreatedAt = now

	_, err := storage.posts.InsertOne(ctx, post)

	if err != nil {
		log.Printf(err.Error())
		err = model.PostCreationFailed
	}

	return post, err
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

func (storage *MongoDatabaseRepository) clear(ctx context.Context) {
	//TODO implement me
	panic("implement me")
}
