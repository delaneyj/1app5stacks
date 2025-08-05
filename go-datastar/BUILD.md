# Build Instructions

## Docker Build (Multi-Stage)

The project includes a multi-stage Dockerfile that:
1. Builds the Go binary with all dependencies
2. Builds CSS with Tailwind CLI
3. Creates a minimal runtime image

### To build:
```bash
docker build --network host -t go-datastar-app .
```

**Note:** The `--network host` flag is required to avoid Docker network interface issues during package installation.

### To run:
```bash
docker run -p 4321:4321 go-datastar-app
```

### Using Docker Compose:
```bash
docker-compose up --build
```

## Manual Build Steps

If Docker build fails, you can build manually:

### Prerequisites:
- Go 1.24+ (required for delaneyj/toolbelt dependency)
- Node.js 20+
- Task (taskfile.dev)

### Build commands:
```bash
# Install tools
task tools

# Generate SQL code
task sqlc

# Generate templates
task templ

# Build CSS
task css

# Build and run
task site
```

### Production build:
```bash
# Build optimized binary
go build -ldflags="-s -w" -o website_bin cmd/site/main.go

# Or with UPX compression
task upx
```

## Fly.io Deployment

### Prerequisites:
- Fly CLI installed (`curl -L https://fly.io/install.sh | sh`)
- Authenticated with Fly (`fly auth login`)

### First-time setup:
```bash
# Setup Fly app and create volume
task fly-setup
```

### Deploy:
```bash
# Deploy to Fly.io
task fly-deploy
```

### Manage app:
```bash
# SSH into app to manage database
fly ssh console

# View volumes
fly volumes list
```

### Monitor:
```bash
# View logs
fly logs

# Check app status
fly status

# Scale app
fly scale count 2
```