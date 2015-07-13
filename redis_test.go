package ts

import (
	"fmt"
	"testing"

	"github.com/garyburd/redigo/redis"
)

func checkRedis(s Store) error {
	conn, err := redis.Dial(s.URL().Scheme, s.URL().Host)
	if err != nil {
		return err
	}
	defer conn.Close()

	// set an integer value in redis and read it back
	conn.Do("SET", "k1", 1)
	n, err := redis.Int(conn.Do("GET", "k1"))
	if err != nil {
		return err
	}

	if n != 1 {
		return fmt.Errorf("Redis error: Expected 1, got %d", n)
	}

	return nil
}

func TestRedis(t *testing.T) {
	testStore(t, NewRedis, checkRedis)
}
