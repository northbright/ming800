package ming800

import (
	"fmt"
	//"io/ioutil"
	//"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	//"path"
	"strings"
	//"regexp"
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
}

var (
	loginActionURL    = "/j_spring_security_check"
	loginRedirectURL  = "/standard/mainController.controller"
	logoutActionURL   = "/j_spring_security_logout"
	logoutRedirectURL = "/index.jsp"
	mainControllerURL = "/standard/mainController.controller"
)

func NewSession(serverURL, company, user, password string) (s *Session, err error) {
	var jar *cookiejar.Jar

	s = &Session{ServerURL: serverURL, Company: company, User: user, Password: password}

	if s.baseURL, err = url.Parse(serverURL); err != nil {
		goto end
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
	var loginURL *url.URL
	var loginURLStr string
	var respCookies []*http.Cookie

	if loginURL, err = url.Parse(loginActionURL); err != nil {
		goto end
	}

	loginURLStr = s.baseURL.ResolveReference(loginURL).String()

	// Login.
	v.Set("dispatcher", "bpm")
	v.Set("j_username", fmt.Sprintf("%s,%s", s.User, s.Company))
	v.Set("j_yey", s.Company)
	v.Set("j_username0", s.User)
	v.Set("j_password", s.Password)
	v.Set("button", "登录")

	if req, err = http.NewRequest("POST", loginURLStr, strings.NewReader(v.Encode())); err != nil {
		goto end
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "*/*")

	if resp, err = http.DefaultTransport.RoundTrip(req); err != nil {
		goto end
	}
	defer resp.Body.Close()

	if !strings.HasSuffix(resp.Header.Get("Location"), loginRedirectURL) {
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
	var logoutURL *url.URL
	var logoutURLStr string

	if !s.LoggedIn {
		goto end
	}

	if logoutURL, err = url.Parse(logoutActionURL); err != nil {
		goto end
	}

	logoutURLStr = s.baseURL.ResolveReference(logoutURL).String()
	if req, err = http.NewRequest("GET", logoutURLStr, nil); err != nil {
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
