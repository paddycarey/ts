package ts

import (
	"errors"
	"fmt"
	"net/url"
	"time"
)

// Redis implements the Store interface, representing a running Redis
// instance.
type Redis struct {
	url *url.URL

	dc          *client
	containerID string
}

// NewRedis starts a temporary Redis instance using Docker, returning a
// Redis struct which contains the required information for clients to interact
// with the server.
func NewRedis() (Store, error) {

	// initialise a new docker client
	dc, err := newClient()
	if err != nil {
		return nil, err
	}

	// start a new container with the official redis instance
	c, err := dc.startContainer("redis:latest", []string{})
	if err != nil {
		return nil, err
	}

	// watch the container's log until redis indicates that it's ready to
	// accept connections
	msg := "ready to accept connections"
	found, err := dc.watchForStringInLogs(c, msg, time.Second*10)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errors.New("Unable to confirm redis instance has started")
	}

	// find the exposed port that docker has mapped to redis' default port
	port, err := findPort(c, 6379)
	if err != nil {
		return nil, err
	}

	u := &url.URL{}
	u.Scheme = "tcp"
	u.Host = fmt.Sprintf("%s:%d", dc.Host, port)

	// initialise and return a new Redis instance
	r := &Redis{
		url:         u,
		dc:          dc,
		containerID: c.ID,
	}
	return r, nil
}

// URL returns a url.URL instance that can be used to interact with the newly
// started Redis instance
func (r *Redis) URL() *url.URL {
	return r.url
}

// Shutdown stops the Redis instance and destroys the Docker container in which
// it was running
func (r *Redis) Shutdown() error {
	return r.dc.removeContainer(r.containerID)
}
