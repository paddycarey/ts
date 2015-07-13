package ts

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

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
	if err != nil {
		return err
	}

	// write 1000 random events into influxdb, all tagged with the name of a shape
	shapes := []string{"circle", "rectangle", "square", "triangle"}
	pts := make([]idbClient.Point, 1000)
	rand.Seed(42)
	for i := 0; i < 1000; i++ {
		pts[i] = idbClient.Point{
			Measurement: "shapes",
			Tags: map[string]string{
				"shape": shapes[rand.Intn(len(shapes))],
			},
			Fields: map[string]interface{}{
				"value": 1,
			},
			Time:      time.Now(),
			Precision: "n",
		}
	}
	bps := idbClient.BatchPoints{Points: pts, Database: strings.TrimPrefix(s.URL().Path, "/"), RetentionPolicy: "default"}
	_, err = con.Write(bps)
	if err != nil {
		return err
	}

	// read data back from influxdb, ensuring the correct total count is returned
	q := idbClient.Query{
		Command:  "SELECT COUNT(value) FROM shapes GROUP BY shape",
		Database: strings.TrimPrefix(s.URL().Path, "/"),
	}
	response, err := con.Query(q)
	if err != nil {
		return err
	}
	if response.Error() != nil {
		return response.Error()
	}
	count := 0
	for _, row := range response.Results[0].Series {
		f, err := strconv.ParseFloat(string(row.Values[0][1].(json.Number)), 64)
		if err != nil {
			return err
		}
		count = count + int(f)
	}
	if count != 1000 {
		return fmt.Errorf("Unexpected count returned: Expected %d, got %d", 1000, count)
	}

	return nil
}

func TestInfluxDB(t *testing.T) {
	testStore(t, NewInfluxDB, checkInfluxDB)
}
