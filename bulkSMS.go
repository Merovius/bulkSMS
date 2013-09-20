package bulkSMS

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type BulkSMS struct {
	username	 string
	password	 string
	sender		 string
	testing		 enumTest
}

type apiCall struct {
	url			string
	parameters	map[string][]string
}

type EnumRoutingGroup int
const (
	Default EnumRoutingGroup = iota
	Economy
	Standard
	Premium
)

type enumTest int
const (
	None enumTest = iota
	AlwaysSucceed
	AlwaysFail
)

type SMS struct {
	Message      string
	Recipients   []string
	RoutingGroup EnumRoutingGroup
	batchId		 int
	status		 int
	statusDescr	 string
}

type Error struct {
	code		int
	descr		string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s (%d)", e.descr, e.code)
}

func New(username string, password string, sender string) *BulkSMS {
	return &BulkSMS{username, password, sender, 0}
}

func (b *BulkSMS) apiCall(call string, parameters map[string][]string) (ret []string, err error) {
	var v url.Values;
	if parameters != nil {
		v = parameters
	} else {
		v = url.Values{}
	}
	v.Add("username", b.username)
	v.Add("password", b.password)
	if b.testing == AlwaysSucceed {
		v.Add("test_always_succeed", "1")
	} else if b.testing == AlwaysFail {
		v.Add("test_always_fail", "1")
	}

	resp, err := http.PostForm(call, v)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	ret = strings.Split(strings.TrimSpace(string(body)), "|")
	return
}

func (b *BulkSMS) GetCredits() (credits float64, err error) {
	ret, err := b.apiCall("http://bulksms.de:5567/eapi/user/get_credits/1/1.1", nil)
	if err != nil {
		return -1, err
	}
	code, err := strconv.Atoi(ret[0])
	if err != nil {
		return -1, err
	}
	if code == 0 {
		credits, err = strconv.ParseFloat(ret[1], 64)
		if err != nil {
			return -1, err
		}
		return
	} else {
		return -1, &Error{ code, ret[1] }
	}
	return
}

func NewSMS(message string, recipients []string) *SMS {
	return &SMS{ message, recipients, Default, 0, -1, "" }
}

func (s *SMS) Status() string {
	return fmt.Sprintf("%s (%d)", s.statusDescr, s.status)
}

func (b *BulkSMS) Send(m *SMS) (err error) {
	params := make(map[string][]string)
	params["message"] = []string{m.Message}
	params["msisdn"] = []string{strings.Join(m.Recipients, ",")}
	if m.RoutingGroup != Default {
		params["routing_group"] = []string{strconv.Itoa(int(m.RoutingGroup))}
	}
	ret, err := b.apiCall("http://bulksms.de:5567/eapi/submission/send_sms/2/2.0", params)
	if err != nil {
		return err
	}
	m.statusDescr = ret[1]
	m.status, err = strconv.Atoi(ret[0])
	if err != nil {
		return err
	}
	if m.status == 0 {
		return nil
	}
	if len(ret) > 2 && ret[2] != "" {
		m.batchId, err = strconv.Atoi(ret[2])
		if err != nil {
			return err
		}
	}
	return &Error{ m.status, ret[1] }
}
