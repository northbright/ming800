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

type Student struct {
	Name          string
	SID           string
	Status        string
	Comments      string
	PhoneNumber   string
	ReceiptNumber string
	Classes       []string
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

func getStudent(data string) (student *Student, err error) {
	student = &Student{}

	arr := map[string]*struct {
		pattern         string
		matchedArrayLen int
		matchedIndex    int
		value           string
	}{
		"name":          {`姓名/别名\s*</td>\s*<td.*>\s*<b>(.*?)\r\n`, 2, 1, ""},
		"sid":           {`学号\s*</td>\s*<td.*>\s*(\S*?)\r\n`, 2, 1, ""},
		"status":        {`审核状态\s*</td>\s*<td.*>\s*(\S?)\r\n`, 2, 1, ""},
		"comments":      {`备注\s*</td>\s*<td.*>\s*((.|\s)*?)</td>`, 3, 1, ""},
		"phoneNumber":   {`电话/联系人\s*</td>\s*<td.*>\s*(.*?)/?\r\n`, 2, 1, ""},
		"receiptNumber": {`发票号</td><td.*>(.*?)</td>`, 2, 1, ""},
	}

	/*
		patterns := map[string]string{
			"name":          `姓名/别名\s*</td>\s*<td.*>\s*<b>(.*?)\n`,
			"sid":           `学号\s*</td>\s*<td.*>\s*(.*?)\n`,
			"status":        `审核状态\s*</td>\s*<td.*>\s*(.*?)\n`,
			"comments":      `备注\s*</td>\s*<td.*>\s*((.|\s)*?)</td>`,
			"phoneNumber":   `电话/联系人\s*</td>\s*<td.*>\s*(.*?)/?\n`,
			"receiptNumber": `发票号</td><td.*>(.*?)</td>`,
		}
	*/

	for k, v := range arr {
		re := regexp.MustCompile(v.pattern)
		matched := re.FindStringSubmatch(data)
		if len(matched) != v.matchedArrayLen {
			err = fmt.Errorf("No student info matched.")
			goto end
		}
		arr[k].value = html.UnescapeString(matched[v.matchedIndex])
	}

	student.Name = arr["name"].value
	student.SID = arr["sid"].value
	student.Status = arr["status"].value
	student.Comments = arr["comments"].value
	student.PhoneNumber = arr["phoneNumber"].value
	student.ReceiptNumber = arr["receiptNumber"].value

end:
	return student, err
}

func (s *Session) GetStudent(id string) (student *Student, err error) {
	var req *http.Request
	var resp *http.Response
	var urlStr string
	var data []byte
	var c *http.Client

	student = &Student{}

	if !s.LoggedIn {
		err = fmt.Errorf("Not logged in.")
		goto end
	}

	urlStr = fmt.Sprintf("%v?student.id=%v", s.urls["viewStudent"].String(), id)

	if req, err = http.NewRequest("GET", urlStr, nil); err != nil {
		goto end
	}

	//if resp, err = s.client.Do(req); err != nil {
	//	goto end
	//}
	c = &http.Client{Jar: s.client.Jar}
	if resp, err = c.Do(req); err != nil {
		goto end
	}
	defer resp.Body.Close()

	if data, err = ioutil.ReadAll(resp.Body); err != nil {
		goto end
	}

	student, err = getStudent(string(data))
	//student.Name = "xx"
	//student.PhoneNumber = "123456"
end:
	return student, err
}
