package ts

import (
	"bufio"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	docker "github.com/fsouza/go-dockerclient"
)

// client provides an interface to all the container management functionality
// that ts requires. All required docker-related functionality is implemented
// as methods on the Client struct.
type client struct {
	// Host contains the IP address (or hostname) of the Docker daemon this
	// client is connected to. This allows users of this struct to easily
	// determine where the containers they start will be running.
	Host string

	client *docker.Client
}

func newClient() (*client, error) {

	defaultEndpoint := "unix:///var/run/docker.sock"
	endpoint := os.Getenv("DOCKER_HOST")
	if endpoint == "" {
		endpoint = defaultEndpoint
	}

	// parse the host url from the endpoint address
	host, err := parseEndpoint(endpoint)
	if err != nil {
		return nil, err
	}

	var dClient *docker.Client
	certPath := os.Getenv("DOCKER_CERT_PATH")
	if certPath == "" {
		// connect to the local docker daemon and initialise a new API client
		dClient, err = docker.NewClient(endpoint)
		if err != nil {
			return nil, err
		}
	} else {
		// the docker daemon is configured to use TLS (probably using
		// boot2docker), so we need to load the appropriate certificates and
		// keys
		ca := fmt.Sprintf("%s/ca.pem", certPath)
		cert := fmt.Sprintf("%s/cert.pem", certPath)
		key := fmt.Sprintf("%s/key.pem", certPath)
		dClient, err = docker.NewTLSClient(endpoint, cert, key, ca)
		if err != nil {
			return nil, err
		}
	}

	return &client{client: dClient, Host: host}, nil
}

func parseEndpoint(endpoint string) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	switch u.Scheme {
	case "unix":
		return "127.0.0.1", nil
	case "http", "https", "tcp":
		host, _, err := net.SplitHostPort(u.Host)
		if err != nil {
			if e, ok := err.(*net.AddrError); ok {
				if e.Err == "missing port in address" {
					return u.Host, nil
				}
			}
			return "", err
		}
		return host, nil
	default:
		return "", err
	}
}

// pullImage pulls the latest image for the given repotag from a public
// registry.
func (c *client) pullImage(repoTag string) error {
	// configuration options that get passed to client.PullImage
	repository, tag := docker.ParseRepositoryTag(repoTag)
	pullImageOptions := docker.PullImageOptions{Repository: repository, Tag: tag}
	// pull image from registry
	return c.client.PullImage(pullImageOptions, docker.AuthConfiguration{})
}

// removeContainer removes a single existing container.
func (c *client) removeContainer(cID string) error {
	// force removal of the container along with any volumes it owns.
	rcOptions := docker.RemoveContainerOptions{ID: cID, RemoveVolumes: true, Force: true}
	if err := c.client.RemoveContainer(rcOptions); err != nil {
		return err
	}

	return nil
}

// newUUID generates a random UUID according to RFC 4122
func newUUID() (string, error) {
	uuid := make([]byte, 16)
	n, err := io.ReadFull(rand.Reader, uuid)
	if n != len(uuid) || err != nil {
		return "", err
	}
	// variant bits; see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80
	// version 4 (pseudo-random); see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:]), nil
}

// startContainer creates and starts a new container for the given repotag
func (c *client) startContainer(repoTag string, env []string) (*docker.Container, error) {

	err := c.pullImage(repoTag)
	if err != nil {
		return nil, err
	}

	name, err := newUUID()
	if err != nil {
		return nil, err
	}

	createContainerOptions := docker.CreateContainerOptions{
		Name:       name,
		Config:     &docker.Config{Image: repoTag, Env: env},
		HostConfig: &docker.HostConfig{PublishAllPorts: true},
	}

	container, err := c.client.CreateContainer(createContainerOptions)
	if err != nil {
		return nil, err
	}

	err = c.client.StartContainer(container.ID, &docker.HostConfig{PublishAllPorts: true})
	if err != nil {
		return nil, err
	}

	return c.client.InspectContainer(container.ID)
}

// findPort checks a running container to find the externally exposed port to
// which the given port has been mapped. An error is returned if the port is
// not found.
func findPort(c *docker.Container, p int64) (int64, error) {

	ps := strconv.FormatInt(p, 10)
	for k, v := range c.NetworkSettings.Ports {
		if k.Port() == ps {
			port, err := strconv.ParseInt(v[0].HostPort, 10, 64)
			return port, err
		}
	}
	return 0, errors.New("Port and IP not found")
}

// watchForStringInLogs attaches to the stdout/stderr of the passed-in
// container, checking the log output to see if it contains msg. If msg is
// found, the function returns true. If the timeout expires before message is
// found, the function returns false.
func (c *client) watchForStringInLogs(container *docker.Container, msg string, timeout time.Duration) (bool, error) {

	// attach to the running docker container, reading from both stdout and stderr
	r, w := io.Pipe()
	options := docker.AttachToContainerOptions{
		Container:    container.ID,
		OutputStream: w,
		ErrorStream:  w,
		Stream:       true,
		Stdout:       true,
		Stderr:       true,
		Logs:         true,
	}
	go c.client.AttachToContainer(options)

	// kick off a goroutine to read lines of log output as they're produced,
	// checking for the msg we're looking for in each line. If the msg is
	// found, it is sent over the provided channel, otherwise an error message
	// (as a string) is sent over the channel.
	cFound := make(chan string, 1)
	go func(reader io.Reader, ch chan string) {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			if strings.Contains(scanner.Text(), msg) {
				ch <- msg
				return
			}
		}
		if err := scanner.Err(); err != nil {
			ch <- err.Error()
			return
		}
	}(r, cFound)

	// block while waiting for a message from the channel, but only as long as
	// the configured timeout.
	select {
	case res := <-cFound:
		// if the string received is the same as the message we're looking for
		// then we return true, otherwise there was an error and we reconstruct
		// the error value before returning it to the caller.
		if res == msg {
			return true, nil
		}
		return false, errors.New(res)
	case <-time.After(timeout):
		return false, fmt.Errorf("timeout exceeded, string not found: \"%s\"", msg)
	}

}
