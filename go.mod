module github.com/dgraph-io/ristretto

go 1.23

require (
	github.com/cespare/xxhash/v2 v2.3.0
	github.com/dgryski/go-farm v0.0.0-20200201041132-a6ae2369ad13
	github.com/dustin/go-humanize v1.0.1
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.9.0
	golang.org/x/sys v0.26.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// we retract v1.0.0 because v0.2.0 is not backwards compatible with v1.0.0.
// The users should upgrade to v2.0.0 instead once released.
retract v1.0.0

// need to retract the next release as well
retract v1.0.1
