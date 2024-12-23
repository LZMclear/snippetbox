package main

import (
	"errors"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"snippetbox/internal/models"
	"snippetbox/internal/validator"
	"strconv"
)

type snippetCreateForm struct { //用于接收文本页数据解析到此结构体
	Title               string     `form:"title"`
	Content             string     `form:"content"`
	Expires             int        `form:"expires"`
	validator.Validator `form:"-"` //-告诉decoder在解码期间忽略此字段
}
type userSignupForm struct { //用于接收用户数据解析到此结构体
	Name                string `form:"name"`
	Email               string `form:"email"`
	Password            string `form:"password"`
	validator.Validator `form:"-"`
}
type userLoginForm struct { //用于接收用户登录数据解析到此结构体
	Email               string `form:"email"`
	Password            string `form:"password"`
	validator.Validator `form:"-"`
}
type updatePasswordForm struct {
	CurrentPassword     string `form:"current_password"`
	NewPassword         string `form:"new_password"`
	ConfirmPassword     string `form:"confirm_password"`
	validator.Validator `form:"-"`
}

func ping(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	//因为httprouter精确匹配"/"，所有不需要下面代码
	//if r.URL.Path != "/" {
	//	app.notFound(w)
	//	return
	//}
	data := app.newTemplateData(r)
	app.render(w, http.StatusOK, "home.html", data)
}

// 接收来自用户的id查询字符串参数
func (app *application) snippetView(w http.ResponseWriter, r *http.Request) {
	//httprouter解析请求，将任何命名的参数存储在请求的上下文（context）中，可以使用ParamsFromContext()提取一个包含命名参数和值的切片
	params := httprouter.ParamsFromContext(r.Context())
	id, err := strconv.Atoi(params.ByName("id"))
	if err != nil {
		app.notFound(w)
		return
	}
	snippet, err := app.snippets.Get(id)
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			app.notFound(w)
		} else {
			app.serverError(w, err)
		}
		return
	}
	//使用PopString提取键为key的值，并且会将他们从会话数据中删除，只使用一次
	flash := app.sessionManager.PopString(r.Context(), "flash")
	data := app.newTemplateData(r)
	data.Snippet = snippet
	data.Flash = flash
	app.render(w, http.StatusOK, "view.html", data)
}

// 查询前十个最新的数据
func (app *application) snippetViews(w http.ResponseWriter, r *http.Request) {
	//w.Header().Set("Content-Type", "application/json")
	snippets, err := app.snippets.Latest()
	if err != nil {
		app.serverError(w, err)
		return
	}
	//创建页面数据模版
	data := app.newTemplateData(r)
	data.Snippets = snippets
	app.render(w, http.StatusOK, "home.html", data)
}

func (app *application) snippetCreate(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = snippetCreateForm{
		Expires: 365,
	}
	app.render(w, http.StatusOK, "create.html", data)
}

func (app *application) snippetCreatePost(w http.ResponseWriter, r *http.Request) {
	var form snippetCreateForm
	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}
	//验证数据格式是否正确
	//因为Validator内嵌入snippetCreateForm中，我们可以直接使用snippetCreateForm对象调用Validator的方法。
	form.CheckField(validator.NotBlank(form.Title), "title", "this field can not be blank")
	form.CheckField(validator.MaxChars(form.Title, 100), "title", "This field cannot be more than 100 characters long")
	form.CheckField(validator.NotBlank(form.Content), "content", "This field cannot be blank")
	form.CheckField(validator.PermittedValue(form.Expires, 1, 7, 365), "expires", "This field must equal 1, 7 or 365")

	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, http.StatusUnprocessableEntity, "create.html", data)
		return
	}
	id, err := app.snippets.Insert(form.Title, form.Content, form.Expires)
	if err != nil {
		app.serverError(w, err)
		return
	}
	//使用Put method创建键为flash的键值对
	app.sessionManager.Put(r.Context(), "flash", "Snippet successfully created!")
	// Redirect the user to the relevant page for the snippet.
	http.Redirect(w, r, fmt.Sprintf("/snippet/view/%d", id), http.StatusSeeOther)
}

func (app *application) userSignup(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = userSignupForm{}
	app.render(w, http.StatusOK, "signup.html", data)
}

func (app *application) userSignupPost(w http.ResponseWriter, r *http.Request) {
	//声明userSignupForm结构体的零值实例
	var form userSignupForm

	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
	}
	//解析后开始验证接收的数据是否正确
	form.CheckField(validator.NotBlank(form.Name), "name", "This field cannot be blank")
	form.CheckField(validator.NotBlank(form.Email), "email", "This field cannot be blank")
	form.CheckField(validator.Matches(form.Email, validator.EmailRX), "email", "This field must be a valid email address")
	form.CheckField(validator.NotBlank(form.Password), "password", "This field cannot be blank")
	form.CheckField(validator.MinChars(form.Password, 8), "password", "This field must be at least 8 characters long")
	//如果错误不为空，将表单数据重新发送到前端页面展示，并展示具体的错误
	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, http.StatusUnprocessableEntity, "signup.html", data)
		return
	}
	//数据格式无误后调用数据库插入数据记录
	err = app.users.Insert(form.Name, form.Email, form.Password)
	if err != nil {
		if errors.Is(err, models.ErrDuplicateEmail) {
			form.AddFieldError("email", "Email address is already in use")
			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, http.StatusUnprocessableEntity, "signup.html", data)
		} else {
			app.serverError(w, err)
		}
		return
	}
	app.sessionManager.Put(r.Context(), "flash", "Your signup was successful. Please log in.")
	http.Redirect(w, r, "/user/login", http.StatusSeeOther)
}

func (app *application) userLogin(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = userLoginForm{}
	app.render(w, http.StatusOK, "login.html", data)
}

func (app *application) userLoginPost(w http.ResponseWriter, r *http.Request) {
	//将表单数据解码到userLoginForm中
	var form userLoginForm
	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}
	//开始检查数据的合法性
	form.CheckField(validator.NotBlank(form.Email), "email", "This field cannot be blank")
	form.CheckField(validator.Matches(form.Email, validator.EmailRX), "email", "This field must be a valid email address")
	form.CheckField(validator.NotBlank(form.Password), "password", "This field cannot be blank")

	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, http.StatusUnprocessableEntity, "login.html", data)
		return
	}

	//检查凭证是否有效，无效则添加一个通用的无此字段错误消息。
	id, err := app.users.Authenticate(form.Email, form.Password)
	if err != nil {
		if errors.Is(err, models.ErrInvalidCredentials) {
			form.AddNonFieldError("Email or password is incorrect")
			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, http.StatusUnprocessableEntity, "login.html", data)
		} else {
			app.serverError(w, err)
		}
		return
	}
	//在当前会话中使用RenewToken更改sessionID，当身份验证状态或权限级别更改更新id是一个很好的实践
	err = app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, err)
		return
	}
	//在当前会话中添加用户id，实现用户登录状态
	app.sessionManager.Put(r.Context(), "authenticatedUserID", id)
	//检查用户会话数据是否含有路径，有取出来并删除他重定向到路由
	path := app.sessionManager.PopString(r.Context(), "origin_url")
	if path != "" {
		http.Redirect(w, r, path, http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/snippet/create", http.StatusSeeOther)
}

func (app *application) userLogoutPost(w http.ResponseWriter, r *http.Request) {
	//更改sessionID
	err := app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, err)
		return
	}
	//删除用户ID实现用户注销功能
	app.sessionManager.Remove(r.Context(), "authenticatedUserID")
	//添加一个flash信息确认用户已经注销登录
	app.sessionManager.Put(r.Context(), "flash", "You've been logged out successfully!")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// 返回about页面
func (app *application) about(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = "This is a about page..."
	app.render(w, http.StatusOK, "about.html", data)
}

// 个人账户信息
func (app *application) account(w http.ResponseWriter, r *http.Request) {
	//获取会话中的用户id  正常程序中，能够访问此handler一定是由认证过的页面发出的请求，所以会话中一定会有authenticatedID
	id := app.sessionManager.GetInt(r.Context(), "authenticatedUserID")
	user, err := app.users.Get(id)
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			http.Redirect(w, r, "/user/login", http.StatusSeeOther)
		} else {
			app.serverError(w, err)
		}
		return
	}
	data := app.newTemplateData(r)
	data.Form = user
	app.render(w, http.StatusOK, "account.html", data)
}

// 更新用户密码  返回用户更新密码界面
func (app *application) accountPasswordUpdate(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = updatePasswordForm{} // 将空值传递过去，什么也不显示
	app.render(w, http.StatusOK, "password_update.html", data)
}

func (app *application) accountPasswordUpdatePost(w http.ResponseWriter, r *http.Request) {
	//解析传递的数据
	var form updatePasswordForm
	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}
	//解析完数据开始检查数据的合法性
	form.CheckField(validator.NotBlank(form.CurrentPassword), "currentPassword", "当前密码不能为空")
	form.CheckField(validator.MinChars(form.NewPassword, 8), "newPassword", "密码至少为8个字符")
	form.CheckField(validator.NotBlank(form.ConfirmPassword), "confirmPassword", "当前密码不能为空")
	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, http.StatusUnprocessableEntity, "password_update.html", data)
		return
	}

	//检测完字段的合法性，开始检查非字段错误 （密码是否正确）
	id := app.sessionManager.GetInt(r.Context(), "authenticatedUserID")
	err = app.users.PasswordUpdate(id, form.CurrentPassword, form.NewPassword)
	if err != nil {
		if errors.Is(err, models.ErrInvalidCredentials) {
			form.AddNonFieldError("密码输入错误")
			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, http.StatusUnprocessableEntity, "password_update.html", data)
		} else {
			app.serverError(w, err)
		}
	}
	//更新成功重定向到用户界面
	http.Redirect(w, r, "/user/account", http.StatusSeeOther)
}
