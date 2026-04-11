//go:build !prod

package report

import _ "embed"

//go:embed assets/index.dev.html
var embeddedHTML []byte
var NoticeContent = []byte(``)
