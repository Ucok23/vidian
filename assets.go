package vidian

import "embed"

//go:embed all:frontend/dist
var EmbeddedFiles embed.FS
