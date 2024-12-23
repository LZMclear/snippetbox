package models

import (
	"database/sql"
	"os"
	"testing"
)

func newTestDB(t *testing.T) *sql.DB {
	//建立一个sql.DB连接池
	//因为我们的sql脚本包含多个sql statements 需要在DSN使用multiStatements=true参数。
	db, err := sql.Open("mysql", "root:251210@tcp(127.0.0.1:3306)/test_snippetbox?parseTime=true&multiStatements=true")
	if err != nil {
		t.Fatal(err)
	}
	//读取sql脚本并执行语句
	script, err := os.ReadFile("./testdata/setup.sql")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(string(script))
	if err != nil {
		t.Fatal(err)
	}
	//使用t.Cleanup注册一个函数,当当前调用newTestDB的测试已经执行完毕时会自动被Go调用
	t.Cleanup(func() {
		script, err := os.ReadFile("./testdata/teardown.sql")
		if err != nil {
			t.Fatal(err)
		}
		_, err = db.Exec(string(script))
		if err != nil {
			t.Fatal(err)
		}
		db.Close()
	})
	return db
}
