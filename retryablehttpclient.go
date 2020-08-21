package request

import (
	"bufio"
	"context"
	"fmt"
	"github.com/projectdiscovery/retryablehttp-go"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"
)

func (c *RetryableHttpClient) Do(conf *Request) (*Response, error) {
	if conf.Method == "" {
		conf.Method = "GET"
	}

	r, err := http.NewRequest(conf.Method, conf.Url, nil)
	if err != nil {
		Eps.Incr(1)
		return nil, err
	}

	req, err := retryablehttp.FromRequest(r)
	req.Host = conf.Host
	if err != nil {
		Eps.Incr(1)
		return nil, err
	}

	req.Header.Set("Content-Type", "Mozilla/5.0 (Windows NT 6.2; rv:30.0) Gecko/20150101 Firefox/32.0")
	if conf.Cookies != nil {
		for key, v := range conf.Cookies {
			req.Header.Add("Cookie", fmt.Sprintf("%s=%s;", key, v.(string)))
		}
	}
	Cps.Incr(1)
	res, err := c.Client.Do(req)
	if err != nil {
		conf.Errors += 1
		Eps.Incr(1)
		return nil, err
	}

	var a = &io.LimitedReader{R: bufio.NewReader(res.Body), N: 5 * 1024 * 1024}
	body, _ := ioutil.ReadAll(a)
	bodyStr := string(body)
	res.Body.Close()
	conf.Errors = 0
	wordsSize := len(strings.Split(bodyStr, " "))
	linesSize := len(strings.Split(bodyStr, "\n"))
	sentencesSize := len(strings.Split(bodyStr, "."))

	resp := &Response{
		StatusCode:       res.StatusCode,
		Headers:          res.Header,
		Data:             body,
		ContentLength:    len(body),
		ContentWords:     wordsSize,
		ContentLines:     linesSize,
		ContentSentences: sentencesSize,
		Request:          conf,
	}
	Gps.Incr(1)
	return resp, nil
}

type RetryableHttpClient struct {
	*retryablehttp.Client
}

func NewRetryableHttpClient() Doer {
	c := &RetryableHttpClient{
		Client: retryablehttp.NewWithHTTPClient(
			&http.Client{
				Transport: retryablehttp.DefaultHostSprayingTransport(),
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse
				},
			},
			retryablehttp.Options{
				RetryWaitMin: 1 * time.Second,
				RetryWaitMax: 5 * time.Second,
				RetryMax:     3,
				Timeout:      time.Second * 10,
			}),
	}
	c.Client.HTTPClient.Transport.(*http.Transport).DialContext = func(ctx context.Context, netw, addr string) (net.Conn, error) {
		return DialFunc(addr)
	}
	return c
}
