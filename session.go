package ming800

import (
	"fmt"
	"html"
	"io/ioutil"
	//"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/northbright/htmlhelper"
)

// Session represents the login session and provides methods to interactive with ming800.
type Session struct {
	// ServerURL is the server base URL of ming800.
	ServerURL string
	// Company is the company(orgnization) name for login.
	Company string
	// User is the user name of ming800.
	User string
	// Password is the user's password.
	Password string
	baseURL  *url.URL
	jar      *cookiejar.Jar
	client   *http.Client
	// LoggedIn represents login status.
	LoggedIn bool
	// urls contains *url.URL of actions.
	urls map[string]*url.URL
}

// Class represents class information.
type Class struct {
	// Name is name of the class.
	Name string
	// Category is the category of the class.
	Category string
	// Teachers are the teachers of the class. One class can have multiple teachers.
	Teachers []string
	// ClassRoom is the class room of the class.
	ClassRoom string
	// Period is the period of the class.
	Period string
}

// Student represeents the student information.
type Student struct {
	// Name is the name of student.
	Name string
	// PhoneNum is the phone number of the contact for the student.
	PhoneNum string
}

// ClassHandler is the handler that a class is found while walking through the ming800.
type ClassHandler func(class Class)

// StudentHandler is the handler that a student is found while walking through the ming800.
type StudentHandler func(class Class, student Student)

var (
	// rawurls contains actions' raw URLs.
	rawurls = map[string]string{
		"login":                    "/j_spring_security_check",
		"loginRedirect":            "/standard/mainController.controller",
		"logout":                   "/j_spring_security_logout",
		"mainControlloer":          "/standard/mainController.controller",
		"listCategoriesAndClasses": "/edu/base/clazzInstance/listCategoryAndClazzInstanceForStudent.action",
		"viewClass":                "edu/base/clazzInstance/viewClazzInstance.action?clazzInstance.id=",
		"listStudentsOfClass":      "/edu/student/basicinfo/liststudentbyclazzinstance.action?clazzInstance.id=",
	}
)

// NewSession creates a new session of ming800.
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

// Login performs the login action.
func (s *Session) Login() (err error) {
	var (
		req         *http.Request
		resp        *http.Response
		v           url.Values
		respCookies []*http.Cookie
	)
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
		err = fmt.Errorf("login redirect URL does not match. login failed: user name and password do not match")
		goto end
	}

	respCookies = resp.Cookies()
	if len(respCookies) != 1 || respCookies[0].Name != "JSESSIONID" {
		err = fmt.Errorf("failed to get JSESSIONID in response cookies")
		goto end
	}

	// Set cookie for cookiejar manually.
	s.client.Jar.SetCookies(s.baseURL, respCookies)

	s.LoggedIn = true
end:
	return err
}

// Logout performs the log out action.
func (s *Session) Logout() (err error) {
	var (
		req  *http.Request
		resp *http.Response
	)

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

// getPageCountForStudentsOfClass parses the HTTP response and find the page count for students of the class.
func getPageCountForStudentsOfClass(content string) int {
	var count int

	p := `共(\d+)页`
	re := regexp.MustCompile(p)

	matched := re.FindStringSubmatch(content)
	if len(matched) != 2 {
		return 0
	}

	count, _ = strconv.Atoi(matched[1])
	return count
}

// walkStudentsOfClass is the internal implementation for Session.walkStudentsOfClass.
// It parses the HTTP response body to walk students of the class.
func walkStudentsOfClass(content string, class Class, studentFn StudentHandler) error {
	csvs := htmlhelper.TablesToCSVs(content)
	// Skip if no students.
	if len(csvs) != 1 {
		return nil
	}

	p := `">(.*)</a>`
	re := regexp.MustCompile(p)

	table := csvs[0]
	for i, row := range table {
		if i == 0 {
			continue
		}

		if len(row) != 9 {
			return fmt.Errorf("failed to parse student info")
		}

		s := Student{}

		matched := re.FindStringSubmatch(row[0])
		if len(matched) != 2 {
			return fmt.Errorf("failed to find student name")
		}
		s.Name = html.UnescapeString(matched[1])
		s.PhoneNum = html.UnescapeString(row[3])

		studentFn(class, s)
	}

	return nil
}

// walkStudentsOfClass walks the students of given class.
func (s *Session) walkStudentsOfClass(classID string, class Class, pageIndex int, studentFn StudentHandler) error {
	var (
		err error
	)

	if !s.LoggedIn {
		return fmt.Errorf("Not logged in")
	}

	urlStr := fmt.Sprintf("%v%v%v%v", s.urls["listStudentsOfClass"].String(), classID, "&pageEntity.pageIndex=", pageIndex)
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	content := string(data)
	if err = walkStudentsOfClass(content, class, studentFn); err != nil {
		return err
	}

	// If it's first page( page index is 1 for ming800),
	// get indexes of next pages and walk them.
	// Skip for next pages.
	if pageIndex <= 1 {
		pageCount := getPageCountForStudentsOfClass(content)

		for i := 1; i < pageCount; i++ {
			if err = s.walkStudentsOfClass(classID, class, i+1, studentFn); err != nil {
				return err
			}
		}
	}

	return nil
}

// getClass gets the class info by given class ID.
func (s *Session) getClass(ID string) (Class, error) {
	var (
		err   error
		class Class
	)

	if !s.LoggedIn {
		return class, fmt.Errorf("Not logged in")
	}

	urlStr := fmt.Sprintf("%v%v", s.urls["viewClass"].String(), ID)
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return class, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return class, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return class, err
	}

	csvs := htmlhelper.TablesToCSVs(string(data))
	if len(csvs) != 2 {
		return class, fmt.Errorf("no class tables found")
	}

	class.Category = strings.TrimRight(html.UnescapeString(csvs[0][1][1]), `(普通)`)
	class.Name = html.UnescapeString(csvs[0][2][1])

	// Skip if table 2 is empty.
	if len(csvs[1]) != 2 {
		return class, nil
	}

	// Get teachers.
	str := strings.TrimRight(html.UnescapeString(csvs[1][1][6]), `<br>`)
	class.Teachers = strings.Split(str, `<br>`)

	class.ClassRoom = html.UnescapeString(csvs[1][1][7])

	// Get period.
	p := `^\S+\d{2}:\d{2}-\d{2}:\d{2}`
	re := regexp.MustCompile(p)
	class.Period = re.FindString(csvs[1][1][8])

	return class, nil
}

// Walk walks through the ming800.
func (s *Session) Walk(classFn ClassHandler, studentFn StudentHandler) error {
	var (
		err error
	)

	if !s.LoggedIn {
		return fmt.Errorf("Not logged in")
	}

	req, err := http.NewRequest("GET", s.urls["listCategoriesAndClasses"].String(), nil)
	if err != nil {
		return err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Pattern to find class name.
	patterns := map[string]string{
		"getClassName": `>(.*)</a>$`,
		"getClassID":   `clazzInstance\.id=(\d+)&action=view`,
	}

	// Compile regexps by given patterns.
	reArr := map[string]*regexp.Regexp{}
	for k, v := range patterns {
		reArr[k] = regexp.MustCompile(v)
	}

	// Parse HTML response to get tables(CSV).
	csvs := htmlhelper.TablesToCSVs(string(data))
	for _, csv := range csvs {
		for j, row := range csv {
			if j == 0 {
				continue
			}

			// Get class IDs
			matched := reArr["getClassID"].FindStringSubmatch(row[6])
			if len(matched) != 2 {
				return fmt.Errorf("Failed to get class ID")
			}

			classID := matched[1]
			// Get class data by ID.
			class, err := s.getClass(classID)
			if err != nil {
				return fmt.Errorf("getClass() error: %v", err)
			}
			classFn(class)

			// Walk students of class.
			if err = s.walkStudentsOfClass(classID, class, 1, studentFn); err != nil {
				return fmt.Errorf("walkStudentsOfClass() error: %v", err)
			}

		}
	}

	return nil
}
