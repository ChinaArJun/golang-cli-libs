package main

import (
	"encoding/json"
	"fmt"
	"github.com/helmutkemper/communsTypesForGolangPlugin"
	"github.com/helmutkemper/dns"
	"io/ioutil"
	"log"
	"os"
	"plugin"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	kPlugFileListPath = "./config/plugin.json"
	//kPlugFileListPath = "/go/src/app/2_dns/config/plugin.json"
	kPluginTickerIntervalMillisecond = 1000
)

type pluginListJson struct {
	Type string      `json:"type"`
	Path string      `json:"path"`
	Conf interface{} `json:"conf"`
}

var pluginHttpServer PluginHttpServerInterface
var pluginData PluginDataInterface
var pluginOnLoad PluginOnLoadInterface
var pluginDns PluginDnsInterface
var fileTimeModifiedLastRead time.Time

type PluginHttpServerInterface interface {
	Connect() error
	OnLoad(conf ...interface{}) error
	SetDataGet(v func([]byte) (error, int, []communsTypes.KeyValueType))
	SetDataPut(v func(communsTypes.KeyValueType) error)
	SetDataDelete(v func([]byte) error)
	SelfRegister() error
	Register(name, target string, port int) error
	GetServiceKeyPrefix() string
}

type PluginDataInterface interface {
	OnLoad(conf ...interface{}) error
	Connect() error
	Close() error
	Check() bool
	Put(value communsTypes.KeyValueType) error
	GetByPrefix(prefix []byte) (error, int, []communsTypes.KeyValueType)
	Get(key []byte) (error, int, []communsTypes.KeyValueType)
	SetOnWatch(watchFunc func([]communsTypes.KeyValueType, []communsTypes.KeyValueType))
	Watch(key []byte)
	Delete(key []byte) error
}

type PluginOnLoadInterface interface {
	OnLoad(...interface{})
}

type PluginDnsInterface interface {
	OnLoad(...interface{}) error
	Set(serviceList map[string]map[dns.Type][]dns.Record)
	GetAddressAndPort() string
	SetServiceByName(serviceName string, v map[dns.Type][]dns.Record)
	SetServiceBySRV(serviceName string, JSon []byte)
	AppendNewRegisterInServiceByName(serviceName string, v dns.Record)
	RemoveRegisterFromServiceByName(serviceName string, v dns.Record)
	RemoveServiceByName(serviceName string)
	Connect() error
	Test() error
}

func openPluginHttpServer(path string, conf interface{}) PluginHttpServerInterface {

	fmt.Printf("path %v\n", path)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}

	plug, err := plugin.Open(path)
	if err != nil {
		log.Fatal(err.Error())
	}

	dataInterface, err := plug.Lookup("PluginData")
	if err != nil {
		log.Fatal(err.Error())
	}

	pluginHttpServerLoaded, ok := dataInterface.(PluginHttpServerInterface)
	if !ok {
		fmt.Println("openPluginHttpServer(): unexpected type from module symbol")
		os.Exit(1)
	}

	err = pluginHttpServerLoaded.OnLoad(conf)
	if err != nil {
		log.Printf("pluginData.OnLoad().Error: %v\n", err.Error())
	}

	pluginHttpServerLoaded.SetDataGet(pluginData.Get)
	pluginHttpServerLoaded.SetDataPut(pluginData.Put)
	pluginHttpServerLoaded.SetDataDelete(pluginData.Delete)

	go pluginHttpServerLoaded.Connect()
	time.Sleep(time.Millisecond * 333)

	pluginHttpServerLoaded.SelfRegister()

	return pluginHttpServerLoaded
}

func openPluginDns(path string, conf interface{}) PluginDnsInterface {

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}

	plug, err := plugin.Open(path)
	if err != nil {
		log.Fatal(err.Error())
	}

	dataInterface, err := plug.Lookup("PluginData")
	if err != nil {
		log.Fatal(err.Error())
	}

	pluginData, ok := dataInterface.(PluginDnsInterface)
	if !ok {
		fmt.Println("openPluginDns(): unexpected type from module symbol")
		os.Exit(1)
	}

	err = pluginData.OnLoad(conf)
	if err != nil {
		log.Printf("pluginData.OnLoad().Error: %v\n", err.Error())
	}

	go pluginData.Connect()
	time.Sleep(time.Millisecond * 333)

	pluginData.Test()

	return pluginData
}

func openPluginData(path string, conf interface{}) PluginDataInterface {

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}

	plug, err := plugin.Open(path)
	if err != nil {
		log.Fatal(err.Error())
	}

	dataInterface, err := plug.Lookup("PluginData")
	if err != nil {
		log.Fatal(err.Error())
	}

	pluginData, ok := dataInterface.(PluginDataInterface)
	if !ok {
		fmt.Println("openPluginData(): unexpected type from module symbol")
		os.Exit(1)
	}

	err = pluginData.OnLoad(conf)
	if err != nil {
		log.Printf("pluginData.OnLoad().Error: %v\n", err.Error())
	}

	err = pluginData.Connect()
	if err != nil {
		log.Printf("pluginData.Connect().Error: %v\n", err.Error())
	}

	return pluginData
}

func openPluginOnLoad(path string, conf interface{}) PluginOnLoadInterface {

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}

	plug, err := plugin.Open(path)
	if err != nil {
		log.Fatal(err.Error())
	}

	onLoadInterface, err := plug.Lookup("PluginOnLoad")
	if err != nil {
		log.Fatal(err.Error())
	}

	pluginOnLoad, ok := onLoadInterface.(PluginOnLoadInterface)
	if !ok {
		fmt.Println("openPluginOnLoad(): unexpected type from module symbol")
		os.Exit(1)
	}

	pluginOnLoad.OnLoad(conf)

	return pluginOnLoad
}

func onDndChange(e dns.Event, k string, old, new interface{}) {

	fmt.Printf("event: %v\n", e.String())

	switch converted := old.(type) {
	case map[string]map[dns.Type][]dns.Record:

		for Type, Records := range converted[k] {
			fmt.Printf("type: %v\n", Type)
			for _, rv := range Records {
				fmt.Printf("old record %v\n", rv)
			}
		}

	case map[dns.Type][]dns.Record:

		for Type, Records := range converted {
			fmt.Printf("type: %v\n", Type)
			for _, rv := range Records {
				fmt.Printf("old record %v\n", rv)
			}
		}

	}

	switch converted := new.(type) {
	case map[string]map[dns.Type][]dns.Record:

		for Type, Records := range converted[k] {
			fmt.Printf("type: %v\n", Type)
			for _, rv := range Records {
				fmt.Printf("new record %v\n", rv)
			}
		}

	case map[dns.Type][]dns.Record:

		for Type, Records := range converted {
			fmt.Printf("type: %v\n", Type)
			for _, rv := range Records {
				fmt.Printf("new record %v\n", rv)
			}
		}

	}

}

func onDnsBeforeOnChange(v func(string, map[string]map[dns.Type][]dns.Record)) {
	fmt.Printf("onDnsBeforeOnChange(%v, %v)\n")
}

// maquina de casa
//go:generate /home/hkemper/Downloads/golang_binaries/go1.11.4/bin/go build -buildmode=plugin -installsuffix=shared -gcflags=-shared -installsuffix=dynlink -gcflags=-dynlink -o /home/hkemper/Dropbox/gRPC/2_dns/plugin/dataPlugin/etcd/etcd.so /home/hkemper/Dropbox/gRPC/2_dns/plugin/dataPlugin/etcd/etcd.go
//go:generate /home/hkemper/Downloads/golang_binaries/go1.11.4/bin/go build -buildmode=plugin -installsuffix=shared -gcflags=-shared -installsuffix=dynlink -gcflags=-dynlink -o /home/hkemper/Dropbox/gRPC/2_dns/plugin/onLoad/setEnvironmentVarByJson.so /home/hkemper/Dropbox/gRPC/2_dns/plugin/onLoad/setEnvironmentVarByJson.go
//go:generate /home/hkemper/Downloads/golang_binaries/go1.11.4/bin/go build -buildmode=plugin -installsuffix=shared -gcflags=-shared -installsuffix=dynlink -gcflags=-dynlink -o /home/hkemper/Dropbox/gRPC/2_dns/plugin/serviceDiscover/dns/benBurkertDns.so /home/hkemper/Dropbox/gRPC/2_dns/plugin/serviceDiscover/dns/benBurkertDns.go
//go:generate /home/hkemper/Downloads/golang_binaries/go1.11.4/bin/go build -buildmode=plugin -installsuffix=shared -gcflags=-shared -installsuffix=dynlink -gcflags=-dynlink -o /home/hkemper/Dropbox/gRPC/2_dns/plugin/serviceDiscover/httpServer/benBurkertDnsCompatibleHttpServer.so /home/hkemper/Dropbox/gRPC/2_dns/plugin/serviceDiscover/httpServer/benBurkertDnsCompatibleHttpServer.go

// maquina da empresa
//go:generate /home/kemper/Programas/Golang/go1.11.4/bin/go build -buildmode=plugin -installsuffix=shared -gcflags=-shared -installsuffix=dynlink -gcflags=-dynlink -o /home/kemper/Projetos/ahgora/gRPC/2_dns/plugin/dataPlugin/etcd/etcd.so /home/kemper/Projetos/ahgora/gRPC/2_dns/plugin/dataPlugin/etcd/etcd.go
//go:generate /home/kemper/Programas/Golang/go1.11.4/bin/go build -buildmode=plugin -installsuffix=shared -gcflags=-shared -installsuffix=dynlink -gcflags=-dynlink -o /home/kemper/Projetos/ahgora/gRPC/2_dns/plugin/onLoad/setEnvironmentVarByJson.so /home/kemper/Projetos/ahgora/gRPC/2_dns/plugin/onLoad/setEnvironmentVarByJson.go
//go:generate /home/kemper/Programas/Golang/go1.11.4/bin/go build -buildmode=plugin -installsuffix=shared -gcflags=-shared -installsuffix=dynlink -gcflags=-dynlink -o /home/kemper/Projetos/ahgora/gRPC/2_dns/plugin/serviceDiscover/dns/benBurkertDns.so /home/kemper/Projetos/ahgora/gRPC/2_dns/plugin/serviceDiscover/dns/benBurkertDns.go
//go:generate /home/kemper/Programas/Golang/go1.11.4/bin/go build -buildmode=plugin -installsuffix=shared -gcflags=-shared -installsuffix=dynlink -gcflags=-dynlink -o /home/kemper/Projetos/ahgora/gRPC/2_dns/plugin/serviceDiscover/httpServer/benBurkertDnsCompatibleHttpServer.so /home/kemper/Projetos/ahgora/gRPC/2_dns/plugin/serviceDiscover/httpServer/benBurkertDnsCompatibleHttpServer.go

func main() {
	var wg sync.WaitGroup

	go tickerFuncToReloadPlugin(time.NewTicker(kPluginTickerIntervalMillisecond * time.Millisecond))

	wg.Add(1)

	wg.Wait()
}

func tickerFuncToReloadPlugin(ticker *time.Ticker) {
	var err error
	var pluginList []pluginListJson
	var pluginFileList []byte
	var fileStatInfo os.FileInfo
	var fileTimeModified time.Time

	for range ticker.C {
		if _, err := os.Stat(kPlugFileListPath); os.IsNotExist(err) {
			log.Printf("plugin not found at apth: %v\n", kPlugFileListPath)
			continue
		}

		fileStatInfo, err = os.Stat(kPlugFileListPath)
		fileTimeModified = fileStatInfo.ModTime()

		if fileTimeModifiedLastRead == fileTimeModified {
			continue
		}

		fileTimeModifiedLastRead = fileTimeModified

		pluginFileList, err = ioutil.ReadFile(kPlugFileListPath)
		if err != nil {
			log.Printf("plugin file read error\n")
			continue
		}

		err = json.Unmarshal(pluginFileList, &pluginList)
		if err != nil {
			log.Printf("plugin file list error: %v\n", err.Error())
		}

		var pass = false
		for _, fileData := range pluginList {
			if fileData.Type == "pluginOnLoad" {
				log.Printf("loading pluginOnLoad() at path %v\n", fileData.Path)
				pluginOnLoad = openPluginOnLoad(fileData.Path, fileData.Conf)
			}
		}

		for _, fileData := range pluginList {
			if fileData.Type == "pluginData" {
				pass = true
				log.Printf("loading pluginData() at path %v\n", fileData.Path)
				pluginData = openPluginData(fileData.Path, fileData.Conf)

				if pluginData != nil {
					var dataLido []communsTypes.KeyValueType
					var dataToPut communsTypes.KeyValueType
					dataToPut.K = []byte("dataPlugin")
					dataToPut.V = []byte("data plugin is alive")
					err = pluginData.Put(dataToPut)
					if err != nil {
						log.Printf("pluginData.Put().Error: %v\n", err.Error())
						pluginData = nil
					}

					err, n, dataLido := pluginData.Get([]byte("dataPlugin"))
					if err != nil {
						log.Printf("pluginData.Get().Error: %v\n", err.Error())
						pluginData = nil
					}

					fmt.Printf("Data [%v]: %s\n", n, dataLido)
				}

				continue
			}
		}

		for _, fileData := range pluginList {
			if fileData.Type == "pluginDns" {
				log.Printf("loading pluginDns() at path %v\n", fileData.Path)
				pluginDns = openPluginDns(fileData.Path, fileData.Conf)
			}

			if fileData.Type == "pluginHttpServer" {
				log.Printf("loading pluginHttpServer() at path %v\n", fileData.Path)
				pluginHttpServer = openPluginHttpServer(fileData.Path, fileData.Conf)
			}
		}

		if pass == false {
			log.Printf("plugin file loaded, but, no one pluging loaded\n")
		}

		if pluginData != nil && pluginDns != nil {
			prefix := pluginHttpServer.GetServiceKeyPrefix()
			pluginData.Watch([]byte(prefix))
			pluginData.SetOnWatch(func(new []communsTypes.KeyValueType, old []communsTypes.KeyValueType) {
				found := len(new)

				for k := range new {

					keyToFind := strings.Replace(string(new[k].K), prefix, "", 1)
					if found == 0 {
						pluginDns.RemoveServiceByName(keyToFind)
					} else {
						pluginDns.SetServiceBySRV(keyToFind, new[k].V)
					}

				}

			})
		}

		if pluginHttpServer != nil && pluginDns != nil {
			addr := pluginDns.GetAddressAndPort()
			re, err := regexp.Compile("^(.*?:)(.*)$")
			if err != nil {
				log.Fatal(err)
			}

			portStr := re.ReplaceAll([]byte(addr), []byte("$2"))
			port, err := strconv.ParseInt(string(portStr), 10, 64)
			if err != nil {
				log.Fatal(err)
			}

			err = pluginHttpServer.Register("dns.service.discover", "", int(port))
			if err != nil {
				log.Fatal(err)
			}
		}

		if pluginData != nil && pluginHttpServer != nil && pluginDns != nil {
			prefix := pluginHttpServer.GetServiceKeyPrefix()

			var dataToPopulateDnsRecords []communsTypes.KeyValueType
			err, _, dataToPopulateDnsRecords = pluginData.GetByPrefix([]byte(prefix))

			for k := range dataToPopulateDnsRecords {
				keyToFind := strings.Replace(string(dataToPopulateDnsRecords[k].K), prefix, "", 1)
				pluginDns.SetServiceBySRV(keyToFind, dataToPopulateDnsRecords[k].V)

			}
		}
	}
}
