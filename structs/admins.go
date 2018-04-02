package structs

import (
	"log"
	"sort"
	"time"

	"github.com/garyburd/redigo/redis"
)

type Admins []Admin

type Admin struct {
	Username             string    `json:"username"`
	AccountCreated       time.Time `json:"accountcreated"`
	AccountCreatedString string    `json:"accountcreatedstring"`
	LastLogin            time.Time `json:"lastlogin"`
	LastLoginString      string    `json:"lastloginstring"`
}

func (p Admins) Len() int {
	return len(p)
}

func (p Admins) Less(i, j int) bool {
	return p[i].AccountCreated.Before(p[j].AccountCreated)
}

func (p Admins) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func DefaultSortAdmins(db redis.Conn) (admins Admins, err error) {
	administrators, _ := redis.Strings(db.Do("SMEMBERS", "admins"))
	for _, admin := range administrators {
		username := admin
		accountcreatedstring, _ := redis.String(db.Do("HGET", "logins", username+":accountcreated"))
		accountcreated, err := time.Parse(time.UnixDate, accountcreatedstring)
		if err != nil {
			log.Println("can't parse accountcreatedstring as time.Unix time object")
		}
		lastloginstring, _ := redis.String(db.Do("HGET", "logins", username+":lastlogin"))
		lastlogin, err := time.Parse(time.UnixDate, lastloginstring)
		if err != nil {
			log.Println("can't parse lastloginstring as time.Unix time object")
		}
		admins = append(admins, Admin{Username: username, AccountCreated: accountcreated, AccountCreatedString: accountcreatedstring, LastLogin: lastlogin, LastLoginString: lastloginstring})
	}
	// Sort by account creation date
	sort.Sort(admins)
	return admins, nil
}
