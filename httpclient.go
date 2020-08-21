package request

import (
	"crypto/tls"
	"github.com/paulbellamy/ratecounter"
	"github.com/sirupsen/logrus"
	"net/url"
	"time"
)

var scriptUserAgent = "Mozilla/5.0 (Windows NT 6.2; rv:30.0) Gecko/20150101 Firefox/32.0"
var cache = tls.NewLRUClientSessionCache(2500)

var Cps = ratecounter.NewRateCounter(20 * time.Second)
var Eps = ratecounter.NewRateCounter(20 * time.Second)
var Gps = ratecounter.NewRateCounter(20 * time.Second)
var start = time.Now()

func Log(interval time.Duration) {
	//f := func () {
	defer recover()
	logrus.WithFields(logrus.Fields{"c/s": Cps.Rate() / int64(100/interval), "g/s": Gps.Rate() / int64(100/interval), "e/s": Eps.Rate() / int64(100/interval), "elapsed": time.Since(start)}).Warn()
	//}()
}

type Request struct {
	Url      string
	Host     string
	Method   string
	Data     string
	Cookies  map[string]interface{}
	Redirect bool
	Errors   int
	BaseData *SinglePageBaseData
}

func (r *Request) Log() *logrus.Entry {
	return logrus.WithField("url", r.Url)
}

func (r *Request) Clone() *Request {
	return &Request{
		Url:      r.Url,
		Host:     r.Host,
		Method:   r.Method,
		Data:     r.Data,
		Cookies:  r.Cookies,
		Redirect: r.Redirect,
	}
}

type SinglePageBaseData struct {
	BaseBody       []byte
	BaseBodyLength int
	BaseStatusCode int
}

//var REQUEST_NUMBER = 0

// Response struct holds the meaningful data returned from request and is meant for passing to filters
type Response struct {
	StatusCode       int
	Headers          map[string][]string
	Data             []byte
	ContentLength    int
	ContentWords     int
	ContentSentences int
	ContentLines     int
	Cancelled        bool
	Request          *Request
}

// GetRedirectLocation returns the redirect location for a 3xx redirect HTTP response
func (resp *Response) GetRedirectLocation(absolute bool) string {
	redirectLocation := ""
	if resp.StatusCode >= 300 && resp.StatusCode <= 399 {
		if loc, ok := resp.Headers["Location"]; ok {
			if len(loc) > 0 {
				redirectLocation = loc[0]
			}
		}
	}

	if absolute {
		redirectUrl, err := url.Parse(redirectLocation)
		if err != nil {
			return redirectLocation
		}
		baseUrl, err := url.Parse(resp.Request.Url)
		if err != nil {
			return redirectLocation
		}
		redirectLocation = baseUrl.ResolveReference(redirectUrl).String()
	}

	return redirectLocation
}

type Doer interface {
	Do(request *Request) (*Response, error)
}

func NewClient(t ClientType) Doer {
	switch t {
	case FastHttp:
		return NewFastHttpClient()
	case Retryable:
		return NewRetryableHttpClient()
	default:
		return nil
	}
}

type ClientType int

const (
	FastHttp ClientType = iota
	Retryable
)
