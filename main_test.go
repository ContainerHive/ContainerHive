package containerhive_test

import "github.com/kcmvp/archunit"

var cmdLayer = archunit.Packages("Cmd", []string{"cmd/ch"})
var pkgLayer = archunit.Packages("Pkg", []string{"pkg/..."})
var internalLayer = archunit.Packages("Internal", []string{"internal/..."})
