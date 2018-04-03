# BigDisk Admin Control Panel
### Andrew Lee
### 4/2/18

BigDisk Admin Control Panel is an application to easily provision static asset storage for large applications. Using the Admin Panel, admins can allocate a set amount of storage (in gibabytes), add administrators to the application and generate a secret token (used as part of the upload and delete endpoints only accessible by the app administrators).

It uses [redis](https://redis.io/) for persistant storage while utilizing uses bcrypt to hash admin user passwords and using encrypted cookies to save application state. The panel also has an IP ban functionality which bans an IP after 5 failed attempts to log in.

## Installation
1. Install docker + docker-compose (with a version new enough to support multi-stage builds)
2. `git clone https://github.com/gilgameshskytrooper/bigdisk.git`
3. `cd bigdisk`
4. `cp docker-compose.yaml docker-compose-actual.yaml`
5. You will need to replace the environment variables for the service bigdisk on `docker-compose-actual.yaml`. `BIGDISKSUPERADMINEMAIL` should be an email you can access. `BIGDISKURL` will be the URL that will be defined in the `Caddyfile` (in my case, https://bigdisk.gilgameshskytrooper.io). Replace `BIGDISKEMAILUSERNAME` and `BIGDISKEMAILPASSWORD` with the login credential for a Gmail account with secure apps turned off (if you want to verify where this is used, look at `email/email.go`). `BIGDISKSUPERADMINPASSWORD` will be any password you would like to use for the `admin` superadmin account on the website.
6. In development, use the `./build` command which will create the necessary folders, and launch docker-compose up [not in detached mode so as to give you the ability to log the calls]. I am using this method instead of normal docker-compose up so that I can add `docker-compose-actual.yaml` in my .gitignore so that private login information is not exposed publically. In production, you will edit `docker-compose.yaml`  instead of `docker-compose-actual.yaml`, and use `docker-compose up -d` to run the application headless (You will need to make sure the `data` and `files` folders exist in the same directory as `docker-compose.yaml` to ensure data persistence is preseved even if the containers are restarted).

**NOTE**
> If you want to serve on a different port, just modify `service.bigdisk.ports*` in `docker-compose-actual.yaml` or `docker-compose.yaml` depending on development or production.
> Also, this application does not implement TLS at all. In order to fully secure the application, TLS is recommended. I will detail a method using [Caddy](https://caddyserver.com/) below.

## TLS Encryption
[Caddy](https://caddyserver.com/) is a wonderful web server brought to us by [mholt](https://github.com/mholt). It is written entirely in Go, and comes with some nifty features to get a fully encrypted website up and running in no time.

Since Caddy uses Let's Encrypt by default, there are 2 necessary components to run the following configuration:
1. You need to run Caddy (and by extension, the BigDisk application) on a machine with a publically accessible IP address.
2. You will need to make sure that you are using one of the TLS providers which have a CoreDNS plugin. (the list of valid DNS providers can be found [here](https://github.com/caddyserver/dnsproviders))
**If #2 isn't satisfied, you can still use Caddy. But you will have to find a different method to get/auto-renew certs using the ACME challenges as detailed in Let's Encrypts documentation. Once you figure out how to do this, you just point the location of those certs in your Caddyfile**

### Getting Caddy

#### Install Go
(Assuming you are using Ubuntu)
```
sudo add-apt-repository ppa:longsleep/golang-backports
sudo apt-get update
sudo apt-get install golang-go
```

Add the following in `~/.profile`
```
export GOPATH=/home/[username]/go
export PATH=$PATH:$(go env GOPATH)/bin
```

Source `~/.profile`
```
source ~/.profile
```

#### Install Caddy

```
go get github.com/mholt/caddy
cd ~/go/src/github.com/mholt/caddy/caddy
```

Add the following lines to line 38 of the file `caddymain/run.go`
```
	_ "blitznote.com/src/caddy.upload"
	_ "github.com/BTBurke/caddy-jwt"
	_ "github.com/SchumacherFM/mailout"
	_ "github.com/Xumeiquer/nobots"
	_ "github.com/abiosoft/caddy-git"
	_ "github.com/caddyserver/dnsproviders/namecheap"
	_ "github.com/caddyserver/forwardproxy"
	_ "github.com/captncraig/caddy-realip"
	_ "github.com/captncraig/cors"
	_ "github.com/casbin/caddy-authz"
	_ "github.com/echocat/caddy-filter"
	_ "github.com/filebrowser/filebrowser/caddy/filemanager"
	_ "github.com/freman/caddy-reauth"
	_ "github.com/hacdias/caddy-minify"
	_ "github.com/hacdias/caddy-webdav"
	_ "github.com/jung-kurt/caddy-cgi"
	_ "github.com/linkonoid/caddy-dyndns"
	_ "github.com/mastercactapus/caddy-proxyprotocol"
	_ "github.com/miekg/caddy-prometheus"
	_ "github.com/nicolasazrak/caddy-cache"
	_ "github.com/pieterlouw/caddy-net/caddynet"
	_ "github.com/pyed/ipfilter"
	_ "github.com/restic/caddy"
	_ "github.com/tarent/loginsrv/caddy"
	_ "github.com/xuqingfeng/caddy-rate-limit"
	_ "github.com/zikes/gopkg"
```

Compile and get executable

```
go run build.go
```

Move executable to somewhere on your system path (~/go/bin/ is a good choice)
```
mv caddy ~/go/bin
```

Make caddy directory

```
sudo mkdir /etc/caddy
```

Create `etc/caddy/Caddyfile` and  add the following contents replacing `[website.com]` with the domain from one of the supported TLS plugins and `[plugin name]` with the DNS provider ([https://github.com/caddyserver/dnsproviders](https://github.com/caddyserver/dnsproviders)). Finally, add your email at [username@email.com]:

```
[website.com] {
  errors stdout
  log stdout
  gzip
  proxy / localhost:8080 {
    transparent
  }

  limits {
    body /upload 10gb
  }

  timeouts {
    read none
    write none
    header none
    idle none
  }

  tls [username@email.com] {
    dns [plugin name]
  }
}
```

To run Caddy, do
```
caddy -conf /etc/caddy/Caddyfile
```

If you need to run Caddy automatically every time your system reboots, you can create a systemd script (if you are on Ubuntu) as follows:

Create the file `/etc/systemd/system/caddy.service`, and insert the following lines:

```
[Unit]
Description=Caddy HTTP/2 web server
Documentation=https://caddyserver.com/docs
After=network-online.target
Wants=network-online.target systemd-networkd-wait-online.service

[Service]
Restart=on-abnormal

; User and group the process will run as.
User=root
Group=root

; Letsencrypt-issued certificates will be written to this directory.
Environment=NAMECHEAP_API_USER=ajlee1000
Environment=NAMECHEAP_API_KEY=48e19c850e62496bb77bb011a42755b3
; Environment=CADDYPATH=/etc/ssl/caddy
Environment=CADDYPATH=/root/.caddy

; Always set "-root" to something safe in case it gets forgotten in the Caddyfile.
ExecStart=/usr/local/bin/caddy -log stdout -agree=true -conf=/etc/caddy/Caddyfile -root=/var/tmp
ExecReload=/bin/kill -USR1 $MAINPID

; Use graceful shutdown with a reasonable timeout
KillMode=mixed
KillSignal=SIGQUIT
TimeoutStopSec=5s

; Limit the number of file descriptors; see `man systemd.exec` for more limit settings.
LimitNOFILE=1048576
; Unmodified caddy is not expected to use more than that.
LimitNPROC=512

; Use private /tmp and /var/tmp, which are discarded after caddy stops.
PrivateTmp=true
; Use a minimal /dev (May bring additional security if switched to 'true', but it may not work on Raspberry Pi's or other devices, so it has been disabled in this dist.)
PrivateDevices=false
; Hide /home, /root, and /run/user. Nobody will steal your SSH-keys.
ProtectHome=false
; Make /usr, /boot, /etc and possibly some more folders read-only.
ProtectSystem=full
; … except /etc/ssl/caddy, because we want Letsencrypt-certificates there.
;   This merely retains r/w access rights, it does not add any new. Must still be writable on the host!
ReadWriteDirectories=/etc/ssl/caddy

; The following additional security directives only work with systemd v229 or later.
; They further restrict privileges that can be gained by caddy. Uncomment if you like.
; Note that you may have to add capabilities required by any plugins in use.
;CapabilityBoundingSet=CAP_NET_BIND_SERVICE
;AmbientCapabilities=CAP_NET_BIND_SERVICE
;NoNewPrivileges=true

[Install]
WantedBy=multi-user.target
```

To use the above systemd script, do `systemctl enable caddy && systemctl start caddy`

### Wrap Up
Using a reverse proxy like Caddy, you will only need to expose port 80 and 443 on the system itself as security does is implemented on the app level and not the system firewall.

Hence, you will need to enable both ports as follows (in Ubuntu):
```
sudo ufw enable to any port 80, 443 proto tcp
```
