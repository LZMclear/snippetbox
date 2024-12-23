package main

import (
	"html/template"
	"path/filepath"
	"snippetbox/internal/models"
	"time"
)

// 用于返回的模版数据类型
type templateData struct {
	CurrentYear     int
	Snippet         *models.Snippet   //文本对象
	Snippets        []*models.Snippet //文本集
	Form            any
	Flash           string //用于在前端显示一次性数据
	IsAuthenticated bool
	CSRFToken       string
}

func newTemplateCache() (map[string]*template.Template, error) {
	//初始化map集合
	cache := map[string]*template.Template{}
	//使用filepath.Global获得一个所有匹配一个指定路径模式的路径切片  //测试里面解析不出来  !!!原因测试环境和生产环境相对路径
	pages, err := filepath.Glob("D:/Goland/GoWorks/src/snippetbox/ui/html/pages/*.html")
	if err != nil {
		return nil, err
	}
	//循环遍历文件路径
	for _, page := range pages {
		//从路径中提取文件名称
		name := filepath.Base(page) //name是文件的名称
		//创建的自定义模版函数必须在解析模版文件前注册，也就是说在调用解析模版函数ParseFiles前需要使用template.New()创建一个空的模版集合
		ts, err := template.New(name).Funcs(functions).ParseFiles("D:/Goland/GoWorks/src/snippetbox/ui/html/base.html")
		if err != nil {
			return nil, err
		}
		//partials文件夹下都每个页面必须的，所以解析该文件夹下全部的文件
		ts, err = ts.ParseGlob("D:/Goland/GoWorks/src/snippetbox/ui/html/partials/*.html")
		if err != nil {
			return nil, err
		}
		//最后再解析目标模版
		ts, err = ts.ParseFiles(page)
		if err != nil {
			return nil, err
		}
		//将模版集添加到缓存中
		cache[name] = ts
	}
	return cache, nil
}

// 自定义模版函数
func humanDate(time time.Time) string {
	return time.Format("02 Jan 2006 at 15:04")
}

// 创建一个template.FuncMap对象包含自定义 humanDate（） 函数
var functions = template.FuncMap{
	"humanDate": humanDate,
}
