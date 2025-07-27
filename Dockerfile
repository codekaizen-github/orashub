FROM golang:1.24 AS base

WORKDIR /usr/src/app

FROM base AS dev

FROM base AS builder

# Add build arguments to specify which component to build and the output binary name
ARG BUILD_COMPONENT=server
ARG BINARY_NAME=app

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod ./
# COPY go.sum ./
RUN go mod download

COPY . .
# Build with CGO disabled for a fully static binary that works well with Alpine
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -v -o /usr/local/bin/${BINARY_NAME} ./${BUILD_COMPONENT}

# Create a final stage for running the application
FROM alpine:3.19 AS final

# Need to redeclare the ARG in this stage to use it
ARG BINARY_NAME=app

# Copy the binary from the builder stage
COPY --from=builder /usr/local/bin/${BINARY_NAME} /usr/local/bin/${BINARY_NAME}

# Install CA certificates for HTTPS connections
RUN apk --no-cache add ca-certificates

# Set the binary name as an environment variable for the CMD instruction
ENV APP_BINARY=${BINARY_NAME}

# Use the environment variable in the CMD
CMD ["/bin/sh", "-c", "/usr/local/bin/${APP_BINARY}"]
