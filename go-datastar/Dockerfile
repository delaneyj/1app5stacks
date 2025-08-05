# Stage 1: Build CSS with Tailwind
FROM node:20-alpine AS css-builder

WORKDIR /app

# Copy necessary files for Tailwind
COPY web/postcss.config.js web/tailwind.config.js ./web/
COPY web/gen/css/site.css ./web/gen/css/
COPY web/ ./web/

# Install Tailwind CSS
RUN npm install -g tailwindcss@3 postcss autoprefixer

# Build CSS
WORKDIR /app/web
RUN tailwindcss -i gen/css/site.css -o static/site.css --minify

# Stage 2: Build Go binary
FROM golang:1.24-alpine AS go-builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev sqlite-dev

# Install Go tools
RUN go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest && \
    go install github.com/delaneyj/toolbelt/sqlc-gen-zombiezen@latest && \
    go install github.com/a-h/templ/cmd/templ@latest && \
    go install golang.org/x/tools/cmd/goimports@latest

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Copy the built CSS from css-builder stage
COPY --from=css-builder /app/web/static/site.css ./web/static/site.css

# Copy other static assets (favicon, datastar.js, etc.)
COPY web/static/ ./web/static/

# Generate SQL code
WORKDIR /app/sql
RUN sqlc generate && goimports -w .

# Generate templates
WORKDIR /app
ENV TEMPL_EXPERIMENT=rawgo
RUN templ generate .

# Build the Go binary (now with CSS embedded)
RUN go build -ldflags="-s -w" -o website_bin cmd/site/main.go

# Install UPX and compress the binary
RUN apk add --no-cache upx && \
    upx -9 website_bin

# Stage 3: Final runtime image
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy the binary from go-builder
COPY --from=go-builder /app/website_bin .

# Create data directory for SQLite database
RUN mkdir -p /app/data

# Set working directory for database
WORKDIR /app/data

# The port is configured via PORT env variable in fly.toml
EXPOSE 8080

# Change back to app directory
WORKDIR /app

# Run the binary
CMD ["./website_bin"]