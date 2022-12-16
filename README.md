# microblog

Twitter-like web service. The final project of the "System Design" course.

## Task

Develop a web-service that implements HTTP API with the functionality of a minimalistic microblog like Twitter. The
service must provide the following functionality:

- creating a new post
- editing existing posts
- getting the post by a unique identifier
- getting all posts of requested user in reverse chronological order with pagination
- getting the posts feed for an authorized user 
> A post's feed is a set of user's posts to which the current user is subscribed, ordered by time.
- user subscription logic

Formal description of API can be found in [api.yaml](./service/api.yaml).

As storage, I use **MongoDB** with caching based on **Redis**.

For background tasks handling, such as updating users feeds,
I run my app in `WORKER` mode + use **Redis** as the message queue.

**Environment variables:**

- `SERVER_PORT` --- port number on which the API should be available. Default value: `8080`.
- `APP_MODE` --- service startup mode. Possible values:
    - `SERVER` --- the service starts the http server.
    - `WORKER` ---  the service starts the worker (message consumer).
- `MONGO_URL` --- MongoDB connection address. Default value: `mongodb://localhost:27017`.
- `MONGO_DBNAME` --- the name of the database that can be used for storage. Default value: `system_design`.
- `REDIS_URL` --- address for connecting to Redis. Default value: `127.0.0.1:6379`.
