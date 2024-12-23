package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/go-playground/form/v4"
	"github.com/justinas/nosurf"
	"net/http"
	"runtime/debug"
	"time"
)

// 写一个错误信息并对错误记录器errLog堆栈跟踪
func (app *application) serverError(w http.ResponseWriter, err error) {
	//使用debug.Stack函数获取协程的堆栈跟踪，将其添加到日志信息，当尝试调试错误时，能够通过堆栈跟踪查看程序的执行路径
	trace := fmt.Sprintf("%s\n%s", err.Error(), debug.Stack())
	app.errorLog.Output(2, trace)
	if app.debug { //为真 将错误写入返回的页面中
		http.Error(w, trace, http.StatusInternalServerError)
	}
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

}

// 发送一个特定的状态码和相关的描述，我们接下来会发送诸如400"Bad Request"当用户发送有问题的请求时
func (app *application) clientError(w http.ResponseWriter, status int) {
	//http.StatusText 根据状态码返回一个文本描述
	http.Error(w, http.StatusText(status), status)
}

// 为了保持一致性，我们同样实现了一个Not Found
func (app *application) notFound(w http.ResponseWriter) {
	app.clientError(w, http.StatusNotFound)
}

// 解决解析执行模版代码重复问题,从缓存中执行模版
func (app *application) render(w http.ResponseWriter, status int, page string, data *templateData) {
	//从模版缓存中根据要执行的名称中提取模版
	ts, ok := app.templateCache[page]
	if !ok {
		err := fmt.Errorf("the template %s does not exist", page)
		app.serverError(w, err)
		return
	}
	//初始化一个新的buffer
	buf := new(bytes.Buffer)

	//将模版写入buffer中，而不是直接写入w中          为什么都用base命名？执行base模版，
	err := ts.ExecuteTemplate(buf, "base", data)

	if err != nil {
		app.serverError(w, err)
		return
	}
	w.WriteHeader(status)
	buf.WriteTo(w)
}

func (app *application) newTemplateData(r *http.Request) *templateData {
	return &templateData{
		CurrentYear:     time.Now().Year(),
		Flash:           app.sessionManager.PopString(r.Context(), "flash"),
		IsAuthenticated: app.isAuthenticated(r),
		CSRFToken:       nosurf.Token(r),
	}
}

// 帮助解析表单数据，处理常见的错误  第二个参数是要存储数据的目标结构体地址
func (app *application) decodePostForm(r *http.Request, dst any) error {
	//先调用r.ParseForm()将请求中的表单数据存储到r.PostForm map中
	err := r.ParseForm()
	if err != nil {
		return err
	}
	err = app.formDecoder.Decode(dst, r.PostForm)
	if err != nil {
		var invalidDecoderError *form.InvalidDecoderError
		//如果我们使用无效的目标地址，会返回*form.InvalidDecoderError
		if errors.As(err, &invalidDecoderError) { //为什么已经是地址类型了还要取地址
			panic(err)
		}
		return err
	}
	return nil
}

// 返回身份验证状态
func (app *application) isAuthenticated(r *http.Request) bool {
	isAuthenticated, ok := r.Context().Value(isAuthenticatedContextKey).(bool)
	if !ok {
		return false
	}
	return isAuthenticated
}
