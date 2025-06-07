//go:generate sh -c "cd web && pnpm install && npm run build"

package proxy

import "embed"

//go:embed web/build/client/*
var webFs embed.FS
