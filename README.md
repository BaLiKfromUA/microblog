# microblog
Twitter-like web service. The final project of the "System Design" course.

## Task

Develop a web-service that implements HTTP API with the functionality of a minimalistic microblog like Twitter. The service must provide the following functionality:

- creating a new post
- getting the post by a unique identifier
- getting all posts of requested user in reverse chronological order with pagination

Formal description of API can be found in [api.yaml](./service/api.yaml).

Environment variables:

- `SERVER_PORT`, if empty then `8080`
