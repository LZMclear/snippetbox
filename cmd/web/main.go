package main

import (
	"database/sql"
	"flag"
	"github.com/alexedwards/scs/mysqlstore"
	"github.com/alexedwards/scs/v2"
	"github.com/go-playground/form/v4"
	_ "github.com/go-sql-driver/mysql"
	"html/template"
	"log"
	"net/http"
	"os"
	"snippetbox/internal/models"
	"time"
)

type application struct {
	errorLog       *log.Logger                  //错误日志记录器
	infoLog        *log.Logger                  //正常日志信息打印记录器
	snippets       models.SnippetModelInterface //SnippetModel实现了关于Snippet增删改查的所有方法
	users          models.UserModelInterface
	templateCache  map[string]*template.Template
	formDecoder    *form.Decoder //自定义表单数据的解码器
	sessionManager *scs.SessionManager
	debug          bool //debug模式开关
}

func main() {
	addr := flag.String("addr", ":4000", "port address") //注意：返回的是一个字符串指针
	dsn := flag.String("dsn", "root:251210@tcp(127.0.0.1:3306)/snippetbox?charset=utf8mb4&parseTime=true", "MYSQL data source name")
	debug := flag.Bool("debug", false, "调试")
	//定义标志后解析
	flag.Parse()

	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)

	errorLog := log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)
	db, err := openDB(*dsn)
	if err != nil {
		errorLog.Fatal(err)
	}
	defer db.Close()
	// Initialize a new template cache...
	templateCache, err := newTemplateCache()
	if err != nil {
		errorLog.Fatal(err)
	}
	//初始化解码器实例
	formDecoder := form.NewDecoder()
	//使用scs.New()初始化会话管理器实例，然后配置使用Mysql作为会话存储
	sessionManager := scs.New()
	sessionManager.Store = mysqlstore.New(db)
	sessionManager.Lifetime = 12 * time.Hour
	//确保我们的session cookies设置了Secure属性
	//设置此选项意味着只有在使用HTTPS连接时，cookie才会由用户的web浏览器发送
	sessionManager.Cookie.Secure = true

	app := &application{
		errorLog:       errorLog,
		infoLog:        infoLog,
		snippets:       &models.SnippetModel{DB: db},
		users:          &models.UserModel{DB: db},
		templateCache:  templateCache,
		formDecoder:    formDecoder,
		sessionManager: sessionManager,
		debug:          *debug,
	}
	mux := app.router()
	//初始化一个http.Server结构包含配置设置
	svr := &http.Server{
		Addr:     *addr,
		ErrorLog: errorLog,
		//应用于所有请求
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		Handler:      mux,
	}
	infoLog.Printf("Starting server on %s", *addr)
	err = svr.ListenAndServeTLS("./tls/cert.pem", "./tls/key.pem")
	errorLog.Fatal(err)
}

// returns a sql.DB connection pool
func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}
