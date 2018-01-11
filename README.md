# Golang-Wol
[![Build Status](https://travis-ci.org/charliemaiors/golang-wol.svg?branch=v1.0.1)](https://travis-ci.org/charliemaiors/golang-wol)

Golang Wol a simple wake on lan service written in go using go html templates, it could be deployed on normal servers and also on low power devices such as a Raspberry pi.
It uses [Sabhiram](https://github.com/sabhiram/go-wol) library in order to send wake on lan (magic) packet, has a web frontend defined using go templates (just for fun) and uses forked [Tatsushid](https://github.com/tatsushid/go-fastping) library in order to check if target host is alive.

## Installation
---
To install this service you could simple go get it using:

```bash
go get github.com/charliemaiors/golang-wol
```
Copy the executable in your home folder, create a folder called ```storage``` and run it using 

```
    ./golang-wol
```
The service will use port 5000 and is available on http plain.

## Advanced installation
---
 You could also run Golang Wol as a system service using systemd template in script configuration and/or define a custom configuration file, this file could be located in a folder called ```config``` at the same directory level of the executable or in ```/etc/wol/```.
 The Configuration file could have these sections:

 ```yaml
 server:
    tls: 
        cert: <certificate-path>
        key: <key-path>
 ```

 This section enable TLS insted of http plain, the port is always 5000.

 ```yaml
server:
    letsencrypt:
        host: <dns domain name>
        cert: <path-to-cert-folder-cache>
 ```

 This section is mutually exclusive with the previous one, it enables [letsencrypt](https://letsencrypt.org/) support for tls connection, but it requires the standard HTTPS port(443). With this configuration the executable must have the capabilities to take control of https port.
 
 ```yaml
 storage:
    path: <db-path>
 ```
 Define the custom location of the database, the executable has to have the capabilities of read,write or create file inside that folder.

## Service
---
This service could be defined as Linux systemd service or windows service using files under script directory. In order to install on linux, copy ```scripts/rwol.service``` in ``` /lib/systemd/system/``` folder and then run  ```#systemctl enable rwol; systemctl start rwol ```. Regarding Windows Service open an elevated powershell and run ```scripts/win-service-install.ps1```, it will check if [Chocolatey](https://chocolatey.org/) is installed and then install [nssm](https://nssm.cc/) in order to define rwol as service.

## Reverse Proxy
---

This service could be installed also behind a reverse proxy defining in the configuration file this section

```yaml
server:
    proxy: 
       enabled: true
       prefix: <reverse-proxy-prefix>
```

And configure your apache reverse proxy in this way

```
<IfModule mod_ssl.c>
        <VirtualHost *:443>
                # TLS/SSL configuration, this is for the use of crypto.
                ServerAdmin webmaster@localhost
                ServerName <your-server-name>
                SSLEngine On
                SSLCertificateFile /etc/openssl/cert.pem
                SSLCertificateKeyFile /etc/openssl/key.pem

                ProxyPass /prefix http://localhost:5000/prefix
                ProxyHTMLURLMap http://localhost:5000 /prefix
                <Location /wol/>
                        ProxyPassReverse  http://localhost:5000/prefix
                        SetOutputFilter proxy-html
                        ProxyHTMLURLMap /          /prefix/
                        ProxyHTMLURLMap /prefix      /prefix #avoid infinite loop
                </Location>
        </VirtualHost>
</IfModule>
```

Apache required modules are: ```mod_proxy```,  ```mod_proxy_html ``` and ```mod_ssl``` if the reverse proxy uses ssl.

Nginx must be configured in order to have a longer timeout period for reverse proxy adding the following snippet to nginx.conf file under http section

```
    proxy_connect_timeout       600;
    proxy_send_timeout          600;
    proxy_read_timeout          600;
    send_timeout                600;
```

After that you could add your reverse proxy configuration to your site, using this snippet

```
    location /prefix {
        proxy_pass http://localhost:5000/prefix;
        proxy_set_header X-Forwarded-Host $host;
        proxy_set_header X-Forwarded-Server $host;
        add_header X-Forwarded-Scheme https;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $remote_addr;
        proxy_set_header Host $host;
    }
```

 ## Docker
---
 This service is also available as container for arm, normal x86 and windows container on the [docker hub](https://hub.docker.com/r/cmaiorano/golang-wol/) using respectively arm, x86 or win as image tag.