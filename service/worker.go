package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/RichardKnop/machinery/v1"
	"github.com/RichardKnop/machinery/v1/config"
	"github.com/RichardKnop/machinery/v1/log"
	"github.com/RichardKnop/machinery/v1/tasks"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"microblog/model"
	"microblog/repo"
	"os"
)

type Consumer struct {
	repo repo.Repository
}

type Producer struct {
	repo   repo.Repository
	server *machinery.Server
}

func StartProducer(r repo.Repository) (Producer, error) {
	producer := Producer{repo: r}
	server, err := startServer(r)
	producer.server = server
	return producer, err
}

func StartConsumer(r repo.Repository) error {
	log.INFO.Printf("Starting worker...")

	consumerTag := "machinery_worker"

	server, err := startServer(r)
	if err != nil {
		return err
	}

	worker := server.NewWorker(consumerTag, 0)

	errorhandler := func(err error) {
		log.ERROR.Println("Something went wrong:", err)
	}

	worker.SetErrorHandler(errorhandler)

	return worker.Launch()
}

func startServer(r repo.Repository) (*machinery.Server, error) {
	url, ok := os.LookupEnv("REDIS_URL")

	if !ok {
		url = "redis://localhost:6379"
	} else {
		url = "redis://" + url
	}

	cnf := &config.Config{
		DefaultQueue:    "machinery_tasks",
		ResultsExpireIn: 3600,
		Broker:          url,
		ResultBackend:   url,
		Redis: &config.RedisConfig{
			MaxIdle:                3,
			IdleTimeout:            240,
			ReadTimeout:            15,
			WriteTimeout:           15,
			ConnectTimeout:         15,
			NormalTasksPollPeriod:  1000,
			DelayedTasksPollPeriod: 500,
		},
	}

	server, err := machinery.NewServer(cnf)
	if err != nil {
		return nil, err
	}

	consumer := Consumer{
		repo: r,
	}

	// Register tasks
	t := map[string]interface{}{
		"streamNewPost": consumer.StreamNewPost,
		"rebuildFeed":   consumer.RebuildFeed,
	}

	return server, server.RegisterTasks(t)
}

func (p *Producer) SendPostTask(ctx context.Context, post model.Post) error {
	serialized, _ := json.Marshal(post)

	task := tasks.Signature{
		Name: "streamNewPost",
		Args: []tasks.Arg{
			{
				Name:  "serialized",
				Type:  "string",
				Value: string(serialized),
			},
		},
	}

	_, err := p.server.SendTaskWithContext(ctx, &task)
	if err != nil {
		return fmt.Errorf("could not send task: %s", err.Error())
	}

	return nil
}

func (p *Producer) SendFeedTask(ctx context.Context, from, to model.UserId) error {
	task := tasks.Signature{
		Name: "rebuildFeed",
		Args: []tasks.Arg{
			{
				Name:  "feedOwner",
				Type:  "string",
				Value: string(from),
			},
			{
				Name:  "newSource",
				Type:  "string",
				Value: string(to),
			},
		},
	}

	_, err := p.server.SendTaskWithContext(ctx, &task)
	if err != nil {
		return fmt.Errorf("could not send task: %s", err.Error())
	}

	return nil
}

func (c *Consumer) StreamNewPost(serialized string) (string, error) {
	var post model.Post
	_ = json.Unmarshal([]byte(serialized), &post)

	// because json ignores token field
	post.Token, _ = primitive.ObjectIDFromHex(string(post.Id))

	followers, err := c.repo.GetSubscribers(context.Background(), post.AuthorId)
	if err != nil {
		log.ERROR.Println(err.Error())
		return "get followers", err
	}

	for _, follower := range followers {
		metadata := model.FeedMetadataDocument{UserId: follower, PostId: post.Id, Token: post.Token}
		err = c.repo.AddPostToFeed(context.Background(), metadata)
		if err != nil {
			log.ERROR.Println(err.Error())
			return "update feed", err
		}
	}

	return "done", nil
}

func (c *Consumer) RebuildFeed(feedOwner, newSource string) (string, error) {
	log.INFO.Printf("user %s subscribed for user %s. Rebuilding feed....", feedOwner, newSource)

	posts, err := drainFullPostPage(c.repo, model.UserId(newSource))

	if err != nil {
		log.ERROR.Println(err.Error())
		return "posts", err
	}

	for _, post := range posts {
		metadata := model.FeedMetadataDocument{UserId: model.UserId(feedOwner), PostId: post.Id, Token: post.Token}
		err = c.repo.AddPostToFeed(context.Background(), metadata)
		if err != nil {
			log.ERROR.Println(err.Error())
			return "update feed", err
		}
	}

	return "done", nil
}

func drainFullPostPage(r repo.Repository, userId model.UserId) ([]model.Post, error) {
	page := model.EmptyPage
	size := 100

	var result []model.Post
	firstTry := true

	for page != model.EmptyPage || firstTry {
		firstTry = false
		arr, newPage, err := r.GetPosts(context.Background(), userId, page, size)
		if err != nil {
			return result, err
		}
		page = newPage
		result = append(result, arr...)
	}

	return result, nil
}
