package config

import (
	"flag"
	"fmt"
	"log"
	"os"

	"main/mapping"
)

type Config struct {
	Mappings []Mapping `yaml:"mappings"`
}

type Mapping struct {
	SourcePort  int    `yaml:"source_port"`
	TargetIP    string `yaml:"target_ip"`
	TargetPort  int    `yaml:"target_port"`
	MappingType string `yaml:"mapping_type"`
}

var config Config

func ParseArgs(apiPort, authCode *string, configPath *string, version string) {

	flag.StringVar(configPath, "config", "config.yml", "配置文件 保存路径")
	flag.StringVar(apiPort, "p", "7655", "API服务监听端口")
	flag.StringVar(authCode, "code", "", "API访问授权码 (必填)")
	versionFlag := flag.Bool("v", false, "打印版本并退出")

	flag.Parse()

	if *versionFlag {
		fmt.Println("版本:", version)
		fmt.Println("iris-n2n-launcher-3 组件")
		fmt.Println("花之链环： @5656565566")
		os.Exit(0)
	}

	if *authCode == "" {
		flag.Usage()
		log.Fatalf("未提供API访问授权码")
	}

	if !fileExists(*configPath) {
		file, err := os.Create(*configPath)
		if err != nil {
			fmt.Println("创建配置文件时出错:", err)
			return
		}
		file.Close()
	}

	cfg, err := parseConfig(*configPath)
	if err != nil {
		log.Fatalf("无法打开配置文件: %v", err)
	}
	config = *cfg

	initializeMappings()
}

func initializeMappings() {
	for _, m := range config.Mappings {
		listenAddr := fmt.Sprintf(":%d", m.SourcePort)
		forwardAddr := fmt.Sprintf("%s:%d", m.TargetIP, m.TargetPort)

		switch m.MappingType {
		case "tcp":
			mapping.AddTcpMapping(listenAddr, forwardAddr)
		case "udp":
			mapping.AddUdpMapping(listenAddr, forwardAddr)
		case "tcpudp", "udptcp", "":
			mapping.AddTcpMapping(listenAddr, forwardAddr)
			mapping.AddUdpMapping(listenAddr, forwardAddr)
		}
	}
}
