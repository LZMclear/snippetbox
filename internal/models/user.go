package models

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
	"strings"
	"time"
)

type User struct {
	ID             int
	Name           string
	Email          string
	HashedPassword []byte
	Created        time.Time
}

type UserModelInterface interface {
	Insert(name, email, password string) error
	Authenticate(email, password string) (int, error)
	Exists(id int) (bool, error)
	Get(id int) (*User, error)
	PasswordUpdate(id int, currentPassword, newPassword string) error
}

// UserModel 包装数据库连接池
type UserModel struct {
	DB *sql.DB
}

func (m *UserModel) Insert(name, email, password string) error {
	//为传进来的密码创建一个加密哈希值
	hashPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return err
	}
	stmt := `INSERT INTO users (name, email, hashed_password, created)
    VALUES(?, ?, ?, UTC_TIMESTAMP())`
	_, err = m.DB.Exec(stmt, name, email, string(hashPassword))
	if err != nil {
		//查看错误是否具有*mysql.MYSQLError
		var mySQLError *mysql.MySQLError
		if errors.As(err, &mySQLError) {
			if mySQLError.Number == 1062 && strings.Contains(mySQLError.Message, "users_uc_email") {
				return ErrDuplicateEmail
			}
		}
		return err
	}
	return nil
}

func (m *UserModel) Authenticate(email, password string) (int, error) {
	//通过给定的email提取出id和密码
	var id int
	var hashedPassword []byte
	stmt := "SELECT id, hashed_password FROM users WHERE email = ?"
	err := m.DB.QueryRow(stmt, email).Scan(&id, &hashedPassword)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrInvalidCredentials
		} else {
			return 0, err
		}
	}
	//检查哈希密码和提供的纯文本密码是否匹配
	err = bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return 0, ErrInvalidCredentials
		} else {
			return 0, err
		}
	}
	//password正确，返回用户id
	return id, err
}

// Exists 使用指定的ID检查用户是否存在
func (m *UserModel) Exists(id int) (bool, error) {
	var exists bool
	stmt := "SELECT EXISTS(SELECT true FROM users WHERE id = ?)"
	err := m.DB.QueryRow(stmt, id).Scan(&exists)
	return exists, err
}

func (m *UserModel) Get(id int) (*User, error) {
	user := User{} //先实例化一个用户对象
	stmt := "select name,email,created from users where id =?"
	err := m.DB.QueryRow(stmt, id).Scan(&user.Name, &user.Email, &user.Created)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoRecord
		} else {
			return nil, err
		}
	}
	return &user, nil
}

// PasswordUpdate 根据用户id查询对应的用户hash密码，将传进来的密码与数据库hash密码匹配，
// 相同对密码进行hash处理，更新数据库，不想同返回错误
func (m *UserModel) PasswordUpdate(id int, currentPassword, newPassword string) error {
	var hashedPassword []byte
	//根据id查询密码
	stmt := "select hashed_password from users where id=?"
	err := m.DB.QueryRow(stmt, id).Scan(&hashedPassword)
	if err != nil {
		//如果没有此记录返回自定义错误，不是返回err
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNoRecord
		} else {
			return err
		}
	}
	//匹配密码
	err = bcrypt.CompareHashAndPassword(hashedPassword, []byte(currentPassword))
	if err != nil { //不为空则匹配失败 查看错误类型
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return ErrInvalidCredentials //密码错误
		} else {
			return err //其他错误
		}
	}
	//更新hashedPassword
	updateStmt := "update users set hashed_password =? where id=?"
	password, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		fmt.Println("hash 化密码错误")
		return err
	}
	_, err = m.DB.Exec(updateStmt, password, id)
	if err != nil {
		return err
	}
	return nil
}
