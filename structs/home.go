package structs

import (
	"github.com/garyburd/redigo/redis"
)

type HomeElements struct {
	Clients
	SystemUsage    float64
	SystemCapacity float64
}

func NewHomeElements(db redis.Conn) (*HomeElements, error) {
	clients, err := DefaultSortClients(db)
	if err != nil {
		return &HomeElements{}, err
	}

	totalcapacity, err := redis.Float64(db.Do("HGET", "system", "totalcapacity"))
	if err != nil {
		return &HomeElements{}, err
	}
	using, err := redis.Float64(db.Do("HGET", "system", "using"))
	if err != nil {
		return &HomeElements{}, err
	}

	return &HomeElements{Clients: clients, SystemUsage: using, SystemCapacity: totalcapacity}, nil
}
