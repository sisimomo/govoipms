package v1

import (
	"net/http"
	"log"
	"encoding/json"
	"errors"
	"mime/multipart"
	"reflect"
	"strings"
	"net/http/httputil"
	"bytes"
	"net/url"
	"io/ioutil"
)

type VOIPClient struct {
	URL      string
	Username string
	Password string
	Debug    bool
}

type StatusResp interface {
	GetStatus() string
}

type BaseResp struct {
	Status string `json:"status"`
}

func (b *BaseResp) GetStatus() string {
	return b.Status
}

type StringValueDescription struct {
	Value       string `json:"value"`
	Description string `json:"description"`
}

type NumberValueDescription struct {
	Value       json.Number `json:"value"`
	Description string `json:"description"`
}

func NewVOIPClient(url, username, password string, debug bool) *VOIPClient {
	return &VOIPClient{url, username, password, debug}
}

func (c *VOIPClient) Call(req *http.Request, respStruct interface{}) (*http.Response, error) {

	if c.Debug {
		out, _ := httputil.DumpRequest(req, true)
		log.Println(string(out))
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return resp, err
	}
	defer resp.Body.Close()

	body := resp.Body
	if c.Debug {
		log.Println("Response: ", resp)

		b, err := ioutil.ReadAll(body)
		if err != nil {
			return nil, err
		}

		log.Println(string(b))

		body = ioutil.NopCloser(bytes.NewReader(b))
	}

	decoder := json.NewDecoder(body)
	if err := decoder.Decode(respStruct); err != nil {
		return nil, err
	}

	return resp, err
}

func (c *VOIPClient) Get(method string, values url.Values, entity interface{}) error {

	u, err := url.Parse(c.URL)
	if err != nil {
		return err
	}

	values.Add("api_username", c.Username)
	values.Add("api_password", c.Password)
	values.Add("method", method)

	u.RawQuery = values.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		panic(err)
	}

	resp, err := c.Call(req, entity)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return errors.New(resp.Status)
	}

	s, ok := entity.(StatusResp)
	if ok && s.GetStatus() != "success" {
		return errors.New(s.GetStatus())
	}

	return nil
}

func (c *VOIPClient) Post(method string, entity interface{}, respStruct interface{}) error {

	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)

	bodyWriter.WriteField("api_username", c.Username)
	bodyWriter.WriteField("api_password", c.Password)
	bodyWriter.WriteField("method", method)

	if err := WriteStruct(bodyWriter, entity); err != nil {
		return err
	}

	contentType := bodyWriter.FormDataContentType()
	bodyWriter.Close()

	req, err := http.NewRequest("POST", c.URL, bodyBuf)
	req.Header.Set("Content-Type", contentType)
	resp, err := c.Call(req, respStruct)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return errors.New(resp.Status)
	}

	s, ok := respStruct.(StatusResp)
	if ok && s.GetStatus() != "success" {
		return errors.New(s.GetStatus())
	}

	return nil
}

func (c *VOIPClient) NewGeneralAPI() *GeneralAPI {
	return &GeneralAPI{c}
}

func (c *VOIPClient) NewAccountsAPI() *AccountsAPI {
	return &AccountsAPI{c}
}

func (c *VOIPClient) NewCDRAPI() *CDRAPI {
	return &CDRAPI{c}
}

func (c *VOIPClient) NewClientAPI() *ClientAPI {
	return &ClientAPI{c}
}

func WriteStruct(writer *multipart.Writer, i interface{}) error {
	val := reflect.Indirect(reflect.ValueOf(i))

	for i := 0; i < val.NumField(); i++ {

		structField := val.Type().Field(i)

		name := strings.TrimSuffix(structField.Tag.Get("json"), ",omitempty") //TODO:Stan the omitempty is rather fragile.
		if name == "" {
			name = strings.ToLower(structField.Name)
		}

		value := val.Field(i).Interface().(string)

		if err := writer.WriteField(name, value); err != nil {
			return err
		}
	}

	return nil
}

