package mysqldb

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	yaml "gopkg.in/yaml.v2"
)

// User xxx
type User struct {
	Id     uint32 `json:"id"`
	Name   string `json:"name"`
	CorpId uint32 `json:"corp_id"` // 公司id
}

// Corp xxx
type Corp struct {
	Id   uint32 `json:"id"`
	Name string `json:"name"`
}

// TestMain xxx
func TestMain(m *testing.M) {
	data, err := ioutil.ReadFile("../confiy.yaml")
	if err != nil {
		fmt.Println(err.Error())
	}
	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		fmt.Println(err.Error())
	}

	// 使用事务，推荐使用Manager全局初始化的方式
	err = Init(cfg)
	if err != nil {
		fmt.Println(err.Error())
	}

	os.Exit(m.Run())
}

// TestTransactionManager_QueryContext
// NoTx
func TestTransactionManager_QueryContext(t *testing.T) {
	rows, err := GlobalManager().QueryContext(context.Background(), "SELECT name FROM user WHERE corp_id = ?", 1)
	if err != nil {
		t.Fatal(err.Error())
	}
	var names []string
	for rows.Next() {
		var tmp string
		_ = rows.Scan(&tmp)
		names = append(names, tmp)
	}
	fmt.Println(names)
}

// TestExecTransaction_A xxx
func TestExecTransaction_A(t *testing.T) {
	var corp = Corp{
		Name: "A",
	}
	var user = User{
		Name: "test",
	}

	err := Transaction(context.Background(), GlobalManager(), func(ctx context.Context) error {
		var corpId int64
		err := GlobalManager().QueryRowContext(ctx, "SELECT id FROM corp WHERE name=?", corp.Name).Scan(&corpId)

		// 不存在插入一条Corp数据
		if err == sql.ErrNoRows {
			result, err := GlobalManager().ExecContext(ctx, "INSERT INTO corp(name) VALUES(?)", corp.Name)
			if err != nil {
				return err
			}

			corpId, err = result.LastInsertId()

			if err != nil {
				return err
			}
		}

		// 插入一条User数据
		_, err = GlobalManager().ExecContext(ctx, "INSERT INTO user(name, corp_id) VALUES(?,?)", user.Name, corpId)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		t.Fatal(err)
	}
}

// TestExecTransaction_B xxx
func TestExecTransaction_B(t *testing.T) {
	var corp = Corp{
		Name: "B",
	}
	var user = User{
		Name: "testB",
	}

	// 外部创建db事务对象
	tx, err := GlobalManager().WDB().Begin()
	if err != nil {
		t.Fatal(err)
	}
	ctx := NewTrans(context.Background(), tx)

	err = Transaction(ctx, GlobalManager(), func(ctx context.Context) error {
		var corpId int64
		err := GlobalManager().QueryRowContext(ctx, "SELECT id FROM corp WHERE name=?", corp.Name).Scan(&corpId)

		// 不存在插入一条Corp数据
		if err == sql.ErrNoRows {
			result, err := GlobalManager().ExecContext(ctx, "INSERT INTO corp(name) VALUES(?)", corp.Name)
			if err != nil {
				return err
			}

			corpId, err = result.LastInsertId()

			if err != nil {
				return err
			}
		}

		// 插入一条User数据
		_, err = GlobalManager().ExecContext(ctx, "INSERT INTO user(name, corp_id) VALUES(?,?)", user.Name, corpId)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		t.Fatal(err)
	}
}

// TestTransactionManager_StmtContext xxx
func TestTransactionManager_StmtContext(t *testing.T) {
	ctx := context.Background()
	// No tx
	stmt, err := GlobalManager().PrepareContext(ctx, "INSERT INTO corp(name) VALUES(?)")
	if err != nil {
		t.Fatalf("Stmt, err = %v, %v", stmt, err)
	}

	defer stmt.Close()

	tx, err := GlobalManager().writeDB.Begin()
	if err != nil {
		t.Fatal(err)
	}

	// With tx
	ctx = NewTrans(ctx, tx)
	err = Transaction(ctx, GlobalManager(), func(ctx context.Context) error {
		// stmt1
		result, err := GlobalManager().Stmt(ctx, stmt).Exec("testCorp111")
		if err != nil {
			return err
		}

		corpId, err := result.LastInsertId()
		if err != nil {
			return err
		}
		fmt.Println(corpId)

		// stmt2
		result, err = GlobalManager().Stmt(ctx, stmt).Exec("testCorp222")
		if err != nil {
			return err
		}

		corpId, err = result.LastInsertId()
		if err != nil {
			return err
		}
		fmt.Println(corpId)

		return nil
	})

	if err != nil {
		t.Fatal(err)
	}
}
