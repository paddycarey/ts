package ts

import (
	"errors"
	"fmt"
	"net/url"
	"time"
)

// Postgres implements the Store interface, representing a running PostgreSQL
// instance.
type Postgres struct {
	url *url.URL

	dc          *client
	containerID string
}

// NewPostgres starts a temporary PostgreSQL instance using Docker, returning a
// Postgres struct which contains the required information for clients to
// interact with the running server.
func NewPostgres() (Store, error) {

	// initialise a new docker client
	dc, err := newClient()
	if err != nil {
		return nil, err
	}

	// start a new container with the official postgres instance
	c, err := dc.startContainer("postgres:latest", []string{
		"POSTGRES_PASSWORD=testpassword",
		"POSTGRES_USER=testy",
	})
	if err != nil {
		return nil, err
	}

	// watch the container's log until postgres indicates that it's ready to
	// accept connections
	msg := "PostgreSQL init process complete; ready for start up."
	found, err := dc.watchForStringInLogs(c, msg, time.Second*30)
	if err != nil {
		return nil, err
	}
	msg = "ready to accept connections"
	found, err = dc.watchForStringInLogs(c, msg, time.Second*30)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errors.New("Unable to confirm PostgreSQL instance has started")
	}

	// find the exposed port that docker has mapped to postgres' default port
	port, err := findPort(c, 5432)
	if err != nil {
		return nil, err
	}

	u := &url.URL{
		Scheme:   "postgres",
		Host:     fmt.Sprintf("%s:%d", dc.Host, port),
		User:     url.UserPassword("testy", "testpassword"),
		Path:     "/testy",
		RawQuery: "sslmode=disable",
	}

	// initialise and return a new Postgres instance
	p := &Postgres{
		url:         u,
		dc:          dc,
		containerID: c.ID,
	}
	return p, nil
}

// URL returns a url.URL instance that can be used to interact with the newly
// started PostgreSQL instance
func (p *Postgres) URL() *url.URL {
	return p.url
}

// Shutdown stops the PostgreSQL instance and destroys the Docker container in
// which it was running
func (p *Postgres) Shutdown() error {
	return p.dc.removeContainer(p.containerID)
}
