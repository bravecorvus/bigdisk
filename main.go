package main

import (
	"net/http"

	"github.com/gilgameshskytrooper/bigdisk/app"
	"github.com/gilgameshskytrooper/bigdisk/utils"
	"github.com/gorilla/mux"
	"github.com/robfig/cron"
	"github.com/urfave/negroni"
)

var (
	globals app.App
)

func init() {
	globals.Initialize()
}

func main() {
	defer globals.DB.Close()

	c := cron.New()
	c.AddFunc("@midnight", func() { globals.RemoveOldRequests() })
	c.Start()

	r := mux.NewRouter()

	// Admin Endpoints
	r.HandleFunc("/", globals.Landing).Methods("GET")
	r.HandleFunc("/newlanding", globals.NewLanding).Methods("GET")
	r.HandleFunc("/resetlanding", globals.ResetLanding).Methods("GET")
	r.HandleFunc("/login", globals.Login).Methods("GET")
	r.HandleFunc("/logout", globals.Logout).Methods("GET")
	r.HandleFunc("/home", globals.Home).Methods("GET")
	r.PathPrefix("/public/").Handler(http.StripPrefix("/public/", http.FileServer(http.Dir(utils.Pwd()+"public/"))))
	r.HandleFunc("/newsecretlink/{publiclink}/{secretlink}", globals.NewSecretLink).Methods("POST")
	r.HandleFunc("/emails/{publiclink}", globals.GetEmails).Methods("GET")
	r.HandleFunc("/addemail/{publiclink}/{email}", globals.AddEmail).Methods("POST")
	r.HandleFunc("/rmemail/{publiclink}/{email}", globals.RemoveEmail).Methods("POST")
	r.HandleFunc("/changecapacity/{publiclink}/{newcapacity}", globals.ChangeCapacity).Methods("POST")
	r.HandleFunc("/changepassword/{oldpassword}/{newpassword}", globals.ChangePassword).Methods("POST")
	r.HandleFunc("/addnewadmin/{username}", globals.AddNewAdmin).Methods("POST")
	r.HandleFunc("/newadminregistration/{secret}", globals.NewAdminRegistration).Methods("GET")
	r.HandleFunc("/registernewadmin/{username}", globals.RegisterNewAdmin).Methods("GET")
	r.HandleFunc("/resetlogin/{username}", globals.ResetLogin).Methods("POST")
	r.HandleFunc("/passwordreset/{secret}", globals.PasswordReset).Methods("GET")
	r.HandleFunc("/resetlogin/{username}", globals.PasswordReset).Methods("GET")
	r.HandleFunc("/submitnewpassword/{username}", globals.SubmitNewPassword).Methods("GET")
	r.HandleFunc("/modifysystemcapacity/{newcapacity}", globals.ModifySystemCapacity).Methods("POST")
	r.HandleFunc("/addnewapp/{publiclink}/{secretlink}/{totalcapacity}", globals.AddNewApp).Methods("POST")
	r.HandleFunc("/deleteapp/{publiclink}", globals.DeleteApp).Methods("POST")
	r.HandleFunc("/getclients", globals.GetClients).Methods("GET")
	r.HandleFunc("/getadmins", globals.GetAdmins).Methods("GET")
	r.HandleFunc("/deleteadmin/{username}", globals.DeleteAdmin).Methods("POST")
	r.HandleFunc("/getbannedips", globals.GetIPs).Methods("GET")
	r.HandleFunc("/unbanip/{ip}", globals.UnbanIP).Methods("POST")

	// Client Endpoints
	r.PathPrefix("/files/").Handler(http.StripPrefix("/files/", http.FileServer(http.Dir(utils.Pwd()+"files/"))))
	r.HandleFunc("/upload/{publiclink}/{secretlink}", globals.UploadFile).Methods("POST")
	r.HandleFunc("/delete/{publiclink}/{secretlink}/{filename}", globals.DeleteFile).Methods("POST")

	n := negroni.Classic()
	n.UseHandler(r)
	err := http.ListenAndServe(":8080", n)
	if err != nil {
		panic(err.Error())
	}
}
