package utils

import (
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"tgwp/log/zlog"
	"time"
)

/*
GetRootPath 获取项目根目录。
优先尝试获取当前可执行文件所在的目录，如果失败则返回当前工作目录。
*/
func GetRootPath(myPath string) string {
	// 获取当前可执行文件的路径
	exePath, err := os.Executable()
	if err != nil {
		// 如果获取失败，回退到当前工作目录
		wd, _ := os.Getwd()
		return filepath.Join(wd, myPath)
	}

	// 对于 go run 运行的情况，可执行文件在临时目录，
	// 此时可以尝试使用工作目录，或者保留 runtime.Caller 作为开发环境的兜底。
	// 但为了部署稳定，通常建议部署时配置文件放在二进制同级目录。
	rootPath := filepath.Dir(exePath)

	// 检查是否在临时目录运行 (go run)
	// 包含 go-build 通常意味着是在 go run 的临时构建目录中
	if filepath.Base(rootPath) == "exe" || filepath.Base(rootPath) == "main" || strings.Contains(rootPath, "go-build") {
		wd, _ := os.Getwd()
		return filepath.Join(wd, myPath)
	}

	return filepath.Join(rootPath, myPath)
}

// StructToMap
//
//	@Description: struct to map
//	@param value
//	@return map[string]interface{}
func StructToMap(value interface{}) map[string]interface{} {
	m := make(map[string]interface{})
	resJson, err := json.Marshal(value)
	if err != nil {
		zlog.Errorf("Json Marshal failed ,msg: %s", err.Error())
		return nil
	}
	err = json.Unmarshal(resJson, &m)
	if err != nil {
		zlog.Errorf("Json Unmarshal failed,msg : %s", err.Error())
		return nil
	}
	return m
}

// StuctToJson
//
//	@Description: struct to json
//	@param value
//	@return string
//	@return error
func StuctToJson(value interface{}) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(data), err
}

// JsonToStruct
//
//	@Description: json to struct
//	@param str
//	@param value
//	@return error
func JsonToStruct(str string, value interface{}) error {
	return json.Unmarshal([]byte(str), value)
}

// RandomCode
//
//	@Description: 生成随机码
//	@return string
func RandomCode() string {
	const charset = "0123456789abcdefghijklmnopqrlstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	result := make([]byte, 6)
	rand.Seed(time.Now().UnixNano())
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

// IdentifyPhone
//
//	@Description: 判定是否为中国手机号
//	@param phone
//	@return bool
func IdentifyPhone(phone string) bool {
	var phoneRegex = regexp.MustCompile(`^1(3[0-9]|4[57]|5[0-35-9]|7[0-9]|8[0-9]|9[8])\d{8}$`)
	return phoneRegex.MatchString(phone)
}

// RecordTime a tool to record time
// e.g [defer utils.RecordTime(time.Now())()]
func RecordTime(start time.Time) func() {
	return func() {
		end := time.Now()
		zlog.Debugf("use time:%d", end.Unix()-start.Unix())
	}
}
