package ming800

import (
	"fmt"
	"html"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
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
	CategoryId      string
	ClassName       string
	Status          string
	BeginTime       string
	EndTime         string
}

type Class struct {
	ClassId         string
	ClassName       string
	ClassInstanceId string
	CategoryId      string
	Status          string
}

type Category struct {
	Id   string
	Name string
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
		"viewCategory":         "/edu/base/clazz/viewClazz.action?clazz.id=",
		"listStudentsOfClass":  "/edu/student/basicinfo/liststudentbyclazzinstance.action?clazzInstance.id=",
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
			e.CategoryId = matched[2]
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

func getStudent(data string) (student Student, err error) {
	var p = ``
	var re *regexp.Regexp
	var matched []string

	student = Student{}
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

func (s *Session) GetStudent(id string) (student Student, err error) {
	var req *http.Request
	var resp *http.Response
	var urlStr string
	var data []byte

	student = Student{}

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

func (s *Session) GetCategory(id string) (category Category, err error) {
	var urlStr = ""
	var req *http.Request
	var resp *http.Response
	var data []byte
	var csvs [][][]string

	category = Category{}

	if !s.LoggedIn {
		err = fmt.Errorf("Not logged in.")
		goto end
	}

	urlStr = fmt.Sprintf("%v%v", s.urls["viewCategory"].String(), id)
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

	csvs = htmlhelper.TablesToCSVs(string(data))
	if len(csvs) != 2 || len(csvs[0]) != 6 || len(csvs[0][2]) != 4 {
		err = fmt.Errorf("Failed to get category details.")
		goto end
	}

	category.Id = id
	category.Name = csvs[0][2][1]

end:
	return category, err

}

func (s *Session) getCategories(data string) (categories []Category, err error) {
	p := `/edu/base/clazz/viewClazz\.action\?clazz\.id=(\d+)`
	re := regexp.MustCompile(p)
	matched := re.FindAllStringSubmatch(data, -1)

	for _, m := range matched {
		category := Category{}

		if category, err = s.GetCategory(m[1]); err != nil {
			goto end
		}
		categories = append(categories, category)
	}
end:
	return categories, err
}

func getClasses(data string) (classes []Class, err error) {
	csvs := htmlhelper.TablesToCSVs(string(data))
	for i, csv := range csvs {
		for j, row := range csv {
			if j == 0 {
				continue
			}

			c := Class{}
			c.ClassId = row[1]

			p := `clazzInstance\.id=(\d+)&clazz\.id=(\d+).*?>(.*?)&nbsp;</a>$`
			re := regexp.MustCompile(p)
			matched := re.FindStringSubmatch(row[0])
			if len(matched) != 4 {
				err = fmt.Errorf("Failed to parse class. table: %v, row: %v\n", i, j)
				goto end
			}

			c.ClassName = strings.Replace(matched[3], " 00:00:00.0", "", -1)
			c.ClassInstanceId = matched[1]
			c.CategoryId = matched[2]

			c.Status = row[3]

			classes = append(classes, c)
		}
	}

end:
	return classes, err

}

func (s *Session) GetCurrentCategoriesAndClasses() (categories []Category, classes []Class, err error) {
	var req *http.Request
	var resp *http.Response
	var data []byte

	if !s.LoggedIn {
		err = fmt.Errorf("Not logged in.")
		goto end
	}

	if req, err = http.NewRequest("GET", s.urls["listCategoryAndClass"].String(), nil); err != nil {
		goto end
	}

	if resp, err = s.client.Do(req); err != nil {
		goto end
	}
	defer resp.Body.Close()

	if data, err = ioutil.ReadAll(resp.Body); err != nil {
		goto end
	}

	if categories, err = s.getCategories(string(data)); err != nil {
		goto end
	}

	if classes, err = getClasses(string(data)); err != nil {
		goto end
	}

end:
	return categories, classes, err
}

func getAllStudentPageLinks(data string) (links []string) {
	p := `<a id="pageindex_\d+" href="(.*?)"`
	re := regexp.MustCompile(p)
	matched := re.FindAllStringSubmatch(data, -1)

	for _, m := range matched {
		links = append(links, m[1])
	}

	return links
}

func (s *Session) getStudentsPerPage(link string) (students []Student, err error) {
	var u *url.URL
	var urlStr string
	var req *http.Request
	var resp *http.Response
	var data []byte
	var csvs [][][]string
	var p string
	var re *regexp.Regexp

	if !s.LoggedIn {
		err = fmt.Errorf("Not logged in.")
		goto end
	}

	u, _ = url.Parse(link)
	urlStr = s.baseURL.ResolveReference(u).String()
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

	csvs = htmlhelper.TablesToCSVs(string(data))
	if len(csvs) != 1 {
		err = fmt.Errorf("Failed to get student list table.")
		goto end
	}

	p = `<a href=".*?student\.id=(.*?)&`
	re = regexp.MustCompile(p)
	for i, row := range csvs[0] {
		// Skip first empty row
		if i == 0 {
			continue
		}

		if len(row) < 0 {
			err = fmt.Errorf("Failed to get student list row.")
		}

		matched := re.FindStringSubmatch(row[0])

		id := matched[1]
		student := Student{}
		if student, err = s.GetStudent(id); err != nil {
			goto end
		}

		students = append(students, student)
	}

end:
	return students, err
}

func (s *Session) getStudentsOfClass(data string) (students []Student, err error) {
	links := getAllStudentPageLinks(data)

	for _, link := range links {
		studentsPerPage := []Student{}
		if studentsPerPage, err = s.getStudentsPerPage(link); err != nil {
			goto end
		}
		students = append(students, studentsPerPage...)
	}
end:
	return students, err
}

func (s *Session) GetStudentsOfClass(classId string) (students []Student, err error) {
	var urlStr string
	var req *http.Request
	var resp *http.Response
	var data []byte

	if !s.LoggedIn {
		err = fmt.Errorf("Not logged in.")
		goto end
	}

	urlStr = fmt.Sprintf("%v%v", s.urls["listStudentsOfClass"].String(), classId)
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

	if students, err = s.getStudentsOfClass(string(data)); err != nil {
		goto end
	}

end:
	return students, err
}

func (s *Session) GetCurrentStudents() (students []Student, err error) {
	var classes []Class

	if _, classes, err = s.GetCurrentCategoriesAndClasses(); err != nil {
		goto end
	}

	for _, class := range classes {
		studentsOfClass := []Student{}
		if studentsOfClass, err = s.GetStudentsOfClass(class.ClassInstanceId); err != nil {
			goto end
		}

		students = append(students, studentsOfClass...)
	}
end:
	return students, err
}
