package app

import (
	"log"
	"os"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/gilgameshskytrooper/bigdisk/crypto"
	"github.com/gilgameshskytrooper/bigdisk/email"
	"github.com/gilgameshskytrooper/bigdisk/utils"
	"github.com/gorilla/securecookie"
	"golang.org/x/crypto/bcrypt"
)

type App struct {
	DB             redis.Conn
	URL            string
	CookieHandler  *securecookie.SecureCookie
	PasswordResets []PasswordResetStruct
	PendingAdmins  []AddNewAdminStruct
}

type PasswordResetStruct struct {
	Link           string
	AssociatedUser string
	Expiration     time.Time
}

type AddNewAdminStruct struct {
	Link           string
	AssociatedUser string
	Expiration     time.Time
}

func (app *App) Initialize() {
	// db, err := redis.Dial("tcp", ":6379")
	db, err := redis.DialURL(os.Getenv("REDISLOCATION"))
	app.DB = db
	if err != nil {
		log.Println(err.Error())
	}
	_, _ = app.DB.Do("SELECT", "0")

	adminpass := os.Getenv("BIGDISKSUPERADMINPASSWORD")
	adminemail := os.Getenv("BIGDISKSUPERADMINEMAIL")
	if adminpass == "" {
		newpassword := crypto.GenerateRandomHash(40)
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newpassword), bcrypt.DefaultCost)
		if err != nil {
			log.Println(err.Error())
		}
		_, _ = app.DB.Do("HSET", "logins", "admin:password", hashedPassword)
		sentornotsent, status := email.SendEmail(adminemail, "BigDisk superadmin password change", "BigDisk superadmin password was not defined as an environment variable and hence, it has been randomly generated.\n\nThe password is ["+newpassword+"].")
		if !sentornotsent {
			log.Println("Sending email failed at", status)
		}
	} else {
		storedhashedpass, _ := redis.String(app.DB.Do("HGET", "logins", "admin:password"))
		samepassword := bcrypt.CompareHashAndPassword([]byte(storedhashedpass), []byte(adminpass))

		if samepassword != nil {
			newpasswordhash, err := bcrypt.GenerateFromPassword([]byte(adminpass), bcrypt.DefaultCost)
			if err != nil {
				log.Println("Couldn't create password hash from given password")
			}
			_, _ = app.DB.Do("HSET", "logins", "admin:password", newpasswordhash)
			sentornotsent, status := email.SendEmail(adminemail, "BigDisk superadmin password change", "Stored BigDisk admin password differed from the one provided by the BIGDISKSUPERADMINPASSWORD environment variable and hence, it has been changed to be the one parsed from environment variables.\n\nThe password is ["+adminpass+"].")
			if !sentornotsent {
				log.Println("Sending email failed at", status)
			}
		}
		systemcapacity := os.Getenv("BIGDISKSYSTEMCAPICITY")
		if systemcapacity == "" {
			dbsystemcapacity, _ := redis.String(app.DB.Do("HGET", "system", "totalcapacity"))
			if dbsystemcapacity == "" {
				_, _ = app.DB.Do("HSET", "system", "totalcapacity", 100)
			}
		} else {
			_, _ = app.DB.Do("HSET", "system", "totalcapacity", systemcapacity)
		}
	}

	app.CookieHandler = securecookie.New(
		securecookie.GenerateRandomKey(64),
		securecookie.GenerateRandomKey(32))
	app.URL = os.Getenv("BIGDISKURL")
	if app.URL == "" {
		app.URL = "http://localhost:8080"
	}
	using, _ := redis.String(app.DB.Do("HGET", "system", "using"))
	if using == "" {
		calculatedusing := utils.DirSize(utils.Pwd() + "files")
		app.DB.Do("HSET", "system", "using", calculatedusing)
	}
}
