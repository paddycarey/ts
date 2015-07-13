package ts

import (
	"testing"
)

// validator is a function type used by tests to verify that a given storage
// backend is active and working correctly. Normally, functions implementing
// this interface should perform a simple write/read cycle using the backend to
// verify its operation.
type validator func(Store) error

// newStore is a common type implemented by the constructors for all provided
// storage backends. It's useful for internal testing, but not particularly for
// use by clients, so it's not exported.
type newStore func() (Store, error)

// testStore is a common function used for testing all available storage
// backends, it provides common setup and teardown functionality and runs tests
// common to all backends.
func testStore(t *testing.T, ns newStore, v validator) {

	store, err := ns()
	if err != nil {
		t.Error(err.Error())
		return
	}
	defer store.Shutdown()

	err = v(store)
	if err != nil {
		t.Error(err.Error())
		return
	}

}
