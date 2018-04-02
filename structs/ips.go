package structs

import (
	"github.com/garyburd/redigo/redis"
)

type IPs []IP

type IP struct {
	Address string `json:"address"`
}

func GetIPsStruct(db redis.Conn) (ips IPs, err error) {
	addresses, _ := redis.Strings(db.Do("SMEMBERS", "bannedips"))
	for _, address := range addresses {
		ips = append(ips, IP{Address: address})
	}
	// Sort by account creation date
	return ips, nil
}
