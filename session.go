package ming800

import (
	"fmt"
	"html"
	"io/ioutil"
	//"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	//"path"
	"regexp"
	"strings"

	"github.com/northbright/htmlhelper"
)

type Session struct {
	ServerURL string
	Company   string
	User      string
	Password  string
	baseURL   *url.URL
	jar       *cookiejar.Jar
	client    *http.Client
	LoggedIn  bool
	// uls contains *url.URL of actions.
	urls map[string]*url.URL
}

type ClassEvent struct {
	ClassInstanceId string
	ClassId         string
	ClassName       string
	Status          string
	BeginTime       string
	EndTime         string
}

type Student struct {
	Name          string
	SID           string
	Status        string
	Comments      string
	PhoneNumber   string
	ReceiptNumber string
	ClassEvents   []ClassEvent
}

var (
	// rawurls contains actions' raw URLs.
	rawurls = map[string]string{
		"login":                "/j_spring_security_check",
		"loginRedirect":        "/standard/mainController.controller",
		"logout":               "/j_spring_security_logout",
		"mainControlloer":      "/standard/mainController.controller",
		"studentSearch":        "/edu/student/search.action",
		"viewStudent":          "/edu/student/basicinfo/viewstudent.action",
		"listCategoryAndClass": "/edu/base/clazzInstance/listCategoryAndClazzInstanceForClazzInstance.action",
	}
)

func NewSession(serverURL, company, user, password string) (s *Session, err error) {
	var jar *cookiejar.Jar

	s = &Session{ServerURL: serverURL, Company: company, User: user, Password: password}

	if s.baseURL, err = url.Parse(serverURL); err != nil {
		goto end
	}

	s.urls = map[string]*url.URL{}
	for k, v := range rawurls {
		u, _ := url.Parse(v)
		s.urls[k] = s.baseURL.ResolveReference(u)
	}

	if jar, err = cookiejar.New(nil); err != nil {
		goto end
	}

	s.client = &http.Client{Jar: jar}

end:
	return s, err
}

func (s *Session) Login() (err error) {
	var req *http.Request
	var resp *http.Response
	var v url.Values = url.Values{}
	var respCookies []*http.Cookie

	// Login.
	v.Set("dispatcher", "bpm")
	v.Set("j_username", fmt.Sprintf("%s,%s", s.User, s.Company))
	v.Set("j_yey", s.Company)
	v.Set("j_username0", s.User)
	v.Set("j_password", s.Password)
	v.Set("button", "登录")

	if req, err = http.NewRequest("POST", s.urls["login"].String(), strings.NewReader(v.Encode())); err != nil {
		goto end
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "*/*")

	if resp, err = http.DefaultTransport.RoundTrip(req); err != nil {
		goto end
	}
	defer resp.Body.Close()

	if !strings.HasSuffix(resp.Header.Get("Location"), rawurls["loginRedirect"]) {
		err = fmt.Errorf("Login redirect URL does not match. Login failed(user name and password do not match.")
		goto end
	}

	respCookies = resp.Cookies()
	if len(respCookies) != 1 || respCookies[0].Name != "JSESSIONID" {
		err = fmt.Errorf("Failed to get JSESSIONID in response cookies.")
		goto end
	}

	// Set cookie for cookiejar manually.
	s.client.Jar.SetCookies(s.baseURL, respCookies)

	s.LoggedIn = true
end:
	return err
}

func (s *Session) Logout() (err error) {
	var req *http.Request
	var resp *http.Response

	if !s.LoggedIn {
		goto end
	}

	if req, err = http.NewRequest("GET", s.urls["logout"].String(), nil); err != nil {
		goto end
	}

	// Set cookie manually following existing cookiejar.
	for _, v := range s.client.Jar.Cookies(s.baseURL) {
		req.AddCookie(v)
	}

	if resp, err = http.DefaultTransport.RoundTrip(req); err != nil {
		goto end
	}
	defer resp.Body.Close()

	s.LoggedIn = false
end:
	return err
}

func (s *Session) SearchStudent(searchBy, value string) (ids []string, err error) {
	var req *http.Request
	var resp *http.Response
	var v url.Values = url.Values{}

	var p = `<a href="/edu/student/basicinfo/viewstudent.action\?student\.id=(?P<id>.*)">`
	var data []byte
	var re *regexp.Regexp
	var matched [][]string

	ids = []string{}

	if value == "" {
		err = fmt.Errorf("Empty search value.")
		goto end
	}

	if !s.LoggedIn {
		err = fmt.Errorf("Not logged in.")
		goto end
	}

	v.Set("searchName", "")
	v.Set("studentTraining.id", "")
	v.Set("action", "search")
	v.Set("searchBy", searchBy)
	v.Set("searchValue", value)
	v.Set("pageEntity.pageRecords", "20")
	v.Set("dispatcher", "search")
	v.Set("studentTrainingName", "")

	if req, err = http.NewRequest("POST", s.urls["studentSearch"].String(), strings.NewReader(v.Encode())); err != nil {
		goto end
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if resp, err = s.client.Do(req); err != nil {
		goto end
	}
	defer resp.Body.Close()

	if data, err = ioutil.ReadAll(resp.Body); err != nil {
		goto end
	}

	re = regexp.MustCompile(p)
	matched = re.FindAllStringSubmatch(string(data), -1)
	for _, v := range matched {
		if len(v) != 2 {
			fmt.Printf("Student ID not found.")
			goto end
		}
		ids = append(ids, v[1])
	}

end:
	return ids, err
}

func (s *Session) SearchStudentByName(name string) (ids []string, err error) {
	return s.SearchStudent("byName", name)
}

func (s *Session) SearchStudentByPhoneNumber(phoneNumber string) (ids []string, err error) {
	return s.SearchStudent("byEmail", phoneNumber)
}

func getClassEventsOfStudent(records [][]string) (events []ClassEvent, err error) {
	events = []ClassEvent{}

	nRow := len(records)
	for i := 1; i <= nRow-1; i++ {
		e := ClassEvent{}

		p := `clazzInstance\.id=(\d+)(?:.*?)clazz\.id=(\d+).*>(\S+)</a>$`
		re := regexp.MustCompile(p)
		matched := re.FindStringSubmatch(records[i][0])
		if len(matched) == 4 {
			e.ClassInstanceId = matched[1]
			e.ClassId = matched[2]
			e.ClassName = html.UnescapeString(matched[3])
		}

		p = `^(\S+)(\d{4}-\d{2}-\d{2}) 00:00:00.0&nbsp;&nbsp;(\d{4}-\d{2}-\d{2})?`
		re = regexp.MustCompile(p)
		matched = re.FindStringSubmatch(records[i][2])
		if len(matched) == 4 {
			e.Status = matched[1]
			e.BeginTime = matched[2]
			e.EndTime = matched[3]
		}

		events = append(events, e)
	}

	return events, err
}

func getStudent(data string) (student *Student, err error) {
	var p = ``
	var re *regexp.Regexp
	var matched []string

	student = &Student{}
	csvs := htmlhelper.TablesToCSVs(data)

	student.Name = html.UnescapeString(strings.TrimLeft(csvs[0][1][1], "<b>"))
	student.SID = csvs[0][1][3]
	student.Status = csvs[0][6][1]
	student.Comments = html.UnescapeString(csvs[0][7][1])

	p = `^(.*?)/`
	re = regexp.MustCompile(p)
	matched = re.FindStringSubmatch(csvs[1][1][1])
	if len(matched) == 2 {
		student.PhoneNumber = matched[1]
	}
	student.ReceiptNumber = csvs[2][1][1]

	// Get classes
	if len(csvs) >= 4 {
		if student.ClassEvents, err = getClassEventsOfStudent(csvs[3]); err != nil {
			goto end
		}
	}
end:
	return student, err
}

func (s *Session) GetStudent(id string) (student *Student, err error) {
	var req *http.Request
	var resp *http.Response
	var urlStr string
	var data []byte

	student = &Student{}

	if !s.LoggedIn {
		err = fmt.Errorf("Not logged in.")
		goto end
	}

	urlStr = fmt.Sprintf("%v?student.id=%v", s.urls["viewStudent"].String(), id)

	if req, err = http.NewRequest("GET", urlStr, nil); err != nil {
		goto end
	}

	if resp, err = s.client.Do(req); err != nil {
		goto end
	}
	defer resp.Body.Close()

	if data, err = ioutil.ReadAll(resp.Body); err != nil {
		goto end
	}

	student, err = getStudent(string(data))
end:
	return student, err
}
