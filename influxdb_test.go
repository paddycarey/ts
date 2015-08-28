package ts

import (
	"fmt"
	"net/url"
	"testing"

	idbClient "github.com/influxdb/influxdb/client"
)

func checkInfluxDB(s Store) error {

	// construct a client configuration and connect to the influxdb server
	u, err := url.Parse(fmt.Sprintf("%s://%s", s.URL().Scheme, s.URL().Host))
	if err != nil {
		return err
	}
	conf := idbClient.Config{URL: *u}
	con, err := idbClient.NewClient(conf)
	if err != nil {
		return err
	}

	// ping the influx server to make sure it's up
	_, _, err = con.Ping()
	return err
}

func TestInfluxDB(t *testing.T) {
	testStore(t, NewInfluxDB, checkInfluxDB)
}
