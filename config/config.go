package config

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/Unknwon/goconfig"
)

var (
	File *goconfig.ConfigFile
	ROOT string
)

const mainIniPath = "/data/conf/env.ini"

type envBinding struct {
	env     string
	section string
	key     string
}

func init() {
	curDir, _ := os.Getwd()
	ROOT = curDir

	configPath := ROOT + mainIniPath
	file, err := goconfig.LoadConfigFile(configPath)
	File = file

	if err != nil {
		fmt.Println("Không thể tải tệp cấu hình:", err)
		File, _ = goconfig.LoadFromData([]byte(""))
	}

	if err = loadIncludeFiles(); err != nil {
		panic("Không thể tải tệp cấu hình bổ sung: " + err.Error())
	}

	applyEnvironment()
	go signalReload()
}

// ListenAddress trả về địa chỉ lắng nghe của một dịch vụ.
// Biến môi trường của dịch vụ được ưu tiên, sau đó đến HOST/PORT của Railway,
// cuối cùng mới dùng giá trị trong data/conf/env.ini.
func ListenAddress(section, defaultPort string) string {
	prefix := serviceEnvPrefix(section)
	host := firstNonEmpty(
		os.Getenv(prefix+"_HOST"),
		os.Getenv("HOST"),
		File.MustValue(section, "host", ""),
	)
	port := firstNonEmpty(
		os.Getenv(prefix+"_PORT"),
		os.Getenv("PORT"),
		File.MustValue(section, "port", defaultPort),
	)

	return net.JoinHostPort(host, port)
}

func ReloadConfigFile() {
	var err error
	configPath := ROOT + mainIniPath
	File, err = goconfig.LoadConfigFile(configPath)
	if err != nil {
		fmt.Println("Không thể tải lại tệp cấu hình:", err)
		return
	}

	if err = loadIncludeFiles(); err != nil {
		fmt.Println("Không thể tải lại tệp cấu hình bổ sung:", err)
		return
	}

	applyEnvironment()
	fmt.Println("Đã tải lại cấu hình thành công.")
}

func SaveConfigFile() error {
	err := goconfig.SaveConfigFile(File, ROOT+mainIniPath)
	if err != nil {
		fmt.Println("Không thể lưu tệp cấu hình:", err)
		return err
	}

	fmt.Println("Đã lưu cấu hình thành công.")
	return nil
}

func loadIncludeFiles() error {
	includeFile := File.MustValue("include_files", "path", "")
	if includeFile != "" {
		includeFiles := strings.Split(includeFile, ",")
		return File.AppendFiles(includeFiles...)
	}

	return nil
}

func applyEnvironment() {
	bindings := []envBinding{
		{env: "GATE_NEED_SECRET", section: "gateserver", key: "need_secret"},
		{env: "SLG_PROXY_URL", section: "gateserver", key: "slg_proxy"},
		{env: "CHAT_PROXY_URL", section: "gateserver", key: "chat_proxy"},
		{env: "LOGIN_PROXY_URL", section: "gateserver", key: "login_proxy"},
		{env: "SLG_NEED_SECRET", section: "slgserver", key: "need_secret"},
		{env: "SLG_IS_DEV", section: "slgserver", key: "is_dev"},
		{env: "CHAT_NEED_SECRET", section: "chatserver", key: "need_secret"},
		{env: "LOGIN_NEED_SECRET", section: "loginserver", key: "need_secret"},
		{env: "XORM_SHOW_SQL", section: "xorm", key: "show_sql"},
		{env: "XORM_LOG_LEVEL", section: "xorm", key: "log_level"},
		{env: "XORM_LOG_FILE", section: "xorm", key: "log_file"},
		{env: "LOG_DIR", section: "log", key: "file_dir"},
		{env: "MAP_DATA_PATH", section: "logic", key: "map_data"},
		{env: "JSON_DATA_PATH", section: "logic", key: "json_data"},
		{env: "GAME_SERVER_ID", section: "logic", key: "server_id"},
	}

	for _, binding := range bindings {
		if value := strings.TrimSpace(os.Getenv(binding.env)); value != "" {
			File.SetValue(binding.section, binding.key, value)
		}
	}
}

func serviceEnvPrefix(section string) string {
	switch section {
	case "httpserver":
		return "HTTP"
	case "gateserver":
		return "GATE"
	case "loginserver":
		return "LOGIN"
	case "chatserver":
		return "CHAT"
	case "slgserver":
		return "SLG"
	default:
		return strings.ToUpper(section)
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

// fileExist kiểm tra tệp hoặc thư mục có tồn tại hay không.
func fileExist(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || os.IsExist(err)
}
