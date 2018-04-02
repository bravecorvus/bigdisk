package structs

import (
	"sort"
	"strconv"
	"strings"

	"github.com/garyburd/redigo/redis"
)

type Alphabetic []string

func (list Alphabetic) Len() int { return len(list) }

func (list Alphabetic) Swap(i, j int) { list[i], list[j] = list[j], list[i] }

func (list Alphabetic) Less(i, j int) bool {
	var si string = list[i]
	var sj string = list[j]
	var si_lower = strings.ToLower(si)
	var sj_lower = strings.ToLower(sj)
	if si_lower == sj_lower {
		return si < sj
	}
	return si_lower < sj_lower
}

type Clients struct {
	Clients []Client `json:"clients"`
}

type Client struct {
	PublicLink    string  `json:"publiclink"`
	SecretLink    string  `json:"secretlink"`
	TotalCapacity float64 `json:"totalcapacity"`
	Using         float64 `json:"using"`
}

func DefaultSortClients(db redis.Conn) (clients Clients, err error) {
	publiclinks, _ := redis.Strings(db.Do("SMEMBERS", "allclients"))
	sort.Sort(Alphabetic(publiclinks))
	for _, publiclink := range publiclinks {
		secretlink, _ := redis.String(db.Do("HGET", "clients", publiclink+":secretlink"))
		totalcapacity, _ := redis.String(db.Do("HGET", "clients", publiclink+":totalcapacity"))
		using, _ := redis.String(db.Do("HGET", "clients", publiclink+":using"))
		floattotalcapacity, _ := strconv.ParseFloat(totalcapacity, 64)
		floatusing, _ := strconv.ParseFloat(using, 64)
		clients.Clients = append(clients.Clients, Client{PublicLink: publiclink, SecretLink: secretlink, TotalCapacity: floattotalcapacity, Using: floatusing})
	}

	return clients, nil
}
