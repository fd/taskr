package taskr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/pborman/uuid"

	"github.com/fd/taskr/pkg/api"
)

type Client struct {
	baseURL string
}

func NewClient(addr string) *Client {
	return &Client{baseURL: "http://" + addr}
}

type TaskOption func(opts *taskOption)

type taskOption struct {
	name         string
	dependencies []string
	waitUntil    time.Time
}

func WaitUntil(t time.Time) TaskOption {
	return func(opts *taskOption) {
		opts.waitUntil = t
	}
}

func Delay(d time.Duration) TaskOption {
	return WaitUntil(time.Now().Add(d))
}

func Name(name string) TaskOption {
	return func(opts *taskOption) {
		opts.name = name
	}
}

func Dependencies(names ...string) TaskOption {
	return func(opts *taskOption) {
		opts.dependencies = append(opts.dependencies, names...)
		sort.Strings(opts.dependencies)

		s := opts.dependencies[:0]
		l := ""
		for _, dep := range opts.dependencies {
			if l != dep {
				l = dep
				s = append(s, dep)
			}
		}

		opts.dependencies = s
	}
}

func (c *Client) Schedule(queue string, payload interface{}, opts ...TaskOption) (string, error) {
	var o taskOption
	for _, opt := range opts {
		opt(&o)
	}
	if o.name == "" {
		o.name = uuid.New()
	}

	payloadData, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	token := uuid.New()

	body := api.CreateOptions{
		ID:           o.name,
		WorkerQueue:  queue,
		Dependencies: o.dependencies,
		Payload:      payloadData,
		Token:        token,
	}

	if !o.waitUntil.IsZero() {
		body.WaitUntil = o.waitUntil.Unix()
	}

	bodyData, err := json.Marshal(&body)
	if err != nil {
		return "", err
	}

	for i := 0; i < 3; i++ {
		err = c.post("/v1/tasks", bodyData)
		if err == os.ErrExist || err == nil {
			return o.name, err
		}
	}

	return "", err
}

func (c *Client) post(u string, data []byte) error {
	req, err := http.NewRequest("POST", c.baseURL+u, bytes.NewReader(data))
	if err != nil {
		return err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()
	defer io.Copy(ioutil.Discard, res.Body)

	if res.StatusCode == 409 {
		return os.ErrExist
	}
	if res.StatusCode == 200 {
		return nil
	}

	return fmt.Errorf("unexpected status code: %d", res.StatusCode)
}
