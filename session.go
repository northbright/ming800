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
	// Periods are the periods of the class.
	// One class may have 2 or more periods.
	Periods []string
}

// Student represeents the student information.
type Student struct {
	// ID is the internal ID of student.
	ID string
	// Name is the name of student.
	Name string
	// PhoneNum is the phone number of the contact for the student.
	PhoneNum string
	// Details store student information in key-value map.
	Details map[string]string
}

// WalkProcessor interface need users to implement callback functions while walking ming800.
type WalkProcessor interface {
	// ClassHandler is the callback when a class is found.
	ClassHandler(class *Class) error
	// StudentHandler is the callback when a student is found.
	StudentHandler(class *Class, student *Student) error
}

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
		"viewStudent":              "/edu/student/basicinfo/viewstudent.action?student.id=",
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
		respCookies []*http.Cookie
	)
	// Login.
	v := url.Values{}
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

func (s *Session) getStudent(row []string) (*Student, error) {
	var err error

	// First () catches student ID, second () catches student name.
	p := `student.id=(.+?)&.*">(.*)</a>`
	re := regexp.MustCompile(p)

	if len(row) != 9 {
		return nil, fmt.Errorf("failed to parse student info")
	}

	student := &Student{}

	matched := re.FindStringSubmatch(row[0])
	if len(matched) != 3 {
		return nil, fmt.Errorf("failed to find student name")
	}
	student.ID = html.UnescapeString(matched[1])
	student.Name = html.UnescapeString(matched[2])
	student.PhoneNum = html.UnescapeString(row[3])

	// Get student details include customized column(e.g. ID card No).
	student.Details, err = s.GetStudentDetails(student.ID)
	if err != nil {
		return nil, fmt.Errorf("GetStudentDetails() error: %v\n", err)
	}

	return student, nil
}

// walkStudentsOfClassOfOnePage is the internal implementation for Session.walkStudentsOfClass.
// It parses the HTTP response body to walk students of the class.
func (s *Session) walkStudentsOfClassOfOnePage(content string, class *Class, processor WalkProcessor) error {
	csvs := htmlhelper.TablesToCSVs(content)
	// Skip if no students.
	if len(csvs) != 1 {
		return nil
	}

	table := csvs[0]
	nRow := len(table)

	// Skip first row and check row number.
	if nRow <= 1 {
		return nil
	}

	concurrency := 30
	sem := make(chan struct{}, concurrency)
	// Make a buffered channel to store returned errors from goroutines.
	chError := make(chan error, nRow-1)
	// Make a buffered channel to store students.
	chStudent := make(chan *Student, nRow-1)

	for i := 1; i < nRow; i++ {
		row := table[i]
		// After first "concurrency" amount of goroutines started,
		// It'll block starting new goroutines until one running goroutine finishs.
		sem <- struct{}{}

		go func(i int) {
			defer func() { <-sem }()

			student, err := s.getStudent(row)
			chError <- err
			chStudent <- student
		}(i)
	}

	// After last goroutine is started,
	// there're still "concurrency" amount of goroutines running.
	// Make sure wait all goroutines to finish.
	for j := 0; j < cap(sem); j++ {
		sem <- struct{}{}
	}

	// Close the error channel.
	close(chError)
	// Close the student channel.
	close(chStudent)

	// Check errors returned from goroutines.
	for e := range chError {
		if e != nil {
			return fmt.Errorf("getStudent() or processor.StudentHandler() error: %v", e)
		}
	}

	// Invoke student handler callback.
	for student := range chStudent {
		if err := processor.StudentHandler(class, student); err != nil {
			return fmt.Errorf("StudentHandler() error: %v", err)
		}
	}

	return nil
}

// walkStudentsOfClass walks the students of given class.
func (s *Session) walkStudentsOfClass(classID string, class *Class, pageIndex int, processor WalkProcessor) error {
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
	if err = s.walkStudentsOfClassOfOnePage(content, class, processor); err != nil {
		return err
	}

	// If it's first page( page index is 1 for ming800),
	// get indexes of next pages and walk them.
	// Skip for next pages.
	if pageIndex <= 1 {
		pageCount := getPageCountForStudentsOfClass(content)

		for i := 1; i < pageCount; i++ {
			if err = s.walkStudentsOfClass(classID, class, i+1, processor); err != nil {
				return err
			}
		}
	}

	return nil
}

// getPeriods parse the HTTP response body to get the class periods.
// One class may have one or more periods.
func getPeriods(content string) []string {
	var periods []string

	arr := strings.Split(strings.TrimRight(content, `<br>`), `<br>`)
	p := `^\S+\d{2}:\d{2}-\d{2}:\d{2}`

	re := regexp.MustCompile(p)
	for _, v := range arr {
		if period := re.FindString(v); period != "" {
			periods = append(periods, period)
		}
	}
	return periods
}

// getClass gets the class info by given class ID.
func (s *Session) getClass(ID string) (*Class, error) {
	var (
		err   error
		class Class
	)

	if !s.LoggedIn {
		return nil, fmt.Errorf("Not logged in")
	}

	urlStr := fmt.Sprintf("%v%v", s.urls["viewClass"].String(), ID)
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	csvs := htmlhelper.TablesToCSVs(string(data))
	if len(csvs) != 2 {
		return nil, fmt.Errorf("no class tables found. class ID: %v, csvs: %v", ID, csvs)
	}

	class.Category = strings.TrimRight(html.UnescapeString(csvs[0][1][1]), `(普通)`)
	class.Name = html.UnescapeString(csvs[0][2][1])

	// Skip if table 2 is empty.
	if len(csvs[1]) != 2 {
		return &class, nil
	}

	// Get teachers.
	str := strings.TrimRight(html.UnescapeString(csvs[1][1][6]), `<br>`)
	class.Teachers = strings.Split(str, `<br>`)

	class.ClassRoom = html.UnescapeString(csvs[1][1][7])

	// Get periods.
	class.Periods = getPeriods(csvs[1][1][8])

	return &class, nil
}

// Walk walks through the ming800.
func (s *Session) Walk(processor WalkProcessor) error {
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

			// Walk class
			if err = processor.ClassHandler(class); err != nil {
				return fmt.Errorf("ClassHandler() error: %v", err)
			}

			// Walk students of class.
			if err = s.walkStudentsOfClass(classID, class, 1, processor); err != nil {
				return fmt.Errorf("walkStudentsOfClass() error: %v", err)
			}
		}
	}

	return nil
}

// GetViewStudentURL returns the URL of view student which response contains details of the student include customized column(e.g. ID card number).
func (s *Session) GetViewStudentURL(ID string) string {
	return fmt.Sprintf("%s%s", s.urls["viewStudent"].String(), ID)
}

func (s *Session) GetStudentDetails(ID string) (map[string]string, error) {
	var (
		err     error
		details = map[string]string{}
	)

	if !s.LoggedIn {
		return nil, fmt.Errorf("Not logged in")
	}

	urlStr := s.GetViewStudentURL(ID)
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Convert HTML tables to CSV.
	csvs := htmlhelper.TablesToCSVs(string(data))
	for _, csv := range csvs {
		for _, row := range csv {
			// Check if column number is even number.
			// Columns can be converted into a key-value map.
			nColumns := len(row)
			if nColumns%2 != 0 {
				continue
			}

			for i := 0; i < nColumns; i += 2 {
				k := htmlhelper.RemoveAllElements(row[i])
				k = html.UnescapeString(k)

				v := htmlhelper.RemoveAllElements(row[i+1])
				v = html.UnescapeString(v)

				details[k] = v
			}
		}
	}

	return details, nil
}
