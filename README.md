# proxiedhttp
A HAProxy's proxy protocol v2 implementation (client-side) for golang standard HTTP server.


### Usage

```go
import (
    "net/http"
    "github.com/dbyio/proxiedhttp"
)

func main() {

    listen, err := net.Listen("tcp", "127.0.0.1:8000")

    // A slice of authorized IP addresses to read the proxy protocol header from
    // nil means any (not recommanded)
    proxySources := []net.IP{net.ParseIP("127.0.0.1")}

	pListen := &proxiedhttp.Listener{
		Listener:    listen,
		ReadTimeout: 5 * time.Second,   // read time out
		AuthSources: proxySources,      
    }
    
    log.Fatal(http.Serve(pListen, http.DefaultServeMux))
}
```