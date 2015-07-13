ts (teststore)
==============

[![Circle CI](https://circleci.com/gh/paddycarey/ts.svg?style=svg)](https://circleci.com/gh/paddycarey/ts)
[![GoDoc](https://godoc.org/github.com/paddycarey/ts?status.svg)](https://godoc.org/github.com/paddycarey/ts)

ts (teststore) is a small Go library that uses Docker to simplify launching
temporary database/cache/storage servers for testing purposes.

**IMPORTANT NOTE:** ts is a work-in-progress and should be treated as such.
Please file issues where appropriate.

ts works with both local docker installs and with boot2docker. Usage from
inside a container is not yet supported, but should be possible so long as the
container has the right privileges.


## Supported Backends

- [Redis][]
- [InfluxDB][]
- [PostgreSQL][]
- More soon...


## Example Usage

Each of teststore's backends implement the same simple API. For example, a
simple test that uses the Redis backend with the [redigo][] library might look
like:

```golang
func TestRedis(t *testing.T) {
	store, err := ts.NewRedis()
	if err != nil {
		t.Error(err.Error())
		return
	}
	defer store.Shutdown()

	u := store.URL()
	conn, err := redis.Dial(u.Scheme, u.Host)
	if err != nil {
		t.Error(err.Error())
		return
	}
	defer conn.Close()

	// set an integer value in redis and read it back
	conn.Do("SET", "k1", 1)
	n, err := redis.Int(conn.Do("GET", "k1"))
	if err != nil {
		t.Error(err.Error())
		return
	}

	if n != 1 {
		t.Error(fmt.Errorf("Redis error: Expected 1, got %d", n))
		return
	}
}
```

See [GoDoc][] for full library documentation.


### Copyright & License

- Copyright © 2015 Patrick Carey (https://github.com/paddycarey)
- Licensed under the **MIT** license.


[GoDoc]: https://godoc.org/github.com/paddycarey/ts
[InfluxDB]: https://influxdb.com/
[PostgreSQL]: http://www.postgresql.org/
[Redis]: http://redis.io/
[redigo]: https://github.com/garyburd/redigo
