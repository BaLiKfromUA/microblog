# microblog
Twitter-like web service. The final project of the "System Design" course.

## Task

Develop a web-service that implements HTTP API with the functionality of a minimalistic microblog like Twitter. The service must provide the following functionality:

- creating a new post
- editing existing posts
- getting the post by a unique identifier
- getting all posts of requested user in reverse chronological order with pagination

Formal description of API can be found in [api.yaml](./service/api.yaml).


As storage, I use **MongoDB** with caching based on **Redis**.

**Environment variables:**

- `SERVER_PORT` --- port number on which the API should be available. If empty then `8080`.
- `STORAGE_MODE` --- storage mode of the posts, one of two values is possible:
    - `inmemory` --- storage in memory (in this mode the service should behave the same as in the first task)
    - `mongo` --- storage in MongoDB. The address and name of the database are passed
      via separate environment variables.
    - `cached` --- storage in MongoDB with caching to Redis. MongoDB and Redis addresses are passed in a separate environment variables.
- `MONGO_URL` --- MongoDB connection address
- `MONGO_DBNAME` --- the name of the database that can be used for storage
