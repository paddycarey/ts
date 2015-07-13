package ts

import (
	"errors"
	"fmt"
	"net/url"
	"time"
)

// InfluxDB implements the Store interface, representing a running InfluxDB
// instance inside a Docker container.
type InfluxDB struct {
	url *url.URL

	dc          *client
	containerID string
}

// NewInfluxDB starts a temporary InfluxDB instance using Docker, returning an
// InfluxDB struct which contains the required information for clients to
// interact with the server.
func NewInfluxDB() (Store, error) {

	// initialise a new docker client
	dc, err := newClient()
	if err != nil {
		return nil, err
	}

	c, err := dc.startContainer("tutum/influxdb:latest", []string{"PRE_CREATE_DB=testdb"})
	if err != nil {
		return nil, err
	}

	found, err := dc.watchForStringInLogs(c, "Creating database", time.Second*10)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errors.New("Unable to confirm influxdb instance has started")
	}

	port, err := findPort(c, 8086)
	if err != nil {
		return nil, err
	}

	u := &url.URL{}
	u.Scheme = "http"
	u.Host = fmt.Sprintf("%s:%d", dc.Host, port)
	u.Path = "/testdb"

	i := &InfluxDB{
		url:         u,
		dc:          dc,
		containerID: c.ID,
	}
	return i, nil
}

// URL returns a url.URL instance that can be used to interact with the newly
// started InfluxDB instance
func (i *InfluxDB) URL() *url.URL {
	return i.url
}

// Shutdown stops the InfluxDB instance and destroys the Docker container in
// which it was running
func (i *InfluxDB) Shutdown() error {
	return i.dc.removeContainer(i.containerID)
}
