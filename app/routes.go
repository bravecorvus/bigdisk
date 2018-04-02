package app

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/gilgameshskytrooper/bigdisk/crypto"
	"github.com/gilgameshskytrooper/bigdisk/email"
	"github.com/gilgameshskytrooper/bigdisk/structs"
	"github.com/gilgameshskytrooper/bigdisk/utils"
	"github.com/gorilla/mux"
	"github.com/yosssi/ace"
	"golang.org/x/crypto/bcrypt"
)

func (app *App) Landing(w http.ResponseWriter, r *http.Request) {
	if app.checkIfIPBanned(r) {
		fmt.Fprintf(w, "<html><p>Your IP has been banned from accessing BigDisk. If you think this is a mistake, please talk to an BigDisk administrator.</p></html>")
		return
	} else {

		_, authenticated := app.authenticateCookie(r)
		if authenticated {
			renewCookie(w, r)
			http.Redirect(w, r, "/home", 302)
			return
		} else {

			template, err := ace.Load("templates/landing", "", nil)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if err = template.Execute(w, nil); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

		}

	}

}

func (app *App) NewLanding(w http.ResponseWriter, r *http.Request) {
	if app.checkIfIPBanned(r) {
		fmt.Fprintf(w, "<html><p>Your IP has been banned from accessing BigDisk. If you think this is a mistake, please talk to an BigDisk administrator.</p></html>")
		return
	} else {

		template, err := ace.Load("templates/newlanding", "", nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err = template.Execute(w, nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	}

}

func (app *App) ResetLanding(w http.ResponseWriter, r *http.Request) {
	if app.checkIfIPBanned(r) {
		fmt.Fprintf(w, "<html><p>Your IP has been banned from accessing BigDisk. If you think this is a mistake, please talk to an BigDisk administrator.</p></html>")
		return
	} else {

		template, err := ace.Load("templates/resetlanding", "", nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err = template.Execute(w, nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

}

func (app *App) Login(w http.ResponseWriter, r *http.Request) {

	username := r.FormValue("username")
	pass := r.FormValue("password")
	target := "/"

	if app.authenticateLogin(username, pass, r) {
		app.setSession(username, w, r)
		target = "/home"
	} else {

		failed := getFailedLogins(r)

		if failed == 100 {
			initializeFailedLogins(w, r)
			http.Redirect(w, r, target, 302)
			return
		} else {
			if failed >= 5 {
				ip, iperr := getIPFromRequest(r)
				if iperr != nil {
					log.Println("iperr", iperr.Error())
				}
				_, err := app.DB.Do("SADD", "bannedips", ip)
				if err != nil {
					log.Println("Error", "_, err = app.DB.Do(\"SADD\", \"bannedips\", ip)", err.Error())
				}
				fmt.Fprintf(w, "<html><p>Your IP has been banned from accessing BigDisk. If you think this is a mistake, please talk to an BigDisk administrator.</p></html>")
				return
			} else {
				app.incrementFailedLogin(w, r)
				http.Redirect(w, r, target, 302)
				return
			}
		}

	}
	http.Redirect(w, r, target, 302)
}

func (app *App) Home(w http.ResponseWriter, r *http.Request) {
	username, authenticated := app.authenticateCookie(r)

	if authenticated {

		app.DB.Do("HSET", "logins", username+":lastlogin", time.Now().Format(time.UnixDate))
		renewCookie(w, r)

		if username == "admin" {

			template, err := ace.Load("templates/superadminhome", "", nil)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			homeelements, err := structs.NewHomeElements(app.DB)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if err = template.Execute(w, homeelements); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

		} else {

			template, err := ace.Load("templates/home", "", nil)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			homeelements, err := structs.NewHomeElements(app.DB)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if err = template.Execute(w, homeelements); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

	} else {
		app.clearSession(w, r)
		http.Redirect(w, r, "/", 302)
	}

}

func (app *App) Logout(w http.ResponseWriter, r *http.Request) {
	app.clearSession(w, r)
	http.Redirect(w, r, "/", 302)
	return
}

func (app *App) NewSecretLink(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	publiclink := vars["publiclink"]
	secretlink := vars["secretlink"]
	_, authenticated := app.authenticateCookie(r)
	if authenticated {
		renewCookie(w, r)
		_, err := app.DB.Do("HSET", "clients", publiclink+":secretlink", secretlink)
		if err != nil {
			w.WriteHeader(403)
			return
		}
		admins, _ := redis.Strings(app.DB.Do("SMEMBERS", "emails:"+publiclink))
		for _, admin := range admins {
			status, _ := email.SendEmail(
				admin+"@stolaf.edu",
				"Your BigDisk secret link has been changed. Please modify your POST requests to use the new link",
				"The new secure token is "+vars["secretlink"]+".\n\nThe location of where the files are served have not changed, but the upload endpoint will now be "+app.URL+"/upload/"+publiclink+"/"+secretlink+" while the delete endpoint is now at "+app.URL+"/delete/"+publiclink+"/"+secretlink+"/[filename]")
			if !status {
				w.WriteHeader(403)
				return
			}
		}
		w.WriteHeader(200)
	}

}

func (app *App) GetEmails(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_, authenticated := app.authenticateCookie(r)
	if authenticated {
		list, _ := redis.Strings(app.DB.Do("SMEMBERS", "emails:"+vars["publiclink"]))
		json.NewEncoder(w).Encode(structs.Emails{Emails: list})
		return
	}
}

func (app *App) RemoveEmail(w http.ResponseWriter, r *http.Request) {
	_, authenticated := app.authenticateCookie(r)
	if authenticated {
		vars := mux.Vars(r)
		_, _ = app.DB.Do("SREM", "emails:"+vars["publiclink"], vars["email"])
	}
}

func (app *App) ChangeCapacity(w http.ResponseWriter, r *http.Request) {
	_, authenticated := app.authenticateCookie(r)
	if authenticated {

		// Get gorilla mux slugs
		vars := mux.Vars(r)
		publiclink := vars["publiclink"]
		newcapacitystring := vars["newcapacity"]
		// In order to get the difference between the current capacity limit and the old capacity limit (in order to subtract the difference from the system's), we will get the old and new values as floats

		newcapacity, err := strconv.ParseFloat(newcapacitystring, 64)
		if err != nil {
			log.Println(err.Error())
		}

		oldcapacity, err := redis.Float64(app.DB.Do("HGET", "clients", publiclink+":totalcapacity"))
		if err != nil {
			log.Println(err.Error())
		}

		// Change the values in the capacity string
		_, _ = app.DB.Do("HSET", "clients", publiclink+":totalcapacity", newcapacitystring)
		// Get the difference between new and old
		difference := newcapacity - oldcapacity
		systemusage, err := redis.Float64(app.DB.Do("HGET", "system", "using"))
		if err != nil {
			log.Println(err.Error())
		}
		systemusage = systemusage + difference
		_, err = app.DB.Do("HSET", "system", "using", systemusage)
	}

}

func (app *App) AddEmail(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	publiclink := vars["publiclink"]
	username := vars["email"]
	_, authenticated := app.authenticateCookie(r)
	if authenticated {
		renewCookie(w, r)
		_, _ = app.DB.Do("SADD", "emails:"+publiclink, username)
		secretlink, err := redis.String(app.DB.Do("HGET", "clients", publiclink+":secretlink"))
		if err != nil {
			log.Println("Failed to get clients secret link")
		}
		email.SendEmail(username+"@stolaf.edu",
			"You have been added as an admin for the application "+publiclink,
			"The secure token for the app is "+secretlink+".\n\nThe location of the files you upload will be served at "+app.URL+"/files/"+publiclink+"/[filename].\n\nIn The upload POST endpoint for this app is at "+app.URL+"/upload/"+publiclink+"/"+secretlink+" while the delete POST endpoint for the app is at "+app.URL+"/delete/"+publiclink+"/"+secretlink+"/[filename].\n\nThe easiest way to test out file uploads/deletes is to use curl from the command line of a Linux/Mac machine. Try \"curl -F 'file=@file.mp4' "+app.URL+"/upload/"+publiclink+"/"+secretlink+"\" (Where your filename is called file.mp4). You should be able to load the file in your web browser by going to "+app.URL+"/files/"+publiclink+"/file.mp4\n\nTo delete the same file, use the command \"curl -X POST "+app.URL+"/delete/"+publiclink+"/"+secretlink+"/file.mp4\". Posting to both of these endpoints will return a JSON object which can be parsed by your program. The JSON object consists of the element Status (which is a bool) and the Message (string). Debug your API calls using the return values encapsulated in the JSON object.\n\nGiven these curl commands, you can just Google around for the equivalent conversion of these commands to a language of your choice to use in your application which requires file uploading.")

	}
}

func (app *App) ChangePassword(w http.ResponseWriter, r *http.Request) {
	username, authenticated := app.authenticateCookie(r)
	if authenticated {
		renewCookie(w, r)
		vars := mux.Vars(r)
		oldpassword := vars["oldpassword"]
		newpassword := vars["newpassword"]

		hashedoldpassword, err := redis.String(app.DB.Do("HGET", "logins", username+":password"))

		valid := bcrypt.CompareHashAndPassword([]byte(hashedoldpassword), []byte(oldpassword))
		if valid != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		newpasswordhash, err := bcrypt.GenerateFromPassword([]byte(newpassword), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, _ = app.DB.Do("HSET", "logins", username+":password", newpasswordhash)
		w.WriteHeader(200)
	}
}

func (app *App) AddNewAdmin(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_, authenticated := app.authenticateCookie(r)
	if authenticated {

		username := vars["username"]
		link := crypto.GenerateRandomHash(40)
		expiration := time.Now().Add(24 * time.Hour)
		app.PendingAdmins = append(app.PendingAdmins, AddNewAdminStruct{Link: link, AssociatedUser: username + "@stolaf.edu", Expiration: expiration})
		sentornotsent, _ := email.SendEmail(
			username+"@stolaf.edu",
			"You have been added to St. Olaf BigDisk",
			"Please visit the link "+app.URL+"/newadminregistration/"+link+" to create your login. This link will expire in 24 hours.")
		if sentornotsent {
			w.WriteHeader(200)
		} else {
			w.WriteHeader(403)
		}
	}
}

func (app *App) NewAdminRegistration(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	secret := vars["secret"]
	validlink := false
	email := ""
	for index, pendingadmin := range app.PendingAdmins {
		if pendingadmin.Link == secret {
			if pendingadmin.Expiration.Sub(time.Now()) < 0 {
				app.PendingAdmins = append(app.PendingAdmins[:index], app.PendingAdmins[index+1:]...)
				fmt.Fprintf(w, "<html><p>Sorry, the link expired on "+pendingadmin.Expiration.Format(time.RFC822)+".<br />Please ask a BigDisk Admin to send out another account creation request.</p></html>")
				return
			} else {
				validlink = true
				email = pendingadmin.AssociatedUser
				break
			}
		}

	}

	if validlink {
		template, err := ace.Load("templates/newadmin", "", nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		newadminelements := structs.NewAdminElements{AdminEmail: email}

		if err = template.Execute(w, newadminelements); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		fmt.Fprintf(w, "<html><p>Sorry, this link is invalid.</p></html>")
	}

}

func (app *App) RegisterNewAdmin(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	email := vars["username"] + "@stolaf.edu"
	pass := r.FormValue("password")
	hashedpassword, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		log.Println("Couldn't generate hash from password")
	}

	_, err = app.DB.Do("SADD", "admins", email)
	_, err = app.DB.Do("HSET", "logins", email+":password", hashedpassword)
	_, err = app.DB.Do("HSET", "logins", email+":accountcreated", time.Now().Format(time.UnixDate))
	if err != nil {
		log.Println("Couldn't save password to database")
	}
	http.Redirect(w, r, "/newlanding", 302)
}

func (app *App) ResetLogin(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	username := vars["username"]

	link := crypto.GenerateRandomHash(40)
	expiration := time.Now().Add(24 * time.Hour)
	app.PasswordResets = append(app.PasswordResets, PasswordResetStruct{Link: link, AssociatedUser: username + "@stolaf.edu", Expiration: expiration})
	sentornotsent, _ := email.SendEmail(
		username+"@stolaf.edu",
		"BigDisk Password Reset",
		"Please visit the link "+app.URL+"/passwordreset/"+link+" to reset you BigDisk login. This link will expire in 24 hours.")
	if sentornotsent {
		w.WriteHeader(200)
	} else {
		w.WriteHeader(403)
	}
}

func (app *App) PasswordReset(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	secret := vars["secret"]
	validlink := false
	email := ""
	for index, pendingpasswordreset := range app.PasswordResets {
		if pendingpasswordreset.Link == secret {
			if pendingpasswordreset.Expiration.Sub(time.Now()) < 0 {
				app.PasswordResets = append(app.PasswordResets[:index], app.PasswordResets[index+1:]...)
				fmt.Fprintf(w, "<html><p>Sorry, the link expired on "+pendingpasswordreset.Expiration.Format(time.RFC822)+".<br />Please make another password reset request at <a src='"+app.URL+"'>"+app.URL+"</a>.</p></html>")
				return
			} else {
				validlink = true
				email = pendingpasswordreset.AssociatedUser
				break
			}
		}

	}

	if validlink {
		template, err := ace.Load("templates/passwordreset", "", nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		passwordresetelements := structs.PasswordResetElements{AdminEmail: email}
		if err = template.Execute(w, passwordresetelements); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		fmt.Fprintf(w, "<html><p>Sorry, this link is invalid.</p></html>")
	}

}

func (app *App) SubmitNewPassword(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	email := vars["username"] + "@stolaf.edu"
	pass := r.FormValue("password")
	hashedpassword, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		log.Println("Couldn't generate hash from password")
	}
	_, err = app.DB.Do("HSET", "logins", email+":password", hashedpassword)
	if err != nil {
		log.Println("Couldn't save password to database")
	}
	http.Redirect(w, r, "/resetlanding", 302)
}

func (app *App) ModifySystemCapacity(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_, authenticated := app.authenticateCookie(r)
	if authenticated {
		newcapacity := vars["newcapacity"]
		_, err := app.DB.Do("HSET", "system", "totalcapacity", newcapacity)
		if err != nil {
			log.Println(err.Error())
			w.WriteHeader(403)
			return
		}
		w.WriteHeader(200)
		return
	}
}

func (app *App) AddNewApp(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_, authenticated := app.authenticateCookie(r)
	if authenticated {
		publiclink := vars["publiclink"]
		secretlink := vars["secretlink"]
		totalcapacity := vars["totalcapacity"]
		totalcapacityfloat, err := strconv.ParseFloat(totalcapacity, 64)
		if err != nil {
			log.Println(err.Error())
		}

		systemusage, _ := redis.Float64(app.DB.Do("HGET", "system", "using"))
		systemusage = systemusage + totalcapacityfloat

		_, _ = app.DB.Do("HSET", "system", "using", systemusage)
		_, _ = app.DB.Do("SADD", "allclients", publiclink)
		_, _ = app.DB.Do("HSET", "clients", publiclink+":secretlink", secretlink)
		_, _ = app.DB.Do("HSET", "clients", publiclink+":totalcapacity", totalcapacity)
		_, _ = app.DB.Do("HSET", "clients", publiclink+":using", "0")
		mkdirerr := os.Mkdir(utils.Pwd()+"files/"+publiclink, 0755)
		if mkdirerr != nil {
			log.Println("Failed to make directory", utils.Pwd()+"files/"+publiclink)
		}
	}
}

func (app *App) DeleteApp(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_, authenticated := app.authenticateCookie(r)
	if authenticated {
		publiclink := vars["publiclink"]
		appcapacity, err := redis.Float64(app.DB.Do("HGET", "clients", publiclink+":totalcapacity"))
		if err != nil {
			log.Println(err.Error())
		}
		app.DB.Do("HDEL", "clients", publiclink+":secretlink")
		app.DB.Do("HDEL", "clients", publiclink+":using")
		app.DB.Do("HDEL", "clients", publiclink+":totalcapacity")
		systemusage, err := redis.Float64(app.DB.Do("HGET", "system", "using"))
		if err != nil {
			log.Println(err.Error())
		}
		newsystemusage := systemusage - appcapacity
		app.DB.Do("HSET", "system", "using", newsystemusage)
		app.DB.Do("SREM", "allclients", publiclink)
		emails, _ := redis.Strings(app.DB.Do("SMEMBERS", "emails:"+publiclink))
		for i := 0; i < len(emails); i = i + 1 {
			app.DB.Do("SPOP", "emails:"+publiclink)
		}
		rmerr := os.RemoveAll(utils.Pwd() + "files/" + publiclink)
		if rmerr != nil {
			log.Println("failed to rm -rf " + utils.Pwd() + "files/" + publiclink)
		}
	}
}

func (app *App) GetClients(w http.ResponseWriter, r *http.Request) {
	_, authenticated := app.authenticateCookie(r)
	if authenticated {
		clients, err := structs.DefaultSortClients(app.DB)
		if err != nil {
			log.Println(err.Error())
		}
		json.NewEncoder(w).Encode(clients)
	}
}

func (app *App) GetAdmins(w http.ResponseWriter, r *http.Request) {
	username, authenticated := app.authenticateCookie(r)
	if authenticated && username == "admin" {
		admins, err := structs.DefaultSortAdmins(app.DB)
		if err != nil {
			log.Println(err.Error())
		}
		json.NewEncoder(w).Encode(admins)
	}
}

func (app *App) DeleteAdmin(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deleteusername := vars["username"]
	username, authenticated := app.authenticateCookie(r)
	if authenticated && username == "admin" {
		_, err := app.DB.Do("SREM", "admins", deleteusername+"@stolaf.edu")
		if err != nil {
			w.WriteHeader(403)
			return
		}
		_, err = app.DB.Do("HDEL", "logins", deleteusername+"@stolaf.edu:password")
		if err != nil {
			w.WriteHeader(403)
			return
		}
		_, err = app.DB.Do("HDEL", "logins", deleteusername+"@stolaf.edu:lastlogin")
		if err != nil {
			w.WriteHeader(403)
			return
		}
		_, err = app.DB.Do("HDEL", "logins", deleteusername+"@stolaf.edu:accountcreated")
		if err != nil {
			w.WriteHeader(403)
			return
		}
		_, err = app.DB.Do("HDEL", "logins", deleteusername+"@stolaf.edu:token")
		if err != nil {
			w.WriteHeader(403)
			return
		}
		w.WriteHeader(200)
		return
	} else {
		w.WriteHeader(403)
		return
	}
}

func (app *App) GetIPs(w http.ResponseWriter, r *http.Request) {
	username, authenticated := app.authenticateCookie(r)
	if authenticated && username == "admin" {
		ips, err := structs.GetIPsStruct(app.DB)
		if err != nil {
			log.Println(err.Error())
		}
		json.NewEncoder(w).Encode(ips)
		return
	}
}

func (app *App) UnbanIP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ip := vars["ip"]
	username, authenticated := app.authenticateCookie(r)
	if authenticated && username == "admin" {
		_, err := app.DB.Do("SREM", "bannedips", ip)
		if err != nil {
			log.Println(err.Error())
			w.WriteHeader(403)
			return
		}
		w.WriteHeader(200)
		return
	}
}

type request struct {
	Filename string `json:"filename"`
}

type response struct {
	Status  bool
	Message string
}

var results []string

func (app *App) UploadFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	publiclink := vars["publiclink"]
	secretlink := vars["secretlink"]

	savedsecretlink, err := redis.String(app.DB.Do("HGET", "clients", publiclink+":secretlink"))
	if savedsecretlink != secretlink {
		log.Println("Is Not Valid Secret Link")
		json.NewEncoder(w).Encode(response{Status: false, Message: "Invalid secret link. Please ask BigDisk Admin for the link"})
		return
	}

	// Get Quotas
	using, err := redis.Float64(app.DB.Do("HGET", "clients", publiclink+":using"))
	if err != nil {
		log.Println("Error getting quotas")
		json.NewEncoder(w).Encode(response{Status: false, Message: "Database Error"})
		return
	}
	totalcapacity, err := redis.Float64(app.DB.Do("HGET", "clients", publiclink+":totalcapacity"))
	if err != nil {
		log.Println("Error getting quotas")
		json.NewEncoder(w).Encode(response{Status: false, Message: "Database Error"})
		return
	}
	availableBytes := int64((totalcapacity - using) * 1073741824)
	log.Println("availableBytes", availableBytes)

	r.Body = http.MaxBytesReader(w, r.Body, availableBytes)
	file, header, err := r.FormFile("file")
	if err != nil {
		log.Println("Client", publiclink, "attempted to upload a file that is over the allotted disk quota by the app.")
		json.NewEncoder(w).Encode(response{Status: false, Message: "You attempted to upload a file that is over the allotted disk quota by the app. If you need more space, please contact a BigDisk Admin"})
		return
	}

	defer file.Close()

	if _, err = os.Stat(utils.Pwd() + "files/" + publiclink + "/" + header.Filename); err == nil {

		log.Println("Client attempted to upload and save file with same name as an existing file.")
		json.NewEncoder(w).Encode(response{Status: false, Message: "The file you are trying to uploade contains the name of a file which already exists in the system. Make sure to name your files with unique names, or delete the file using the delete endpoint before attempting to upload the file"})
		return

	}

	out, err1 := os.Create(utils.Pwd() + "files/" + publiclink + "/" + header.Filename)
	if err1 != nil {
		json.NewEncoder(w).Encode(response{Status: false, Message: "Couldn't create file" + header.Filename})
		return
	}
	defer out.Close()
	_, err2 := io.Copy(out, file)
	if err2 != nil {
		json.NewEncoder(w).Encode(response{Status: false, Message: err2.Error()})
		return
	}
	fi, _ := out.Stat()
	newusing := using*1073741824 + float64(fi.Size())
	_, err = app.DB.Do("HSET", "clients", publiclink+":using", newusing/1073741824)
	json.NewEncoder(w).Encode(response{Status: true, Message: "File uploaded successfully"})
}

func (app *App) DeleteFile(w http.ResponseWriter, r *http.Request) {
	log.Println("DeleteFile()")
	vars := mux.Vars(r)
	publiclink := vars["publiclink"]
	secretlink := vars["secretlink"]
	filename := vars["filename"]

	savedsecretlink, err := redis.String(app.DB.Do("HGET", "clients", publiclink+":secretlink"))
	if err != nil {
		log.Println("Database Error")
		json.NewEncoder(w).Encode(response{Status: false, Message: "Database error"})
		return
	}
	if savedsecretlink != secretlink {
		log.Println("Is Not Valid Secret Link")
		json.NewEncoder(w).Encode(response{Status: false, Message: "Invalid secret link. Please ask BigDisk Admin for the link"})
		return
	}
	file, err := os.Open(utils.Pwd() + "files/" + publiclink + "/" + filename)
	if err != nil {
		log.Println(err.Error())
	}
	fi, _ := file.Stat()
	using, err := redis.Float64(app.DB.Do("HGET", "clients", publiclink+":using"))
	if err != nil {
		log.Println("Database Error")
		json.NewEncoder(w).Encode(response{Status: false, Message: "Database Error"})
		return
	}
	newusing := (using * 1073741824) - float64(fi.Size())
	file.Close()
	_, err = app.DB.Do("HSET", "clients", publiclink+":using", newusing/1073741824)
	if err != nil {
		log.Println("Database Error")
		json.NewEncoder(w).Encode(response{Status: false, Message: "Database Error"})
		return
	}
	err = os.Remove(utils.Pwd() + "files/" + publiclink + "/" + filename)
	if err != nil {
		log.Println("failed to remove all files for app ", publiclink)
		json.NewEncoder(w).Encode(response{Status: false, Message: "failed to remove all files for app " + publiclink})
	}
	json.NewEncoder(w).Encode(response{Status: true, Message: "Deleting " + filename + " successful"})
}
