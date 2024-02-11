package main

import "fmt"

var proxies = []string{
	"<user>:<password>@<host>:<port>",
}

type Proxy struct {
	User     string
	Password string
	Host     string
	Port     int
}

func (p *Proxy) String() string {
	return fmt.Sprintf("%s:%s@%s:%d", p.User, p.Password, p.Host, p.Port)
}

type Proxies struct {
	Items []Proxy
}
