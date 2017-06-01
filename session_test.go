package ming800_test

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"path"

	"github.com/northbright/ming800"
	"github.com/northbright/pathhelper"
)

// Run "go test -c && ./ming800.test" to load config.json and do the test.
func Example() {
	// 1. Create a "config.json" like this to load settings:
	/*{
	        "server_url": "http://localhost:8080",
		"company": "my_company",
		"user": "Frank",
		"password": "my_password"
	}*/

	// 2. Run "go test -c && ./ming800.test" to load config.json and do the test.

	type Config struct {
		ServerURL string `json:"server_url"`
		Company   string `json:"company"`
		User      string `json:"user"`
		Password  string `json:"password"`
	}

	var err error
	var buf []byte
	var currentDir, configFile string
	var s *ming800.Session
	var config Config
	var names = []string{"Emma", "王"}
	var phoneNums = []string{"135"}
	var ids []string
	var categories []ming800.Category
	var classes []ming800.Class
	var students []ming800.Student

	currentDir, _ = pathhelper.GetCurrentExecDir()
	configFile = path.Join(currentDir, "config.json")

	// Load Conifg
	if buf, err = ioutil.ReadFile(configFile); err != nil {
		log.Printf("Load config file error: %v\n", err)
		goto end
	}

	if err = json.Unmarshal(buf, &config); err != nil {
		log.Printf("Parse config err: %v\n", err)
		goto end
	}

	// New a session
	if s, err = ming800.NewSession(config.ServerURL, config.Company, config.User, config.Password); err != nil {
		log.Printf("NewSession() error: %v\n", err)
		goto end
	}

	// Login
	if err = s.Login(); err != nil {
		log.Printf("Login() error: %v\n", err)
		goto end
	}

	log.Printf("login() successfully.\n")

	// Search
	// 1. Search student by name.
	for _, name := range names {
		log.Printf("SearchStudentByName(%v) starting...\n", name)

		if ids, err = s.SearchStudentByName(name); err != nil {
			log.Printf("error: %v\n", err)
			goto end
		}

		log.Printf("Found %v ids: %v\n\n", len(ids), ids)

		// Get students.
		log.Printf("Get students starting...\n")
		for _, id := range ids {
			student := ming800.Student{}
			if student, err = s.GetStudent(id); err != nil {
				log.Printf("error: %v\n", err)
				goto end
			}
			log.Printf("%v, %v, %v\n", student.Name, student.PhoneNumber, student.ReceiptNumber)
			for _, e := range student.ClassEvents {
				log.Printf("%v, %v, %v, %v\n", e.ClassName, e.Status, e.BeginTime, e.EndTime)
			}
		}
	}

	// 2. Search student by phone.
	for _, phoneNum := range phoneNums {
		log.Printf("SearchStudentByPhoneNumber(%v) starting...\n", phoneNum)

		if ids, err = s.SearchStudentByPhoneNumber(phoneNum); err != nil {
			log.Printf("error: %v\n", err)
			goto end
		}

		log.Printf("Found %v ids: %v\n\n", len(ids), ids)

		// Get students.
		log.Printf("Get students starting...\n")
		for _, id := range ids {
			student := ming800.Student{}
			if student, err = s.GetStudent(id); err != nil {
				log.Printf("error: %v\n", err)
				goto end
			}
			log.Printf("%v, %v, %v\n", student.Name, student.PhoneNumber, student.ReceiptNumber)
			for _, e := range student.ClassEvents {
				log.Printf("%v, %v, %v, %v\n", e.ClassName, e.Status, e.BeginTime, e.EndTime)
			}

		}
	}

	// Get current categories and classes.
	log.Printf("Get current categories and classes starting...\n")
	if categories, classes, err = s.GetCurrentCategoriesAndClasses(); err != nil {
		log.Printf("error: %v\n", err)
		goto end
	}

	log.Printf("Catetories: \n")
	for i, category := range categories {
		log.Printf("%v: %v\n", i, category)
	}

	log.Printf("Classes: \n")
	for _, class := range classes {
		log.Printf("%v\n", class)
	}

	// Get current students.
	log.Printf("Get current students starting...\n")
	if students, err = s.GetCurrentStudents(); err != nil {
		log.Printf("error: %v\n", err)
		goto end
	}

	for i, s := range students {
		log.Printf("%v: %v\n", i, s)
	}

	// Logout
	if err = s.Logout(); err != nil {
		log.Printf("Logout() error: %v\n", err)
		goto end
	}

	log.Printf("logout() successfully.\n")
end:
	// Output:
}
