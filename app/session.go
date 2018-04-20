package app

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"

	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/gilgameshskytrooper/bigdisk/crypto"
	"golang.org/x/crypto/bcrypt"
)

func (app *App) getUserName(r *http.Request) (userName string) {
	if cookie, err := r.Cookie("username"); err == nil {
		cookieValue := make(map[string]string)
		if err = app.CookieHandler.Decode("username", cookie.Value, &cookieValue); err == nil {
			userName = cookieValue["username"]
		}
	}
	return userName
}

func (app *App) initializeFailedLogins(w http.ResponseWriter, r *http.Request) {
	value := map[string]string{
		"failedlogins": "1",
	}
	failedlogins, err := app.CookieHandler.Encode("failedlogins", value)
	if err == nil {
		cookie := &http.Cookie{
			Name:  "failedlogins",
			Value: failedlogins,
			Path:  "/",
		}
		http.SetCookie(w, cookie)
	}
}

func (app *App) checkIfIPBanned(r *http.Request) bool {
	ip, err := app.getIPFromRequest(r)
	if err != nil {
		log.Println("Can't get IP from request in function checkIfIPBanned")
		return true
	}
	ipbanned, _ := redis.Bool(app.DB.Do("SISMEMBER", "bannedips", ip))
	if ipbanned {
		return true
	} else {
		return false
	}

}

func (app *App) getFailedLogins(r *http.Request) int {
	fails := 100
	if cookie, err := r.Cookie("failedlogins"); err == nil {
		cookieValue := make(map[string]string)
		if err = app.CookieHandler.Decode("failedlogins", cookie.Value, &cookieValue); err == nil {
			failsstring := cookieValue["failedlogins"]
			fails, strconverr := strconv.Atoi(failsstring)
			if strconverr != nil {
				log.Println("couldn't convert failed attempt string value to integer")
			}
			return fails
		}
	}
	return fails
}

func (app *App) incrementFailedLogin(w http.ResponseWriter, r *http.Request) {
	fails := app.getFailedLogins(r) + 1
	failsstring := strconv.Itoa(fails)

	value := map[string]string{
		"failedlogins": failsstring,
	}
	failedlogins, err := app.CookieHandler.Encode("failedlogins", value)
	if err == nil {
		cookie := &http.Cookie{
			Name:  "failedlogins",
			Value: failedlogins,
			Path:  "/",
		}
		http.SetCookie(w, cookie)
	}
}

func (app *App) setSession(username string, w http.ResponseWriter, r *http.Request) {

	value1 := map[string]string{
		"username": username,
	}

	name, err := app.CookieHandler.Encode("username", value1)
	if err == nil {
		cookie := &http.Cookie{
			Name:  "username",
			Value: name,
			Path:  "/",
		}
		http.SetCookie(w, cookie)
	}

	tomorrow := time.Now().Add(24 * time.Hour)
	value2 := map[string]string{
		"expiration": tomorrow.Format(time.RFC3339),
	}

	expiration, err := app.CookieHandler.Encode("expiration", value2)
	if err == nil {
		cookie := &http.Cookie{
			Name:  "expiration",
			Value: expiration,
			Path:  "/",
		}
		http.SetCookie(w, cookie)
	}

	ip, iperr := app.getIPFromRequest(r)
	if iperr != nil {
		log.Println(iperr)
	}

	value3 := map[string]string{
		"ipaddr": ip,
	}

	ipaddr, err := app.CookieHandler.Encode("ipaddr", value3)
	if err == nil {
		cookie := &http.Cookie{
			Name:  "ipaddr",
			Value: ipaddr,
			Path:  "/",
		}
		http.SetCookie(w, cookie)
	}

	tokenval := crypto.GenerateRandomHash(20)
	encryptedtoken, err := bcrypt.GenerateFromPassword([]byte(tokenval), bcrypt.DefaultCost)
	if err != nil {
		log.Println(err.Error())
	}
	_, _ = app.DB.Do("HSET", "logins", username+":token", encryptedtoken)
	value4 := map[string]string{
		"token": tokenval,
	}

	token, err := app.CookieHandler.Encode("token", value4)
	if err == nil {
		cookie := &http.Cookie{
			Name:  "token",
			Value: token,
			Path:  "/",
		}
		http.SetCookie(w, cookie)
	}

	value5 := map[string]string{
		"failedlogins": "0",
	}
	failedlogins, err := app.CookieHandler.Encode("failedlogins", value5)
	if err == nil {
		cookie := &http.Cookie{
			Name:  "failedlogins",
			Value: failedlogins,
			Path:  "/",
		}
		http.SetCookie(w, cookie)
	}

}

func (app *App) clearSession(w http.ResponseWriter, r *http.Request) {
	username := app.getUserName(r)
	usernamecookie := &http.Cookie{
		Name:   "username",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	}
	http.SetCookie(w, usernamecookie)

	expirationcookie := &http.Cookie{
		Name:   "expiration",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	}
	http.SetCookie(w, expirationcookie)

	tokencookie := &http.Cookie{
		Name:   "token",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	}
	http.SetCookie(w, tokencookie)
	_, _ = app.DB.Do("HDEL", "logins", username+":token")
}

func (app *App) renewCookie(w http.ResponseWriter, r *http.Request) {
	tomorrow := time.Now().Add(24 * time.Hour)
	value := map[string]string{
		"expiration": tomorrow.Format(time.RFC3339),
	}

	expiration, err := app.CookieHandler.Encode("expiration", value)
	if err == nil {
		cookie := &http.Cookie{
			Name:  "expiration",
			Value: expiration,
			Path:  "/",
		}
		http.SetCookie(w, cookie)
	}
}

func (app *App) authenticateLogin(username, password string, r *http.Request) bool {
	ip, err := app.getIPFromRequest(r)
	if err != nil {
		log.Println(err.Error())
	}
	isbanned, _ := redis.Bool(app.DB.Do("SISMEMBER", "bannedips", ip))
	if isbanned {
		return false
	}
	savedhashpassword, _ := redis.String(app.DB.Do("HGET", "logins", username+":password"))
	valid := bcrypt.CompareHashAndPassword([]byte(savedhashpassword), []byte(password))
	if valid != nil {
		log.Println("Is Not Valid Password")
		return false
	}
	return true
}

func (app *App) authenticateCookie(r *http.Request) (string, bool) {
	ip, err := app.getIPFromRequest(r)
	if err != nil {
		log.Println(err.Error())
	}
	isbanned, _ := redis.Bool(app.DB.Do("SISMEMBER", "bannedips", ip))
	if isbanned {
		return "", false
	}

	// Decode value of expiration from client cookie, check to see if the cookies expired
	var expirationstring string
	var expiration time.Time
	if expirationcookie, err := r.Cookie("expiration"); err == nil {
		expirationcookieValue := make(map[string]string)
		if err = app.CookieHandler.Decode("expiration", expirationcookie.Value, &expirationcookieValue); err == nil {
			expirationstring = expirationcookieValue["expiration"]
		}
	} else {
		return "", false
	}
	expiration, err = time.Parse(time.RFC3339, expirationstring)
	if err != nil {
		log.Println("couldn't parse expiration")
		return "", false
	}
	now := time.Now()
	if expiration.Sub(now).Seconds() < 0 {
		return "", false
	}

	// Decode the username from client cookie store it in variable username
	username := app.getUserName(r)

	// Decode IP address and check against the associated IP address in the redis database to ensure that the IP is the same
	ipfromcookie := ""
	if ipcookie, err := r.Cookie("ipaddr"); err == nil {
		ipcookieValue := make(map[string]string)
		if err = app.CookieHandler.Decode("ipaddr", ipcookie.Value, &ipcookieValue); err == nil {
			ipfromcookie = ipcookieValue["ipaddr"]
		}
	} else {
		return "", false
	}

	ipfromrequest, iperr := app.getIPFromRequest(r)
	if iperr != nil {
		log.Println(err.Error())
		return "", false
	}

	// Decode token from client cookie store and store it in variable token
	tokenfromcookie := ""
	if tokencookie, err := r.Cookie("token"); err == nil {
		tokencookieValue := make(map[string]string)
		if err = app.CookieHandler.Decode("token", tokencookie.Value, &tokencookieValue); err == nil {
			tokenfromcookie = tokencookieValue["token"]
		}
	} else {
		return "", false
	}

	// Get hashed token from database
	tokenfromdb, _ := redis.String(app.DB.Do("HGET", "logins", username+":token"))

	// Use bcrypt.CompareHashAndPassword() method in order to compare the encrypted token hash received from the database with the token extracted from the users cookie
	tokenerror := bcrypt.CompareHashAndPassword([]byte(tokenfromdb), []byte(tokenfromcookie))

	// Massive if clause to make sure that everything about the cookie is just right
	// 1) Since we extract the user's IP, we can compare that with the one stored in the DB to see if they match
	// 2) The first thing this function checks is to see if the token expired or not (set to expire after 24 hours of nonusage)
	// 3) The token must match that stored in the DB (since the token is generated randomly during the inital login, it fairly difficult to brute force it
	if ipfromcookie == ipfromrequest && ipfromcookie != "" && tokenerror == nil && tokenfromdb != "" && tokenfromcookie != "" && username != "" {
		return username, true
	} else {
		return "", false
	}

}

func (app *App) getIPFromRequest(req *http.Request) (string, error) {
	ip, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return "", fmt.Errorf("userip: %q is not IP:port", req.RemoteAddr)
	}

	userIP := net.ParseIP(ip)
	if userIP == nil {
		return "", fmt.Errorf("userip: %q is not IP:port", req.RemoteAddr)
	}
	return userIP.String(), nil
}
