package feign

import (
	"fmt"
	"github.com/gaojh/eureka-client/eureka"
	"github.com/go-resty/resty/v2"
	"log"
	"net/url"
	"strings"
)

type Client struct {
	eurekaClient *eureka.Client
	headers      map[string]string
	body         interface{}
	params       map[string]string
	result       interface{}
}

type Result struct {
	Resp *resty.Response
	Err  error
}

func NewClient(eurekaClient *eureka.Client) *Client {
	client := &Client{
		eurekaClient: eurekaClient,
		headers:      nil,
		body:         nil,
		params:       nil,
		result:       nil,
	}
	return client
}

func (c *Client) Header(k, v string) *Client {
	if c.headers == nil {
		c.headers = make(map[string]string)
	}
	c.headers[k] = v
	return c
}

func (c *Client) Body(body interface{}) *Client {
	c.body = body
	return c
}

func (c *Client) Params(p map[string]string) *Client {
	c.params = p
	return c
}

func (c *Client) SetResult(result interface{}) *Client {
	c.result = result
	return c
}

func (c *Client) Get(rawURL string) *Result {
	u, err := url.Parse(rawURL)
	if err != nil {
		return &Result{
			Resp: nil,
			Err:  err,
		}
	}

	request := c.app(strings.ToUpper(u.Host)).R()
	if c.body != nil {
		request.SetBody(c.body)
	}

	if c.result != nil {
		request.SetResult(c.result)
	}

	if c.params != nil {
		request.SetQueryParams(c.params)
	}

	if c.headers != nil {
		request.SetHeaders(c.headers)
	}

	resp, err := request.SetQueryString(u.RawQuery).Get(u.Path)
	return &Result{
		Resp: resp,
		Err:  err,
	}
}

func (c *Client) Post(rawURL string) *Result {
	u, err := url.Parse(rawURL)
	if err != nil {
		return &Result{
			Resp: nil,
			Err:  err,
		}
	}
	request := c.app(strings.ToUpper(u.Host)).R()
	if c.body != nil {
		request.SetBody(c.body)
	}

	if c.result != nil {
		request.SetResult(c.result)
	}

	if c.params != nil {
		request.SetQueryParams(c.params)
	}

	if c.headers != nil {
		request.SetHeaders(c.headers)
	}

	resp, err := request.Post(u.Path)
	return &Result{
		Resp: resp,
		Err:  err,
	}
}

func (c *Client) app(app string) *resty.Client {
	appUrls, ok := c.getAppUrls(app)
	if !ok {
		return nil
	}
	u, err := DoBalance("random", appUrls)
	if err != nil {
		log.Println(err)
		return nil
	}

	restyClient := resty.New()
	restyClient.HostURL = u
	log.Println(fmt.Sprintf("选择url：%v", u))
	return restyClient
}

func (c *Client) getApplication(app string) (eureka.Application, bool) {
	for _, application := range c.eurekaClient.Applications.Applications {
		if application.Name == app {
			return application, true
		}
	}

	return eureka.Application{}, false
}

func (c *Client) getAppUrls(app string) ([]string, bool) {
	application, ok := c.getApplication(app)
	if ok {
		var tmpAppUrls []string
		for _, instance := range application.Instances {
			tmpAppUrls = append(tmpAppUrls, strings.TrimRight(instance.HomePageURL, "/"))
		}
		return tmpAppUrls, true
	}
	log.Println(fmt.Sprintf("未获取到该应用的实例信息：%v", app))
	return nil, false
}
