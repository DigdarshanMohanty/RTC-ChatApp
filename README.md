# Chat Backend

A real-time chat application backend built with Go, featuring WebSocket support, JWT authentication, and in-memory storage.

## Features

- **Real-time messaging** via WebSocket connections
- **JWT-based authentication** with configurable expiry
- **Room-based chat system** with support for multiple chat rooms
- **In-memory storage** (easily replaceable with database)
- **RESTful API** for user management and room operations
- **Graceful shutdown** handling
- **Comprehensive logging** and error handling
- **CORS support** for frontend integration

## API Endpoints

### Authentication
- `POST /api/register` - User registration
- `POST /api/login` - User login

### Chat Rooms
- `GET /api/rooms` - List all chat rooms
- `POST /api/rooms/create` - Create a new chat room

### Messages
- `GET /api/messages?roomId=<id>&limit=<count>` - Get messages from a room
- `GET /ws?roomId=<id>` - WebSocket connection for real-time chat

### Health Check
- `GET /health` - Server health status

## Environment Variables

- `PORT` - Server port (default: 8081)
- `JWT_SECRET` - Secret key for JWT signing (default: dev-super-secret-change-me)
- `JWT_EXPIRY` - JWT token expiry in hours (default: 24)
- `LOG_LEVEL` - Logging level (default: info)
- `MAX_MESSAGE_LENGTH` - Maximum message length (default: 1000)

## Getting Started

### Prerequisites
- Go 1.24.4 or higher

### Installation
1. Clone the repository
2. Navigate to the backend directory: `cd chat-backend`
3. Install dependencies: `go mod tidy`
4. Run the server: `go run cmd/server/main.go`

### Default Configuration
- Server runs on port 8081
- A default "General" chat room is created automatically
- JWT tokens expire after 24 hours
- Maximum message length is 1000 characters

## WebSocket Protocol

### Connection
Connect to `/ws?roomId=<room_id>` with the Authorization header containing the JWT token.

### Message Format
```json
{
  "type": "message",
  "room_id": 1,
  "sender_id": 123,
  "username": "john_doe",
  "content": "Hello, world!",
  "ts": 1640995200000
}
```

### Sending Messages
Send messages in this format:
```json
{
  "content": "Your message here"
}
```

## Project Structure

```
chat-backend/
├── cmd/
│   └── server/
│       └── main.go          # Application entry point
├── config/
│   └── config.go            # Configuration management
├── handlers/
│   ├── auth_handler.go      # Authentication endpoints
│   ├── chat_handler.go      # Chat room endpoints
│   └── message_handler.go   # Message endpoints
├── models/
│   ├── user.go              # User data model
│   ├── message.go           # Message data model
│   └── chatroom.go          # Chat room data model
├── repository/
│   ├── user_repo.go         # User data access
│   ├── message_repo.go      # Message data access
│   └── chat_repo.go         # Chat room data access
├── services/
│   ├── auth_service.go      # Authentication business logic
│   ├── chat_service.go      # Chat room business logic
│   └── message_service.go   # Message business logic
├── utils/
│   └── jwt.go               # JWT utility functions
├── ws/
│   └── websocket.go         # WebSocket hub and client management
├── go.mod                   # Go module file
└── go.sum                   # Go module checksums
```

## Future Enhancements

- Database persistence (PostgreSQL, MongoDB)
- User presence indicators
- Message reactions and replies
- File sharing capabilities
- Push notifications
- Rate limiting and spam protection
- Message encryption
- Admin panel for room management

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the MIT License.
