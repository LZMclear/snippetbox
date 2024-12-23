package main

import (
	"context"
	"fmt"
	"github.com/justinas/nosurf"
	"net/http"
)

func secureHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//为了便于阅读，分成多行
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; style-src 'self' fonts.googleapis.com; font-src fonts.gstatic.com")
		w.Header().Set("Referrer-Policy", "origin-when-cross-origin")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "deny")
		w.Header().Set("X-XSS-Protection", "0")
		next.ServeHTTP(w, r)
	})
}

// 记录请求信息
func (app *application) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		app.infoLog.Printf("%s - %s %s %s", request.RemoteAddr, request.Proto, request.Method, request.URL.RequestURI())
		next.ServeHTTP(writer, request)
	})
}

// 当请求中触发panic时，因为每个请求都是在单独的goroutine中执行的，所以只会关闭对应的线程，不会影响到主线程。并且返回的为空response
// 实践是返回的panic错误？？？
// 创建中间件恢复panic并调用app.serverError方法，可以利用defer在panic展开堆栈 时总是被调用的事实
func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		defer func() {
			//使用内置包检查是否有panic
			if err := recover(); err != nil {
				writer.Header().Set("Connection", "close")
				app.serverError(writer, fmt.Errorf("err: %s", err))
			}
		}()
		next.ServeHTTP(writer, request)
	})
}

func (app *application) requireAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//如果用户未登录,将他重定向到登录界面  并将请求路由保存在会话数据中
		if !app.isAuthenticated(r) {
			app.sessionManager.Put(r.Context(), "origin_url", r.URL.Path)
			http.Redirect(w, r, "/user/login", http.StatusSeeOther)
			return
		}
		//如果登录的话,设置Cache-Control: no-store头要求认证不存储在用户浏览器缓存中
		w.Header().Add("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

// 创建一个中间件，使用带有Secure，Path，HttpOnly属性集的自定义CSRF cookie
func noSurf(next http.Handler) http.Handler {
	csrfHandler := nosurf.New(next)
	csrfHandler.SetBaseCookie(http.Cookie{
		HttpOnly: true,
		Path:     "/",
		Secure:   true,
	})
	return csrfHandler
}

func (app *application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//从会话中提取authenticatedUserID的值 如果id为零，说明没有登录，不做处理
		id := app.sessionManager.GetInt(r.Context(), "authenticatedUserID")
		if id == 0 {
			next.ServeHTTP(w, r)
			return
		}
		//检测到id时，查询数据库是否存在这个用户
		exists, err := app.users.Exists(id)
		if err != nil {
			app.serverError(w, err)
			return
		}
		//请求来自认证过的用户，创建上下文的副本包含新的键值对
		if exists {
			ctx := context.WithValue(r.Context(), isAuthenticatedContextKey, true)
			//将副本包含在请求中
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}
