# File Uploader

## Depends

"github.com/BurntSushi/toml"
"github.com/satori/go.uuid"
"gopkg.in/ldap.v2"

## Requirements

You must fill in the necessary items in the config.toml file.

### Building

Use the appropriate build_ARCH.sh script

### Server

* The listen port for the file upload web server
* If you want to enable SSL or not
* The path to the necessary SSL certs if you want SSL

## Why

I wanted to write a simple program that could be use to upload files to a web server.  The idea is that you could run an Nginx server and the URL to the file could be used to send to a remote Chrome client using the Admin Chrome tool.

## Launching

```shell
$ ./uploadvideo_linux -help
Usage of ./uploadvideo_linux:
  -conf string
        Config file for this listener and ldap configs
```

## Usage

Open your browser to the listen port and upload your file.

## Config TOML Format

### Listen Section

This section is used to configure the port you want the web interface for this tool to listen.  It is also where you configure SSL support.

```shell
[listen]
ssl = false
cert = "wildcard_certificate"
key = "wildcard_key"
port = 8081
```

### LDAP Section

Here you configure the following:

* `useldap` is a boolean which will either enable or disable authentication
* LDAP Server and port: `host` and `port`
* `base` is the LDAP DN where user accounts will be searched for.
* `groupbase` is the LDAP group DN where the search query will look for any group that a user is a member of
* `binddn` and `bindpassword` is the bind user that will authenticate to LDAP to perform the user search
* `groupname` the group a user must be a member of to be able to upload files

If authentitcation is disabled then all TVs listed in the configuration toml will be in the selection dropdown.

The code is written to attempt to use StartTLS for the LDAP connection.

The `memberOf` attribute is used to determine group membership.  The group you must be a member of is what you choose as the value for the `groupname` key.

```shell
[ldap]
useldap = true
host = "ldap.example.com"
port = 389
base = "cn=users,cn=accounts,dc=example,dc=com"
groupbase = "cn=groups,cn=accounts,dc=example,dc=com"
binddn = "cn=users,cn=accounts,dc=example,dc=com"
bindpassword = "some password"
groupname = "uploadfiles"
```

Effectively, this is the ldapsearch query happening.

```shell
ldapsearch -x -D "BIND_DN_USER" -W -b "cn=users,cn=accounts,dc=example,dc=com" "(&(uid=USER)(memberOf=uploadfiles,cn=groups,cn=accounts,dc=example,dc=com))" dn
```

The group membership would look like the following

```shell
memberOf: cn=uploadfiles,cn=groups,cn=accounts,dc=example,dc=com
```

### Upload Section

* The path to save the uploaded files
