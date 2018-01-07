package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gilgameshskytrooper/bigdisk/utils"
)

func main() {
	fs := http.FileServer(http.Dir(utils.Pwd() + "/public"))
	http.Handle("/", fs)
	log.Println("Listening...")
	// if tlserr != nil {
	// os.Exit(1)
	// log.Fatal(http.ListenAndServe(":8080", nil))
	// }
	if utils.Exists(utils.Pwd()+"/server.crt") && utils.Exists(utils.Pwd()+"/server.key") {
		log.Fatal(http.ListenAndServeTLS(":8080", utils.Pwd()+"/server.crt", utils.Pwd()+"/server.key", nil))
	} else {
		fmt.Println("If you want the program to utilize TLS (i.e. host an encrypted HTTPS front end, please do the following in command line in the same directory as the bigdisk executable to first create a private self-signed rsa key, then a public key (x509) key based on the private key:\n\topenssl genrsa -out server.key 2048\n\topenssl req -new -x509 -sha256 -key server.key -out server.crt -days 3650\nThen make sure you finish filling in the details asked in command line.\n\nFor now, unencrypted http will be used.")
		log.Fatal(http.ListenAndServe(":8080", nil))
	}
}
