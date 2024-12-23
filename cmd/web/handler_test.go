package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"snippetbox/internal/assert"
	"testing"
)

func TestPing(t *testing.T) {
	//初始化一个新的httptest.ResponseRecorder本质上是http.ResponseWriter
	//可以用来记录响应状态码，标头，正文
	recorder := httptest.NewRecorder()

	//初始化一个新的虚拟http.Request
	request, err := http.NewRequest(http.MethodGet, "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	//调用ping处理函数
	ping(recorder, request)

	//调用http.ResponseRecorder的result函数获取由ping处理函数生成的http.Response
	response := recorder.Result()

	assert.Equal(t, response.StatusCode, http.StatusOK)
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	bytes.TrimSpace(body)
	assert.Equal(t, string(body), "OK")
}

// 端到端测试
func TestPing2(t *testing.T) {
	//创建一个新的application实例
	app := newTestApplication(t)
	//创建一个新的测试服务，将app.router返回的handler作为值传递进去
	ts := newTestServer(t, app.router())
	defer ts.Close()

	code, _, body := ts.get(t, "/ping")
	assert.Equal(t, code, http.StatusOK)

	assert.Equal(t, body, "OK")

}

func TestSnippetView(t *testing.T) {
	app := newTestApplication(t)

	//创建一个新的测试服务
	server := newTestServer(t, app.router())
	defer server.Close()

	//创建测试表
	tests := []struct {
		name     string
		urlPath  string
		wantCode int
		wantBody string
	}{
		{ //测试通不过，因为加载不了模版文件，不知道为什么
			name:     "Valid ID",
			urlPath:  "/snippet/view/1",
			wantCode: http.StatusOK,
			wantBody: "An old silent pond...",
		},
		{
			name:     "Non-existent ID",
			urlPath:  "/snippet/view/2",
			wantCode: http.StatusNotFound,
		},
		{
			name:     "Negative ID",
			urlPath:  "/snippet/view/-1",
			wantCode: http.StatusNotFound,
		},
		{
			name:     "Decimal ID",
			urlPath:  "/snippet/view/1.23",
			wantCode: http.StatusNotFound,
		},
		{
			name:     "String ID",
			urlPath:  "/snippet/view/foo",
			wantCode: http.StatusNotFound,
		},
		{
			name:     "Empty ID",
			urlPath:  "/snippet/view/",
			wantCode: http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, _, body := server.get(t, tt.urlPath)
			assert.Equal(t, code, tt.wantCode)
			if tt.wantBody != "" {
				assert.StringContains(t, body, tt.wantBody)
			}
		})
	}
}

func TestUserSignup(t *testing.T) {
	app := newTestApplication(t)
	server := newTestServer(t, app.router())
	defer server.Close()
	_, _, body := server.get(t, "/user/signup")
	//从html页面检测出来的csrf token
	validCSRFToken := extractCSRFToken(t, body)
	//以和fmt.Printf()相同的方式运行，将信息输出在测试输出
	//t.Logf("CSRF token is: %q", validCSRFToken)

	const (
		validName     = "Bob"
		validPassword = "validPa$$word"
		validEmail    = "bob@example.com"
		formTag       = "<form action='/user/signup' method='POST' novalidate>"
	)
	tests := []struct {
		name         string
		userName     string
		userEmail    string
		userPassword string
		csrfToken    string
		wantCode     int
		wantFormTag  string
	}{
		{
			name:         "Valid submission",
			userName:     validName,
			userEmail:    validEmail,
			userPassword: validPassword,
			csrfToken:    validCSRFToken,
			wantCode:     http.StatusSeeOther,
		},
		{
			name:         "Invalid CSRF Token",
			userName:     validName,
			userEmail:    validEmail,
			userPassword: validPassword,
			csrfToken:    "wrongToken",
			wantCode:     http.StatusBadRequest,
		},
		{
			name:         "Empty name",
			userName:     "",
			userEmail:    validEmail,
			userPassword: validPassword,
			csrfToken:    validCSRFToken,
			wantCode:     http.StatusUnprocessableEntity,
			wantFormTag:  formTag,
		},
		{
			name:         "Empty email",
			userName:     validName,
			userEmail:    "",
			userPassword: validPassword,
			csrfToken:    validCSRFToken,
			wantCode:     http.StatusUnprocessableEntity,
			wantFormTag:  formTag,
		},
		{
			name:         "Empty password",
			userName:     validName,
			userEmail:    validEmail,
			userPassword: "",
			csrfToken:    validCSRFToken,
			wantCode:     http.StatusUnprocessableEntity,
			wantFormTag:  formTag,
		},
		{
			name:         "Invalid email",
			userName:     validName,
			userEmail:    "bob@example.",
			userPassword: validPassword,
			csrfToken:    validCSRFToken,
			wantCode:     http.StatusUnprocessableEntity,
			wantFormTag:  formTag,
		},
		{
			name:         "Short password",
			userName:     validName,
			userEmail:    validEmail,
			userPassword: "pa$$",
			csrfToken:    validCSRFToken,
			wantCode:     http.StatusUnprocessableEntity,
			wantFormTag:  formTag,
		},
		{
			name:         "Duplicate email",
			userName:     validName,
			userEmail:    "dupe@example.com",
			userPassword: validPassword,
			csrfToken:    validCSRFToken,
			wantCode:     http.StatusUnprocessableEntity,
			wantFormTag:  formTag,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			form := url.Values{}
			form.Add("name", test.userName)
			form.Add("email", test.userEmail)
			form.Add("password", test.userPassword)
			form.Add("csrf_token", test.csrfToken)
			code, _, body := server.postForm(t, "/user/signup", form)
			//得到响应的状态码和响应体
			assert.Equal(t, code, test.wantCode)
			if test.wantFormTag != "" {
				assert.StringContains(t, body, test.wantFormTag)
			}
		})
	}
}

func TestSnippetCreate(t *testing.T) {
	//未经身份验证的用户将被重定向到登录表单。
	//向经过身份验证的用户显示用于创建新代码段的表单

	//创建一个测试配置实例
	app := newTestApplication(t)
	//创建测试服务，并发起请求
	server := newTestServer(t, app.router())
	defer server.Close()

	//未认证，发送/snippet/create 会重定向到登录界面
	t.Run("Unauthenticated", func(t *testing.T) {
		code, header, _ := server.get(t, "/snippet/create")
		assert.Equal(t, code, http.StatusSeeOther)
		assert.Equal(t, header.Get("Location"), "/user/login")
	})
	t.Run("Authenticated", func(t *testing.T) {
		_, _, body := server.get(t, "/user/login")
		token := extractCSRFToken(t, body)

		//执行登录操作
		form := url.Values{}
		form.Add("email", "alice@example.com")
		form.Add("password", "pa$$word")
		form.Add("csrf_token", token)
		server.postForm(t, "/user/login", form) //执行登录操作后，检查context中是否有用户id

		code, _, body := server.get(t, "/snippet/create")

		assert.Equal(t, code, http.StatusOK)
		assert.StringContains(t, body, "<form action='/snippet/create' method='POST'>")
	})
}
