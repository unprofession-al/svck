package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type checks []*check

func NewChecks(conf map[string]service, fakeHost, fakeProto, userAgent string, timeout int) (checks, error) {
	items := []*check{}

	for serviceName, service := range conf {
		for _, address := range service.Addresses {
			for testName, test := range service.Tests {
				for resourceName, resource := range test.Resources {
					c := check{
						service:         serviceName,
						test:            testName,
						resource:        resourceName,
						address:         address,
						contains:        resource.Contains,
						status:          test.Status,
						expectedHeaders: test.ExpectedHeaders,
						timeout:         timeout,
					}

					useAddress := address
					if fakeHost != "" {
						useAddress = fakeHost
					}
					resourceURL := strings.TrimLeft(resource.URL, "/")
					url := fmt.Sprintf("://%s/%s", useAddress, resourceURL)

					if fakeProto != "" {
						url = fmt.Sprintf("%s%s", fakeProto, url)
					} else if test.SSL {
						url = fmt.Sprintf("https%s", url)
					} else {
						url = fmt.Sprintf("http%s", url)
					}

					req, err := http.NewRequest("GET", url, nil)
					if err != nil {
						return checks{}, err
					}

					if resource.ContentType != "" {
						req.Header.Set("Content-Type", resource.ContentType)
					}

					if fakeHost != "" {
						req.Host = address
					}

					if test.SSL {
						req.Header.Set("X-Forwarded-Proto", "https")
					} else {
						req.Header.Set("X-Forwarded-Proto", "http")
					}

					for headerName, headerContent := range test.RequestHeaders {
						req.Header.Set(headerName, headerContent)
					}

					req.Header.Set("User-Agent", userAgent)

					c.request = req

					items = append(items, &c)
				}
			}
		}
	}

	return items, nil
}

type check struct {
	// identifiers
	service  string
	test     string
	resource string
	address  string

	// request
	request *http.Request
	timeout int

	// validation
	contains        []string
	status          int
	expectedHeaders map[string][]string

	// result
	response *http.Response
	duration float64
	success  bool
	reason   []string
}

func (c *check) MarshalJSON() ([]byte, error) {
	type Request struct {
		URL    string      `json:"url"`
		Header http.Header `json:"header"`
	}

	req := &Request{
		URL:    c.request.URL.String(),
		Header: c.request.Header,
	}

	type Response struct {
		StatusCode    int         `json:"status_code"`
		Proto         string      `json:"proto"`
		Header        http.Header `json:"header"`
		ContentLength int64       `json:"content_length"`
	}

	resp := &Response{
		StatusCode:    c.response.StatusCode,
		Proto:         c.response.Proto,
		Header:        c.response.Header,
		ContentLength: c.response.ContentLength,
	}

	return json.Marshal(&struct {
		// additional
		Name string `json:"name"`

		// identifiers
		Service  string `json:"service"`
		Test     string `json:"test"`
		Resource string `json:"resource"`
		Address  string `json:"address"`

		// request
		Request *Request `json:"request"`
		Timeout int      `json:"timeout"`

		// validation
		Contains        []string            `json:"contains"`
		Status          int                 `json:"status"`
		ExpectedHeaders map[string][]string `json:"expected_headers"`

		// result
		Response *Response `json:"response"`
		Duration float64   `json:"duration"`
		Success  bool      `json:"success"`
		Reason   []string  `json:"reason"`
	}{
		Name:            c.name(),
		Service:         c.service,
		Test:            c.test,
		Resource:        c.resource,
		Address:         c.address,
		Request:         req,
		Timeout:         c.timeout,
		Contains:        c.contains,
		ExpectedHeaders: c.expectedHeaders,
		Status:          c.status,
		Response:        resp,
		Duration:        c.duration,
		Success:         c.success,
		Reason:          c.reason,
	})
}

func (c *check) asCurl() string {
	return fmt.Sprintf("curl -v %s '%s'", c.requestHeaders("-H "), c.request.URL.String())
}

func (c *check) name() string {
	return fmt.Sprintf("%s@%s/%s/%s", c.service, c.address, c.test, c.resource)
}

func (c *check) requestHeaders(prefix string) string {
	headers := ""
	for k, v := range c.request.Header {
		for _, value := range v {
			headers = fmt.Sprintf("%s%s\"%s: %s\" ", headers, prefix, k, value)
		}
	}
	headers = fmt.Sprintf("%s%s\"Host: %s\"", headers, prefix, c.request.Host)

	return headers
}

func (c *check) responseHeaders(prefix string) string {
	headers := ""

	if c.response == nil {
		return headers
	}

	for k, v := range c.response.Header {
		for _, value := range v {
			headers = fmt.Sprintf("%s%s\"%s: %s\" ", headers, prefix, k, value)
		}
	}

	return headers
}

func (c *check) run() {
	var err error
	c.reason = []string{}
	duration := 0.0

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},

		Timeout: time.Duration(c.timeout) * time.Second,
	}
	start := time.Now()

	c.response, err = client.Do(c.request)
	if err != nil {
		c.reason = append(c.reason, fmt.Sprintf("Error requesting %s: %s", c.request.URL, err.Error()))
		c.duration = duration
		c.success = false
		return
	}

	c.duration = time.Since(start).Seconds()

	body := []byte{}
	if len(c.contains) > 0 {
		body, err = ioutil.ReadAll(c.response.Body)
		if err != nil {
			c.reason = append(c.reason, fmt.Sprintf("Error reading body of %s: %s", c.request.URL, err.Error()))
			c.success = false
			return
		}
	}

	if c.status == c.response.StatusCode {
		c.success = true
	} else {
		c.success = false
		c.reason = append(c.reason, fmt.Sprintf("Expected %d, received %d", c.status, c.response.StatusCode))
	}

	for _, contains := range c.contains {
		found, err := regexp.MatchString(contains, string(body))
		if err != nil {
			c.reason = append(c.reason, fmt.Sprintf("Error parsing regexp %s: %s", contains, err.Error()))
			c.success = false
			return
		}
		if !found {
			c.success = false
			c.reason = append(c.reason, fmt.Sprintf("Content regexp '%s' not in body", contains))
		}
	}

	for name, values := range c.expectedHeaders {
		recievedValues, ok := c.response.Header[name]
		if !ok {
			c.success = false
			c.reason = append(c.reason, fmt.Sprintf("No header '%s' received", name))
			continue
		}
		for _, v := range values {
			found := false
			for _, rv := range recievedValues {
				found, err = regexp.MatchString(v, rv)
				if err != nil {
					c.reason = append(c.reason, fmt.Sprintf("Error parsing regexp %s: %s", rv, err.Error()))
					c.success = false
					return
				}
				if found {
					continue
				}
			}
			if !found {
				c.success = false
				c.reason = append(c.reason, fmt.Sprintf("No value '%s' found for header '%s'", v, name))
			}
		}
	}
}
