# proxiedhttp
A HAProxy's proxy protocol v2 implementation (client-side) for golang standard HTTP server.


### Usage

```go
import (
    "net/http"
    "github.com/dbyio/proxiedhttp"
)

// A slice of authorized IP addresses to read the proxy protocol header from
// nil means any (not recommanded)
proxySources := []net.IP{net.ParseIP("127.0.0.1")}

// connexion read timeout
readTimeout := 5 * time.Second

listen, _ := net.Listen("tcp", "127.0.0.1:8000")
pListen := &proxiedhttp.Listener{
	Listener:    listen,
	ReadTimeout: readTimeout,
	AuthSources: proxySources,      
}

http.Serve(pListen, http.DefaultServeMux)
```