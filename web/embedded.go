package webassets

import "embed"

//go:embed all:cloud-step-web-1g/dist
var FS embed.FS

//go:embed favicon.jpg
var FaviconData []byte
