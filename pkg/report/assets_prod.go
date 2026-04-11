//go:build prod

package report

import _ "embed"

//go:embed assets/index.html
var embeddedHTML []byte
