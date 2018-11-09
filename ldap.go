package main

import (
	"crypto/tls"
	"fmt"
	"log"

	ldap "gopkg.in/ldap.v2"
)

// LDAPAuthUser - authenticate the user to LDAP
// return true and a slice of TVs the user is allowed to access
func LDAPAuthUser(username string, password string) (bool) {

	log.Print("Connecting to LDAP server")

	// Connect to the LDAP server
	l, err := ldap.Dial("tcp", fmt.Sprintf("%s:%d", Config.LDAP.Host, Config.LDAP.Port))
	if err != nil {
		log.Print(err)
		return false
	}
	defer l.Close()

	log.Print("Executing StartTLS")

	// Reconnect with TLS
	err = l.StartTLS(&tls.Config{InsecureSkipVerify: true})
	if err != nil {
		log.Print(err)
		return false
	}

	log.Print("Bind as readonly user")

	// Connect with the user to verify the account is valid
	err = l.Bind(Config.LDAP.BindDN, Config.LDAP.BindPassword)
	if err != nil {
		log.Print(err)
		log.Print(Config.LDAP.BindDN)
		return false
	}

	log.Printf("Creating search query for %s", username)

	// Search for the given username
	searchRequest := ldap.NewSearchRequest(
		Config.LDAP.Base,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 10, 0, false,
		fmt.Sprintf("(&(uid=%s)(memberOf=%s,%s))", username, Config.LDAP.GroupName, Config.LDAP.GroupBase),
		[]string{"dn"},
		nil,
	)

	log.Printf("Executing search query for %s", username)

	// Search for the user and being a member of the TV
	sr, err := l.Search(searchRequest)
	if err != nil {
		log.Print("Search failure: ", err)
		return false
	}

	log.Printf("Checking if %s has any entries", username)

	if len(sr.Entries) != 1 {
		log.Print("User does not exist or too many entries returned")
		return false
	}

	userdn := sr.Entries[0].DN

	log.Printf("Logging into LDAP as %s", username)

	// Bind as the user to verify their password
	err = l.Bind(userdn, password)
	if err != nil {
		log.Print(err)
		return false
	}

	return true
}
