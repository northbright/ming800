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
	// MobilePhoneNum is the phone number of the contact for the student.
	MobilePhoneNum string
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

// Logout performs the log out action.
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

/*
// SearchStudent searches the student in ming800 by given search type and value.
//
// Params:
//     searchBy: search type.
//               Available values: "byName" for search by name and "byEmail" for search by phone.
//     value: search value.
// Return:
//     returns the IDs of matched students.
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

// SearchStudentByName searches student by name.
func (s *Session) SearchStudentByName(name string) (ids []string, err error) {
	return s.SearchStudent("byName", name)
}

// SearchStudentByMobilePhoneNum searches student by phone number.
func (s *Session) SearchStudentByMobilePhoneNum(phoneNumber string) (ids []string, err error) {
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
			e.InstanceID = matched[1]
			e.CategoryId = matched[2]
			e.Name = html.UnescapeString(matched[3])
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
		student.MobilePhoneNum = matched[1]
	}
	student.ReceiptNumber = csvs[2][1][1]

	// Get class events.
	if len(csvs) >= 4 {
		if student.ClassEvents, err = getClassEventsOfStudent(csvs[3]); err != nil {
			goto end
		}
	}
end:
	return student, err
}

// GetStudent gets the student by ID.
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

// GetCategory gets the category by ID.
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

func getClassSchedule(data string) (ClassSchedule, error) {
	var (
		err           error
		classSchedule ClassSchedule
		row           []string
	)

	csvs := htmlhelper.TablesToCSVs(data)
	if len(csvs) != 2 {
		err = fmt.Errorf("Class schedule table not found")
		return classSchedule, err
	}

	// Check if no class schedule data. It may not exist.
	if len(csvs[1]) != 2 {
		return classSchedule, err
	}

	row = csvs[1][1]
	classSchedule.BeginDate = row[2]
	classSchedule.EndDate = row[3]

	// Get teachers.
	s := html.UnescapeString(strings.TrimRight(row[4], "<br>"))
	classSchedule.Teachers = strings.Split(s, "<br>")

	classSchedule.ClassRoom = html.UnescapeString(row[5])

	// Get period.
	p := `^\S+\s\d{2}:\d{2}-\d{2}:\d{2}`
	re := regexp.MustCompile(p)
	classSchedule.Period = re.FindString(row[6])

	return classSchedule, nil
}

// GetClassSchedule gets the class schedule data by given class instance ID.
func (s *Session) GetClassSchedule(classInstanceId string) (ClassSchedule, error) {
	var err error
	var req *http.Request
	var resp *http.Response
	var urlStr string
	var schedule ClassSchedule
	var data []byte

	if !s.LoggedIn {
		err = fmt.Errorf("Not logged in.")
		goto end
	}

	urlStr = fmt.Sprintf("%v%v", s.urls["viewClassSchedule"].String(), classInstanceId)
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

	schedule, err = getClassSchedule(string(data))
end:
	return schedule, err
}

func (s *Session) getClasses(data string) (classes []Class, err error) {
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

			c.Name = strings.Replace(matched[3], " 00:00:00.0", "", -1)
			c.InstanceID = matched[1]
			c.CategoryId = matched[2]

			c.Status = row[3]
			if c.Schedule, err = s.GetClassSchedule(c.InstanceID); err != nil {
				goto end
			}

			classes = append(classes, c)
		}
	}

end:
	return classes, err

}

// GetCurrentCategoriesAndClasses gets the current categories and classes in ming800.
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

	if classes, err = s.getClasses(string(data)); err != nil {
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

// GetStudentsOfClass gets the students of one class by given class instance ID.
func (s *Session) GetStudentsOfClass(classInstanceId string) (students []Student, err error) {
	var urlStr string
	var req *http.Request
	var resp *http.Response
	var data []byte

	if !s.LoggedIn {
		err = fmt.Errorf("Not logged in.")
		goto end
	}

	urlStr = fmt.Sprintf("%v%v", s.urls["listStudentsOfClass"].String(), classInstanceId)
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

// GetCurrentStudents gets all current students in ming800.
func (s *Session) GetCurrentStudents() (students []Student, err error) {
	var classes []Class

	if _, classes, err = s.GetCurrentCategoriesAndClasses(); err != nil {
		goto end
	}

	for _, class := range classes {
		studentsOfClass := []Student{}
		if studentsOfClass, err = s.GetStudentsOfClass(class.InstanceID); err != nil {
			goto end
		}

		students = append(students, studentsOfClass...)
	}
end:
	return students, err
}
*/

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
		s.MobilePhoneNum = html.UnescapeString(row[3])

		studentFn(class, s)
	}

	return nil
}

func (s *Session) WalkStudentsOfClass(classID string, class Class, pageIndex int, studentFn StudentHandler) error {
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
			if err = s.WalkStudentsOfClass(classID, class, i+1, studentFn); err != nil {
				return err
			}
		}
	}

	return nil
}

// GetClass gets the class info by given class ID.
func (s *Session) GetClass(ID string) (Class, error) {
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
	/*for i, csv := range csvs {
		for j, row := range csv {
			log.Printf("i: %v, j: %v, row(l=%v): %v", i, j, len(row), row)
		}
	}*/

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
			class, err := s.GetClass(classID)
			if err != nil {
				return fmt.Errorf("GetClass() error: %v", err)
			}
			classFn(class)

			// Walk students of class.
			if err = s.WalkStudentsOfClass(classID, class, 1, studentFn); err != nil {
				return fmt.Errorf("WalkStudentsOfClass() error: %v", err)
			}

		}
	}

	return nil
}
