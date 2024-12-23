package main

import (
	"bytes"
	"github.com/alexedwards/scs/v2"
	"github.com/go-playground/form/v4"
	"html"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"regexp"
	"snippetbox/internal/models/mocks"
	"testing"
	"time"
)

// 创建一个newTestApplication helper 此函数返回一个application实例包含虚拟依赖
func newTestApplication(t *testing.T) *application {
	templateCache, err := newTemplateCache()
	if err != nil {
		t.Fatal(err)
	}
	//表单解码器
	decoder := form.NewDecoder()
	//会话管理实例，我们使用和生产环境相同的设置，除了没有为会话管理设置存储。如果没有设置存储，scs包会自动默认使用in-memory存储。
	//这对于测试目的是理想的
	sessionManager := scs.New()
	sessionManager.Lifetime = 12 * time.Hour
	sessionManager.Cookie.Secure = true
	return &application{
		infoLog:        log.New(io.Discard, "", 0),
		errorLog:       log.New(io.Discard, "", 0),
		snippets:       &mocks.SnippetModel{}, // Use the mocks.
		users:          &mocks.UserModel{},    // Use the mocks.
		templateCache:  templateCache,
		formDecoder:    decoder,
		sessionManager: sessionManager,
	}
}

// 定义一个testServer类型嵌入一个httptest.Server实例
type testServer struct {
	*httptest.Server
}

func newTestServer(t *testing.T, h http.Handler) *testServer {
	ts := httptest.NewTLSServer(h)

	//初始化一个cookie jar
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	//测试服务客户端添加cookie jar 任何响应cookie将会被存储在这里，并且随着之后的请求一起发送
	ts.Client().Jar = jar

	//禁用测试服务器客户端的重定向跟踪，这个函数会被调用当客户端接受一个3xxx响应
	//总是返回一个http.ErrUseLastResponse错误使客户端集中返回接收的响应
	ts.Client().CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &testServer{ts}
}

func (ts *testServer) get(t *testing.T, urlPath string) (int, http.Header, string) {
	resp, err := ts.Client().Get(ts.URL + urlPath)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	body = bytes.TrimSpace(body)
	return resp.StatusCode, resp.Header, string(body)
}

// 从html响应体提取CSRF token
var csrfTokenRX = regexp.MustCompile(`<input type='hidden' name='csrf_token' value='(.+)'>`)

func extractCSRFToken(t *testing.T, body string) string {
	// Use the FindStringSubmatch method to extract the token from the HTML body.
	// Note that this returns an array with the entire matched pattern in the
	// first position, and the values of any captured data in the subsequent
	// positions.
	matches := csrfTokenRX.FindStringSubmatch(body)
	if len(matches) < 2 {
		t.Fatal("no csrf token found in body")
	}
	return html.UnescapeString(string(matches[1]))
}

// 用来向服务发送带有特定数据的post请求
func (ts *testServer) postForm(t *testing.T, urlPath string, form url.Values) (int, http.Header, string) {
	resp, err := ts.Client().PostForm(ts.URL+urlPath, form)
	if err != nil {
		t.Fatal(err)
	}
	//从响应体读取数据
	body, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	bytes.TrimSpace(body)
	// Return the response status, headers and body.
	return resp.StatusCode, resp.Header, string(body)
}
