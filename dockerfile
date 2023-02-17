# Use an official Go runtime as a parent image
FROM golang:1.15-alpine AS build

# Set the working directory to /app
WORKDIR /app

# Copy the current directory contents into the container at /app
COPY . /app

# Build the application
RUN go build -o real-time-chat

# Use an official Redis image as a parent image
FROM redis:6.2-alpine

# Set the working directory to /app
WORKDIR /app

# Copy the application binary from the build image
COPY --from=build /app/real-time-chat .

# Expose port 8080 for the HTTP server
EXPOSE 8080

# Start the application
CMD [ "./real-time-chat" ]
