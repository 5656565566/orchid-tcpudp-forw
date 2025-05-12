package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"main/config"
	"main/mapping"
)

var (
	authCode   string
	configPath string
)

func SetAuthCode(code string) {
	authCode = code
}

func SetConfigPath(path string) {
	configPath = path
}

func validateAuthCode(r *http.Request) error {
	code := r.Header.Get("Authorization")
	if code != authCode {
		return fmt.Errorf("未授权的访问")
	}
	return nil
}

func ApiAddMapping(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "无效的请求方法", http.StatusMethodNotAllowed)
		return
	}
	if err := validateAuthCode(r); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var data map[string]interface{}
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()

	if err := decoder.Decode(&data); err != nil {
		http.Error(w, "无效的 JSON", http.StatusBadRequest)
		return
	}

	listenAddr, ok := data["listenAddr"].(string)
	if !ok {
		http.Error(w, "缺少listenAddr参数", http.StatusBadRequest)
		return
	}

	forwardAddr, ok := data["forwardAddr"].(string)
	if !ok {
		http.Error(w, "缺少forwardAddr参数", http.StatusBadRequest)
		return
	}

	mappingType, ok := data["mappingType"].(string)
	if !ok {
		http.Error(w, "缺少mappingType参数", http.StatusBadRequest)
		return
	}

	// 检查重复映射
	checkExisting := func(mType string) bool {
		switch mType {
		case "tcp":
			_, exists := mapping.MappingsTcp.Load(listenAddr)
			return exists
		case "udp":
			_, exists := mapping.MappingsUdp.Load(listenAddr)
			return exists
		}
		return false
	}

	switch mappingType {
	case "tcp":
		if checkExisting("tcp") {
			http.Error(w, "TCP映射已存在", http.StatusConflict)
			return
		}
		if err := mapping.AddTcpMapping(listenAddr, forwardAddr); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	case "udp":
		if checkExisting("udp") {
			http.Error(w, "UDP映射已存在", http.StatusConflict)
			return
		}
		if err := mapping.AddUdpMapping(listenAddr, forwardAddr); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	case "udptcp", "tcpudp":
		if checkExisting("tcp") || checkExisting("udp") {
			http.Error(w, "TCP或UDP映射已存在", http.StatusConflict)
			return
		}
		if err := mapping.AddUdpMapping(listenAddr, forwardAddr); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := mapping.AddTcpMapping(listenAddr, forwardAddr); err != nil {
			mapping.DeleteUdpMapping(listenAddr) // 回滚已添加的UDP映射
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	default:
		http.Error(w, "无效的映射类型", http.StatusBadRequest)
		return
	}

	if temp, ok := data["temp"].(bool); !ok || !temp {
		_, port := config.ParseAddr(listenAddr)
		targetIP, targetPort := config.ParseAddr(forwardAddr)
		newMapping := config.Mapping{
			SourcePort:  port,
			TargetIP:    targetIP,
			TargetPort:  targetPort,
			MappingType: mappingType,
		}
		config.UpdateConfig(newMapping)
		config.WriteConfig(configPath)
	}

	w.WriteHeader(http.StatusOK)
}

func ApiDeleteMapping(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "无效的请求方法", http.StatusMethodNotAllowed)
		return
	}
	if err := validateAuthCode(r); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	listenAddr := r.URL.Query().Get("listenAddr")
	if listenAddr == "" {
		http.Error(w, "缺少listenAddr参数", http.StatusBadRequest)
		return
	}

	mappingType := r.URL.Query().Get("mappingType")
	if mappingType == "" {
		http.Error(w, "缺少mappingType参数", http.StatusBadRequest)
		return
	}

	switch mappingType {
	case "tcp":
		if err := mapping.DeleteTcpMapping(listenAddr); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	case "udp":
		if err := mapping.DeleteUdpMapping(listenAddr); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	case "tcpudp", "udptcp":
		if err := mapping.DeleteTcpMapping(listenAddr); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := mapping.DeleteUdpMapping(listenAddr); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	_, port := config.ParseAddr(listenAddr)
	cfg := config.GetConfig()
	cfg.Mappings = config.RemoveMappingBySourcePort(cfg.Mappings, port, mappingType)
	config.SetConfig(cfg)
	config.WriteConfig(configPath)

	w.WriteHeader(http.StatusOK)
}

func ApiQueryMappings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "无效的请求方法", http.StatusMethodNotAllowed)
		return
	}
	if err := validateAuthCode(r); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	type MappingInfo struct {
		ListenAddr  string `json:"listen_addr"`
		ForwardAddr string `json:"forward_addr"`
		MappingType string `json:"mapping_type"`
	}

	var mappingsInfo []MappingInfo

	mapping.MappingsTcp.Range(func(key, value interface{}) bool {
		m := value.(*mapping.TcpPortMapping)
		mappingsInfo = append(mappingsInfo, MappingInfo{
			ListenAddr:  m.ListenAddr,
			ForwardAddr: m.ForwardAddr,
			MappingType: "tcp",
		})
		return true
	})

	mapping.MappingsUdp.Range(func(key, value any) bool {
		m := value.(*mapping.UdpPortMapping)
		mappingsInfo = append(mappingsInfo, MappingInfo{
			ListenAddr:  m.ListenAddr,
			ForwardAddr: m.ForwardAddr,
			MappingType: "udp",
		})
		return true
	})

	jsonResponse, err := json.Marshal(mappingsInfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := config.WriteConfig(configPath); err != nil {
		fmt.Print(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
}
