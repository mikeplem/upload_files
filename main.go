package main

import (
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/satori/go.uuid"
)

// ======================

// ConfigFile holds the user supplied configuration file - it is placed here since it is a global
var ConfigFile *string

// Config is the structure of the TOML config structure
var Config tomlConfig

type tomlConfig struct {
	Listen listenconfig `toml:"listen"`
	LDAP   ldapconfig   `toml:"ldap"`
	Upload uploadconfig `toml:"upload"`
}

type listenconfig struct {
	SSL  bool
	Cert string
	Key  string
	Port int
}

type ldapconfig struct {
  UseLDAP      bool
	Host         string
	Port         int
	Base         string
	GroupBase    string
	BindDN       string
	BindPassword string
	GroupName    string
}

type uploadconfig struct {
	Path string
}

// ======================

const loginTpl = `
<!DOCTYPE html>
<html>
<head>
	<title>TV uploadFile Uploads</title>
</head>

<body bgcolor='#3284D6'>
	<h1>TV Video Uploads</h1>
	Login with your LDAP credentials.
	<p />
	<form method="POST" action="/login">
		Please fill in your LDAP username
		<input type="text" name="username" placeholder="username">
		<br>
		Please fill in your LDAP password
		<input type="password" name="password" placeholder="password">
		<p />
		<input type="submit" value="Login">
	</form>
</body>
</html>`

const uploadTpl = `
<!DOCTYPE html>
<html>
<head>
	<title>Upload File</title>
</head>

<body bgcolor='#3284D6'>
	<form method="POST" enctype="multipart/form-data" action="/upload">
	  <b>Select a file to upload</b><p />
	  <input type="file" name="fileupload" value="fileupload" id="fileupload">
		<p />
	  <input type="submit" value="Upload File">
	</form>
</body>
</html>`

const doneTpl = `
<!DOCTYPE html>
<html>
<head>
	<title>File Uploaded</title>
	<meta http-equiv="refresh" content="3;url=/choose">
</head>
<body bgcolor='#3284D6'>
<b>File Uploaded</b>
<p />
You will be redirected to the file chooser in 3 seconds.
</body>
</html>`

// ========================


func homePage(res http.ResponseWriter, req *http.Request) {
	if Config.LDAP.UseLDAP {

		t, err := template.New("webpage").Parse(loginTpl)
		if err != nil {
			log.Print(err)
			return
		}

		err = t.Execute(res, Config)
		if err != nil {
			log.Print("execute: ", err)
			return
		}
	} else {

		// create session token for user
		sessionToken, err := uuid.NewV4()
		if err != nil {
			log.Printf("sessionToken failed to create: %s", err)
			return
		}

		// Finally, we set the client cookie for "session_token" as the session token we just generated
		// we also set an expiry time of 120 seconds
		http.SetCookie(res, &http.Cookie{
			Name:    "session_token",
			Value:   sessionToken.String(),
			Expires: time.Now().Add(120 * time.Second),
		})

		log.Println("homePage / redirect to /choose")
		http.Redirect(res, req, "/choose", 302)

	}
}

func login(res http.ResponseWriter, req *http.Request) {

	formUsername := req.FormValue("username")
	formPassword := req.FormValue("password")

	log.Printf("Logging in with user: %s\n", formUsername)

	authenticated := LDAPAuthUser(formUsername, formPassword)

	if authenticated {

		// create session token for user
		sessionToken, err := uuid.NewV4()
		if err != nil {
			log.Printf("sessionToken failed to create: %s", err)
			return
		}

		// Finally, we set the client cookie for "session_token" as the session token we just generated
		// we also set an expiry time of 120 seconds
		http.SetCookie(res, &http.Cookie{
			Name:    "session_token",
			Value:   sessionToken.String(),
			Expires: time.Now().Add(120 * time.Second),
		})

		//log.Printf("login cookie set with token: %s\n", sessionToken.String())
		log.Println("/login redirect to /choose")
		http.Redirect(res, req, "/choose", 302)
	} else {
		log.Println("User did not authenticate.")
		http.Redirect(res, req, "/", 302)
	}

}

func chooseFile (res http.ResponseWriter, req *http.Request) {
	_, err := req.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			res.WriteHeader(http.StatusUnauthorized)
			return
		}
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	t, err := template.New("webpage").Parse(uploadTpl)
	if err != nil {
		log.Print(err)
		return
	}

	err = t.Execute(res, Config)
	if err != nil {
		log.Print("execute: ", err)
		return
	}
}

func uploadFile (res http.ResponseWriter, req *http.Request) {
	_, err := req.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			res.WriteHeader(http.StatusUnauthorized)
			return
		}
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Print("Handling Formfile")

	uploadedFile, handler, err := req.FormFile("fileupload")
	if err != nil {
	   log.Println(err)
	   return
	}
	defer uploadedFile.Close()

	filePath := fmt.Sprintf("%s/%s", Config.Upload.Path, handler.Filename)

	log.Printf("Saving file %s", filePath)

	saveFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
	   log.Println(err)
	   return
	}
	defer saveFile.Close()
	io.Copy(saveFile, uploadedFile)

	t, err := template.New("webpage").Parse(doneTpl)
	if err != nil {
		log.Print(err)
		return
	}

	err = t.Execute(res, Config)
	if err != nil {
		log.Print("execute: ", err)
		return
	}

}

func init() {

	ConfigFile = flag.String("conf", "", "Config file for this listener and ldap configs")

	flag.Parse()

	if _, err := toml.DecodeFile(*ConfigFile, &Config); err != nil {
		log.Fatal(err)
	}

}

func main() {

	// Setup the proper format for the listening port on any interface
	listenPort := fmt.Sprintf(":%d", Config.Listen.Port)

	http.HandleFunc("/", homePage)
	http.HandleFunc("/login", login)
	http.HandleFunc("/choose", chooseFile)
	http.HandleFunc("/upload", uploadFile)

	if Config.Listen.SSL == true {
		log.Println("Listening on port " + listenPort + " with SSL")
		err := http.ListenAndServeTLS(listenPort, Config.Listen.Cert, Config.Listen.Key, nil)
		if err != nil {
			log.Fatal("ListenAndServe: ", err)
		}
	} else {
		log.Println("Listening on port " + listenPort + " without SSL")
		err := http.ListenAndServe(listenPort, nil)
		if err != nil {
			log.Fatal("ListenAndServe: ", err)
		}
	}

}
