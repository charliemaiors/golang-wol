# Golang-Wol

Golang Wol a simple wake on lan service written in go using go html templates, it could be deployed on normal servers and also on low power devices such as a Raspberry pi.
It uses [Sabhiram](https://github.com/sabhiram/go-wol) library in order to send wake on lan (magic) packet, has a web frontend defined using go templates (just for fun) and uses [Tatsushid](https://github.com/tatsushid/go-fastping) library in order to check if target host is alive.

## Installation
---