package request

import (
	"crypto/tls"
	"fmt"
	"github.com/valyala/fasthttp"
	"strings"
	"time"
)

type FastHttpClient struct {
	*fasthttp.Client
}

func GetBody(response *fasthttp.Response) ([]byte, error) {
	contentEncoding := strings.ToLower(string(response.Header.Peek("Content-Encoding")))
	var body []byte
	var err error
	switch contentEncoding {
	case "", "none", "identity":
		body, err = response.Body(), nil
	case "gzip":
		body, err = response.BodyGunzip()
	case "deflate":
		body, err = response.BodyInflate()
	default:
		// TODO: support `br`
		body, err = []byte{}, fmt.Errorf("unsupported Content-Encoding: %v", contentEncoding)
	}
	return body, err
}

func NewFastHttpClient() Doer {
	return &FastHttpClient{
		Client: &fasthttp.Client{
			Name: scriptUserAgent,
			TLSConfig: &tls.Config{
				InsecureSkipVerify: true,
				ClientSessionCache: cache,
				Renegotiation:      tls.RenegotiateFreelyAsClient,
			},
			DialDualStack: false,
			//Dial:                      DialFunc,
			MaxConnsPerHost:           500,
			MaxIdleConnDuration:       3 * time.Second,
			MaxConnDuration:           30 * time.Second,
			MaxIdemponentCallAttempts: 5,
			ReadBufferSize:            8096,
			WriteBufferSize:           4096,
			ReadTimeout:               20 * time.Second,
			WriteTimeout:              20 * time.Second,
			MaxResponseBodySize:       1024 * 1024 * 5,
		},
	}
}
func (c *FastHttpClient) Do(conf *Request) (*Response, error) {
	if conf.Method == "" {
		conf.Method = "GET"
	}
	request := fasthttp.AcquireRequest()
	response := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(request)
	defer fasthttp.ReleaseResponse(response)
	request.SetRequestURI(conf.Url)
	request.SetConnectionClose()
	request.Header.SetMethod(conf.Method)
	if conf.Host != "" {
		request.SetHost(conf.Host)
	} else {
		request.SetHostBytes(request.URI().Host())
	}
	request.Header.SetUserAgent("Mozilla/5.0 (Windows NT 6.2; rv:30.0) Gecko/20150101 Firefox/32.0")
	request.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	if conf.Cookies != nil {
		for key, v := range conf.Cookies {
			request.Header.SetCookie(key, v.(string))
		}
	}
	Cps.Incr(1)
	var err error
	if conf.Redirect {
		err = c.Client.DoRedirects(request, response, 12)
	} else {
		err = c.Client.DoTimeout(request, response, time.Second*10)
	}
	//logrus.Debugf("HTTP request: %s", request.Header.String())
	if err != nil {
		Eps.Incr(1)
		return nil, err
	}
	body, err := GetBody(response)
	//logrus.Debugf("HTTP response: %s %s", request.Header.String(), response.Body())
	header := make(map[string][]string)
	response.Header.VisitAll(func(key, value []byte) {
		header[string(key)] = []string{string(value)}
	})
	wordsSize := len(strings.Split(string(body), " "))
	sentencesSize := len(strings.Split(string(body), "."))
	linesSize := len(strings.Split(string(body), "\n"))

	r := &Response{
		StatusCode:       response.StatusCode(),
		Headers:          header,
		Data:             body,
		ContentLength:    len(body),
		ContentWords:     wordsSize,
		ContentLines:     linesSize,
		ContentSentences: sentencesSize,
		Cancelled:        false,
		Request:          conf,
	}
	Gps.Incr(1)
	return r, nil
}
