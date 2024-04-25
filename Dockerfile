# Use the official Go image as a base
FROM golang:1.21-alpine AS builder

# Set the current working directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go app
RUN go build -o assessment-tax .

# Start a new stage from scratch
FROM alpine:latest

# Set the working directory to /app
WORKDIR /app

# Copy the binary file from the builder stage to the new stage
COPY --from=builder /app/assessment-tax .

# Command to run the executable
CMD ["./assessment-tax"]