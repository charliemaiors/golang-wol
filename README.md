# Golang-Wol

Golang Wol a simple wake on lan service written in go using go html templates, it could be deployed on normal servers and also on low power devices such as a Raspberry pi.
It uses [Sabhiram](https://github.com/sabhiram/go-wol) library in order to send wake on lan (magic) packet, has a web frontend defined using go templates (just for fun) and uses [Tatsushid](https://github.com/tatsushid/go-fastping) library in order to check if target host is alive.

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

## Reverse Proxy
---

This service could be installed also behind a reverse proxy defining in the configuration file this section

```yaml
server:
    proxy: true
```

And configure your apache reverse proxy in this way (nginx configuration will be defined as soon as I've spare time)

```
<IfModule mod_ssl.c>
        <VirtualHost *:443>
                # TLS/SSL configuration, this is for the use of crypto.
                ServerAdmin webmaster@localhost
                ServerName <your-server-name>
                SSLEngine On
                SSLCertificateFile /etc/openssl/cert.pem
                SSLCertificateKeyFile /etc/openssl/key.pem

                ProxyPass /prefix http://localhost:5000
                ProxyHTMLURLMap http://localhost:5000 /prefix
                <Location /wol/>
                        ProxyPassReverse  http://localhost:5000
                        SetOutputFilter proxy-html
                        ProxyHTMLURLMap /          /prefix/
                        ProxyHTMLURLMap /prefix      /prefix #avoid infinite loop
                </Location>
        </VirtualHost>
</IfModule>
```

Apache required modules are: ```mod_proxy```,  ```mod_proxy_html ``` and ```mod_ssl``` if the reverse proxy uses ssl.

 ## Docker
---
 This service is also available as container for arm, normal x86 and windows container on the [docker hub](https://hub.docker.com/r/cmaiorano/golang-wol/) using respectively arm, x86 or win as image tag.