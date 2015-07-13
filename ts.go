package ts

import "net/url"

// Store is the common interface implemented by all provided storage backends
type Store interface {
	URL() *url.URL
	Shutdown() error
}
