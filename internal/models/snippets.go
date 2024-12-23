package models

import (
	"database/sql"
	"errors"
	"time"
)

type Snippet struct {
	ID      int
	Title   string
	Content string
	Created time.Time
	Expires time.Time
}

type SnippetModelInterface interface {
	Insert(title string, content string, expires int) (int, error)
	Get(id int) (*Snippet, error)
	Latest() ([]*Snippet, error)
}

// SnippetModel 定义一个SnippetModel包装一个sql.DB连接池
type SnippetModel struct {
	DB *sql.DB
}

// Insert will insert a new snippet into the database.
func (m *SnippetModel) Insert(title string, content string, expires int) (int, error) {
	sqlStr := "insert into snippets (title,content,created,expires)values (?,?,UTC_TIMESTAMP(), DATE_ADD(UTC_TIMESTAMP(), INTERVAL ? DAY))"
	result, err := m.DB.Exec(sqlStr, title, content, expires)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

// Get will return a specific snippet based on its id.
func (m *SnippetModel) Get(id int) (*Snippet, error) {
	//查询是顺便对日期进行检查，过期不要
	sqlStr := "select * from snippets where expires > UTC_TIMESTAMP() AND id = ?"
	queryRow := m.DB.QueryRow(sqlStr, id)
	s := &Snippet{}
	//使用row.Scan从sql.row复制值到结构体的每个字段中去 注意！row.Scan()中的参数为指针类型，并且参数数量必须与查询到的字段数量一致
	err := queryRow.Scan(&s.ID, &s.Title, &s.Content, &s.Created, &s.Expires)
	if err != nil {
		//如果没有查询到数据，会返回一个sql.ErrNoRows
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoRecord
		} else {
			return nil, err
		}

	}
	return s, nil
}

// Latest will return the 10 most recently created snippets.
func (m *SnippetModel) Latest() ([]*Snippet, error) {
	sqlStr := "select *from snippets where expires>UTC_TIMESTAMP() order by id desc limit 10"
	rows, err := m.DB.Query(sqlStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	snippets := make([]*Snippet, 0) //创建一个长度为0的切片，相当于初始化 用来存放数据
	//遍历结果集中的行数据，迭代完毕自动关闭底层数据库连接
	for rows.Next() {
		s := &Snippet{}
		err := rows.Scan(&s.ID, &s.Title, &s.Content, &s.Created, &s.Expires)
		if err != nil {
			return nil, err
		}
		snippets = append(snippets, s)
	}
	//当rows.Next()运行完毕，检查遍历过程中遇到的所有错误 不要假设遍历整个结果集没有任何错误
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return snippets, nil
}
