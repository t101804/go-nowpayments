package core

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/matm/go-nowpayments/pkg/config"
	"github.com/rotisserie/eris"
)

type baseURL string

const (
	// ProductionBaseURL is the URL to the production service.
	ProductionBaseURL baseURL = "https://api.nowpayments.io/v1"
	// SandBoxBaseURL is the URL to the sandbox service.
	SandBoxBaseURL = "https://api-sandbox.nowpayments.io/v1"
)

// SendParams are parameters needed to build and send an HTTP request to the service.
type SendParams struct {
	Body      io.Reader
	Into      interface{}
	Path      string
	RouteName string
	Values    url.Values
	// JWT token obtained after authentication.
	Token string
}

type routeAttr struct {
	method string
	path   string
}

var routes map[string]routeAttr = map[string]routeAttr{
	"status":         {http.MethodGet, "/status"},
	"currencies":     {http.MethodGet, "/currencies"},
	"estimate":       {http.MethodGet, "/estimate"},
	"min-amount":     {http.MethodGet, "/min-amount"},
	"payment-status": {http.MethodGet, "/payment"},
	"auth":           {http.MethodPost, "/auth"},
}

var (
	defaultURL baseURL = SandBoxBaseURL
)

var debug = false

// WithDebug prints out debugging info about HTTP traffic.
func WithDebug(d bool) {
	debug = d
}

// UseBaseURL sets the base URL to use to connect to NOWPayment's API.
func UseBaseURL(b baseURL) {
	defaultURL = b
}

// BaseURL returns the base URL used to connect to NOWPayment's API.
func BaseURL() string {
	return string(defaultURL)
}

// HTTPSend sends to endpoint with an optional request body and get the HTTP
// response result in into.
func HTTPSend(p *SendParams) error {
	if p == nil {
		return eris.New("nil params")
	}
	client := &http.Client{}
	method, path := routes[p.RouteName].method, routes[p.RouteName].path
	if path == "" {
		return eris.New(fmt.Sprintf("empty path for endpoint %q", p.RouteName))
	}
	u := string(defaultURL) + path
	if p.Values != nil {
		u += "?" + p.Values.Encode()
	}
	req, err := http.NewRequest(method, u, p.Body)
	if err != nil {
		return eris.Wrap(err, p.RouteName)
	}
	// Extra headers.
	req.Header.Add("X-API-KEY", config.APIKey())
	if p.Body != nil {
		req.Header.Add("Content-Type", "application/json")
	}
	if p.Token != "" {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", p.Token))
	}

	if debug {
		fmt.Println(">>> DEBUG")
		fmt.Println(req.Method, req.URL.String())
		fmt.Println("<<< DEBUG")
	}
	res, err := client.Do(req)
	if err != nil {
		return eris.Wrap(err, p.RouteName)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		type errResp struct {
			StatusCode int    `json:"statusCode"`
			Code       string `json:"code"`
			Message    string `json:"message"`
		}
		z := &errResp{}
		d := json.NewDecoder(res.Body)
		err = d.Decode(&z)
		if err != nil {
			return eris.Wrap(err, p.RouteName)
		}
		return eris.New(fmt.Sprintf("code %d (%s): %s", z.StatusCode, z.Code, z.Message))
	}
	d := json.NewDecoder(res.Body)
	err = d.Decode(&p.Into)
	return eris.Wrap(err, p.RouteName)
}