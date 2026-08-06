package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/llr104/slgserver/config"
	"xorm.io/xorm"
	"xorm.io/xorm/log"
)

var MasterDB *xorm.Engine

// TestDB giữ tên hàm cũ để không làm thay đổi các service đang gọi nó.
// Khi DATABASE_URL tồn tại, hệ thống kết nối trực tiếp PostgreSQL của Supabase.
func TestDB() error {
	return Init()
}

func Init() error {
	if databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL")); databaseURL != "" {
		return initEngine("postgres", databaseURL)
	}

	mysqlConfig, err := config.File.GetSection("mysql")
	if err != nil {
		return fmt.Errorf("không đọc được cấu hình MySQL: %w", err)
	}

	return initEngine("mysql", fillMySQLDSN(mysqlConfig))
}

func fillMySQLDSN(mysqlConfig map[string]string) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=%s&parseTime=True&loc=Local",
		mysqlConfig["user"],
		mysqlConfig["password"],
		mysqlConfig["host"],
		mysqlConfig["port"],
		mysqlConfig["dbname"],
		mysqlConfig["charset"])
}

func initEngine(driver, dataSourceName string) error {
	engine, err := xorm.NewEngine(driver, dataSourceName)
	if err != nil {
		return fmt.Errorf("không thể tạo kết nối %s: %w", driver, err)
	}

	engine.SetMaxIdleConns(envInt("DB_MAX_IDLE_CONNS", config.File.MustInt("mysql", "max_idle", 2)))
	engine.SetMaxOpenConns(envInt("DB_MAX_OPEN_CONNS", config.File.MustInt("mysql", "max_conn", 10)))

	showSQL := envBool("XORM_SHOW_SQL", config.File.MustBool("xorm", "show_sql", false))
	logLevel := envInt("XORM_LOG_LEVEL", config.File.MustInt("xorm", "log_level", 1))
	logFile := strings.TrimSpace(os.Getenv("XORM_LOG_FILE"))
	if logFile == "" {
		logFile = config.File.MustValue("xorm", "log_file", "")
	}

	if logFile != "" {
		if dir := filepath.Dir(logFile); dir != "." {
			_ = os.MkdirAll(dir, 0o755)
		}
		if file, fileErr := os.Create(logFile); fileErr == nil {
			engine.SetLogger(log.NewSimpleLogger(file))
		} else {
			fmt.Println("Không thể mở tệp nhật ký SQL:", fileErr)
		}
	}

	engine.SetLogLevel(log.LogLevel(logLevel))
	engine.ShowSQL(showSQL)

	if err = engine.Ping(); err != nil {
		_ = engine.Close()
		return fmt.Errorf("không thể kết nối %s: %w", driver, err)
	}

	if MasterDB != nil {
		_ = MasterDB.Close()
	}
	MasterDB = engine

	fmt.Printf("Đã kết nối cơ sở dữ liệu %s thành công.\n", driver)
	return nil
}

func envInt(name string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		fmt.Printf("Biến %s không hợp lệ, dùng giá trị mặc định %d.\n", name, fallback)
		return fallback
	}
	return parsed
}

func envBool(name string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		fmt.Printf("Biến %s không hợp lệ, dùng giá trị mặc định %t.\n", name, fallback)
		return fallback
	}
	return parsed
}

func StdMasterDB() *sql.DB {
	if MasterDB == nil {
		return nil
	}
	return MasterDB.DB().DB
}
