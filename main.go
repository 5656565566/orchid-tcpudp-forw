package main

import (
	"log"
	"net/http"
	"os"

	"main/api"
	"main/config"
)

var (
	version    = "1.2.1"
	apiPort    string
	authCode   string
	configPath string
)

func main() {
	config.ParseArgs(&apiPort, &authCode, &configPath, version)
	api.SetAuthCode(authCode)
	api.SetConfigPath(configPath)

	http.HandleFunc("/api/add", api.ApiAddMapping)
	http.HandleFunc("/api/delete", api.ApiDeleteMapping)
	http.HandleFunc("/api/query", api.ApiQueryMappings)

	log.SetOutput(os.Stdout) // 防止日志收集被打印到错误里

	log.Printf("API 服务正在监听 %s 端口", apiPort)
	log.Fatal(http.ListenAndServe("127.0.0.1:"+apiPort, nil))
}
