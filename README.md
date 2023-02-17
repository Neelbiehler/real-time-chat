# Real-Time Chat Application in Go

This is a real-time chat application built in Go using WebSockets and Redis. It allows multiple clients to connect and communicate with each other in real-time.

## Features

- Real-time messaging: Clients can send and receive messages in real-time, using WebSockets.
- Redis Pub/Sub: The application uses Redis Pub/Sub to broadcast messages to all connected clients.
- Scalability: The application can be easily scaled by running multiple instances and using a load balancer.