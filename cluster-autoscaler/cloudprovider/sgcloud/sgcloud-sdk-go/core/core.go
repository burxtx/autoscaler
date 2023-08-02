package core

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/sgcloud/sgcloud-sdk-go/util"
)

type Config struct {
	Region     string `json:"region"`
	Endpoint   string
	APIVersion string
	Protocol   string
	UserAgent  string
	ProxyHost  string
	ProxyPort  int
	//ConnectionTimeoutInMillis time.Duration // default value: 10 * time.Second in http.DefaultTransport
	MaxConnections int           // default value: 2 in http.DefaultMaxIdleConnsPerHost
	Timeout        time.Duration // default value: 0 in http.Client
	RetryPolicy    RetryPolicy
	Checksum       bool
	Auth           Authenticator
	Debug          bool
}

// Region contains all regions of sgcloud.
var Region = map[string]string{
	"debug": "debug",
	"bqj":   "bqj",
}

func (config *Config) GetRegion() string {
	region := config.Region
	if region == "" {
		region = Region["debug"]
	}

	return region
}

type Client struct {
	*Config
	httpClient *http.Client
	debug      bool
}

// NewClient retunrs client
func NewClient(config *Config) *Client {
	return &Client{config, newHttpClient(config), config.Debug}
}

// SetDebug enables debug mode of bce.Client instance.
func (c *Client) SetDebug(debug bool) {
	c.debug = debug
}

func newHttpClient(config *Config) *http.Client {
	transport := new(http.Transport)

	if defaultTransport, ok := http.DefaultTransport.(*http.Transport); ok {
		transport.Proxy = defaultTransport.Proxy
		transport.Dial = defaultTransport.Dial
		transport.TLSHandshakeTimeout = defaultTransport.TLSHandshakeTimeout
	}
	if config.ProxyHost != "" {
		host := config.ProxyHost
		if config.ProxyPort > 0 {
			host += ":" + strconv.Itoa(config.ProxyPort)
		}
		proxyUrl, err := url.Parse(util.HostToURL(host, "http"))
		if err != nil {
			panic(err)
		}

		transport.Proxy = http.ProxyURL(proxyUrl)
	}

	/*
		if c.ConnectionTimeout > 0 {
			transport.TLSHandshakeTimeout = c.ConnectionTimeout
		}
	*/

	if config.MaxConnections > 0 {
		transport.MaxIdleConnsPerHost = config.MaxConnections
	}

	return &http.Client{
		Transport: transport,
		Timeout:   config.Timeout,
	}
}

// GetURL generates the full URL of http request for Baidu Cloud API.
func (c *Client) GetURL(host, uriPath string, params map[string]string) string {
	if strings.Index(uriPath, "/") == 0 {
		uriPath = uriPath[1:]
	}

	if c.APIVersion != "" {
		uriPath = fmt.Sprintf("%s/%s", c.APIVersion, uriPath)
	}

	return util.GetURL(c.Protocol, host, uriPath, params)
}

// RetryPolicy defined an interface for retrying of bce.Client.
type RetryPolicy interface {
	GetMaxErrorRetry() int      // GetMaxErrorRetry specifies the max retry count.
	GetMaxDelay() time.Duration // GetMaxDelay specifies the max delay time for retrying.

	// GetDelayBeforeNextRetry specifies the delay time for next retry.
	GetDelayBeforeNextRetry(err error, retriesAttempted int) time.Duration
}

// DefaultRetryPolicy is the default implemention of interface bce.RetryPolicy.
type DefaultRetryPolicy struct {
	MaxErrorRetry int
	MaxDelay      time.Duration
}

// NewDefaultRetryPolicy returns DefaultRetryPolicy
func NewDefaultRetryPolicy(maxErrorRetry int, maxDelay time.Duration) *DefaultRetryPolicy {
	return &DefaultRetryPolicy{maxErrorRetry, maxDelay}
}

// GetMaxErrorRetry specifies the max retry count.
func (policy *DefaultRetryPolicy) GetMaxErrorRetry() int {
	return policy.MaxErrorRetry
}

// GetMaxDelay specifies the max delay time for retrying.
func (policy *DefaultRetryPolicy) GetMaxDelay() time.Duration {
	return policy.MaxDelay
}

// GetDelayBeforeNextRetry specifies the delay time for next retry.
func (policy *DefaultRetryPolicy) GetDelayBeforeNextRetry(err error, retriesAttempted int) time.Duration {
	if !policy.shouldRetry(err, retriesAttempted) {
		return -1
	}
	duration := (1 << uint(retriesAttempted)) * 300 * time.Millisecond
	if duration > policy.GetMaxDelay() {
		return policy.GetMaxDelay()
	}
	return duration
}

func (policy *DefaultRetryPolicy) shouldRetry(err error, retriesAttempted int) bool {
	if retriesAttempted > policy.GetMaxErrorRetry() {
		return false
	}
	if bceError, ok := err.(*Error); ok {
		if bceError.StatusCode == http.StatusInternalServerError {
			log.Println("Retry for internal server error.")
			return true
		}
		if bceError.StatusCode == http.StatusServiceUnavailable {
			log.Println("Retry for service unavailable.")
			return true
		}
	} else {
		log.Printf("Retry for unknow error: %s", err.Error())
		return true
	}
	return false
}

// SendRequest sends a http request to the endpoint of Baidu Cloud API.
func (c *Client) SendRequest(req *Request) (response *Response, err error) {

	if c.RetryPolicy == nil {
		c.RetryPolicy = NewDefaultRetryPolicy(3, 20*time.Second)
	}
	var buf []byte
	if req.Body != nil {
		buf, _ = ioutil.ReadAll(req.Body)
	}

	for i := 0; ; i++ {
		response, err = nil, nil
		if c.debug {
			util.Debug("", fmt.Sprintf("Request: httpMethod = %s, requestUrl = %s, requestHeader = %v",
				req.Method, req.URL.String(), req.Header))
		}
		t0 := time.Now()
		req.Body = ioutil.NopCloser(bytes.NewBuffer(buf))
		resp, httpError := c.httpClient.Do(req.raw())
		t1 := time.Now()
		response = NewResponse(resp)
		if c.debug {
			util.Debug("", fmt.Sprintf("http request: %s  do use time: %v", req.URL.String(), t1.Sub(t0)))
			statusCode := -1
			resString := ""
			var resHead http.Header
			if resp != nil {
				statusCode = resp.StatusCode
				re, err := response.GetBodyContent()
				if err != nil {
					util.Debug("", fmt.Sprintf("getbodycontent error: %v", err))
				}
				resString = string(re)
				resHead = resp.Header
			}
			util.Debug("", fmt.Sprintf("Response: status code = %d, httpMethod = %s, requestUrl = %s",
				statusCode, req.Method, req.URL.String()))
			util.Debug("", fmt.Sprintf("Response Header:  = %v", resHead))
			util.Debug("", fmt.Sprintf("Response body:  = %s", resString))
		}

		if httpError != nil {
			duration := c.RetryPolicy.GetDelayBeforeNextRetry(httpError, i+1)
			if duration <= 0 {
				err = httpError
				return response, err
			}
			time.Sleep(duration)
			continue
		}
		if resp.StatusCode >= http.StatusBadRequest {
			err = buildError(response)
		}
		if err == nil {
			return response, err
		}

		duration := c.RetryPolicy.GetDelayBeforeNextRetry(err, i+1)
		if duration <= 0 {
			return response, err
		}

		time.Sleep(duration)
	}
}
