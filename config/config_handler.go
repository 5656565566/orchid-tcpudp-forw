package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

func parseConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil
}

func deduplicateMappings(mappings []Mapping) []Mapping {
	seen := make(map[string]bool)
	var result []Mapping

	for _, m := range mappings {
		key := fmt.Sprintf("%d-%s-%d-%s", m.SourcePort, m.TargetIP, m.TargetPort, m.MappingType)
		if !seen[key] {
			seen[key] = true
			result = append(result, m)
		}
	}
	return result
}

func WriteConfig(filename string) error {
	config.Mappings = deduplicateMappings(config.Mappings)

	data, err := yaml.Marshal(&config)
	if err != nil {
		fmt.Println("数据错误")
		fmt.Println(err)
		return err
	}

	err = os.WriteFile(filename, data, 0777)
	if err != nil {
		fmt.Println("文件写入错误")
		fmt.Println(err)
		return err
	}

	return nil
}

func RemoveMappingBySourcePort(mappings []Mapping, sourcePort int, mappingType string) []Mapping {
	var updatedMappings []Mapping
	for _, mapping := range mappings {
		if mapping.SourcePort != sourcePort {
			updatedMappings = append(updatedMappings, mapping)
		}
		if mapping.SourcePort == sourcePort && (mapping.MappingType == "tcpudp" || mapping.MappingType == "udptcp") {
			if mappingType == "tcp" {
				mapping.MappingType = "udp"
				updatedMappings = append(updatedMappings, mapping)
			}
			if mappingType == "udp" {
				mapping.MappingType = "tcp"
				updatedMappings = append(updatedMappings, mapping)
			}
		}
	}
	return updatedMappings
}

func UpdateConfig(newMapping Mapping) {
	if len(config.Mappings) == 0 {
		config.Mappings = []Mapping{newMapping}
	} else {
		config.Mappings = append(config.Mappings, newMapping)
	}
}

func GetConfig() *Config {
	return &config
}

func SetConfig(cfg *Config) {
	config = *cfg
}

func ParseAddr(addr string) (string, int) {
	parts := strings.Split(addr, ":")
	if len(parts) != 2 {
		return "", 0
	}
	var port int
	fmt.Sscan(parts[1], &port)
	return parts[0], port
}
