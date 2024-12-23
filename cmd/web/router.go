package main

import (
	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
	"net/http"
)

func (app *application) router() http.Handler {
	//初始化一个路由
	router := httprouter.New()
	//创建一个处理器包装notFound函数将他作为404 NotFound响应的自定义处理函数
	router.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.notFound(w)
	})
	//创建一个文件服务器，从指定目录中提供文件
	fileServer := http.FileServer(http.Dir("./ui/static"))
	//为所有以/static/开头的URL路径使用mux.Handle函数注册文件服务作为一个处理器
	//在到达文件服务前，我们去掉/static前缀
	router.Handler(http.MethodGet, "/static/*filepath", http.StripPrefix("/static", fileServer))

	//测试样例
	router.HandlerFunc(http.MethodGet, "/ping", ping)

	//为我们的动态路由创建一个中间件链包含指定的中间件
	//LoadAndSave中间件包装动态路由自动加载保存每个HTTP请求和响应的会话数据
	dynamic := alice.New(app.sessionManager.LoadAndSave, noSurf, app.authenticate)
	//dynamic.ThenFunc返回的是Handler类型，需要使用Handler函数注册路由
	//Unprotected application routes using the "dynamic" middleware chain
	router.Handler(http.MethodGet, "/", dynamic.ThenFunc(app.snippetViews))
	router.Handler(http.MethodGet, "/snippet/view/:id", dynamic.ThenFunc(app.snippetView))
	router.Handler(http.MethodGet, "/user/signup", dynamic.ThenFunc(app.userSignup))
	router.Handler(http.MethodPost, "/user/signup", dynamic.ThenFunc(app.userSignupPost))
	router.Handler(http.MethodGet, "/user/login", dynamic.ThenFunc(app.userLogin))
	router.Handler(http.MethodPost, "/user/login", dynamic.ThenFunc(app.userLoginPost))
	router.Handler(http.MethodGet, "/about", dynamic.ThenFunc(app.about))
	// Protected (authenticated-only) application routes, using a new "protected"
	// middleware chain which includes the requireAuthentication middleware.
	protected := dynamic.Append(app.requireAuthentication)
	router.Handler(http.MethodGet, "/snippet/create", protected.ThenFunc(app.snippetCreate))
	router.Handler(http.MethodPost, "/snippet/create", protected.ThenFunc(app.snippetCreatePost))
	router.Handler(http.MethodPost, "/user/logout", protected.ThenFunc(app.userLogoutPost))
	router.Handler(http.MethodGet, "/user/account", protected.ThenFunc(app.account))
	router.Handler(http.MethodGet, "/account/password/update", protected.ThenFunc(app.accountPasswordUpdate))
	router.Handler(http.MethodPost, "/account/password/update", protected.ThenFunc(app.accountPasswordUpdatePost))

	standard := alice.New(app.recoverPanic, app.logRequest, secureHandler)
	return standard.Then(router)
}
