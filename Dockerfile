ARG GO_VERSION=1.24.2
ARG NODE_VERSION=20

# Build React app
FROM node:${NODE_VERSION}-alpine AS frontend
WORKDIR /app/web

# Install pnpm
RUN npm install -g pnpm

# Copy package files first for better caching
COPY web/package.json web/pnpm-lock.yaml ./

# Install dependencies
RUN pnpm install

# Copy web source code excluding node_modules
COPY web/app ./app
COPY web/public ./public
COPY web/tsconfig.json web/react-router.config.ts web/vite.config.ts ./

# Build React app
RUN pnpm build

# Build Go app
FROM golang:${GO_VERSION}-alpine AS golang
WORKDIR /app
COPY . .
# Copy the built frontend assets
COPY --from=frontend /app/web/build ./web/build

RUN go mod download
RUN go mod verify

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /proxy cmd/proxy/main.go

# Final image
FROM gcr.io/distroless/static:latest
COPY --from=golang /proxy .
COPY --from=golang /app/config.yml .

EXPOSE 8888

CMD ["/proxy"]
