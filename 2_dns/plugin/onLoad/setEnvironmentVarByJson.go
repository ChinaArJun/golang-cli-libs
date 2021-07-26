package main

import (
	"encoding/json"
	"github.com/emicklei/go-restful/log"
	"io/ioutil"
	"os"
)

type OnLoad struct{}

type PluginOnLoadInterface interface {
	OnLoad(...interface{})
}

//fixme: return error
func (el *OnLoad) OnLoad(conf ...interface{}) {
	var err error
	var fileContent []byte
	var jsonData map[string]interface{}

	if _, err = os.Stat(conf[0].([]interface{})[0].(string)); os.IsNotExist(err) {
		return
	}

	fileContent, err = ioutil.ReadFile(conf[0].([]interface{})[0].(string))

	err = json.Unmarshal(fileContent, &jsonData)
	if err != nil {
		log.Printf("plugin onLoad() json error: %v\n", err.Error())
		return
	}

	for key, value := range jsonData {
		err = os.Setenv(key, value.(string))
		if err != nil {
			log.Printf("plugin onLoad() > os.Setenv(%v, %v).error: %v\n", key, value, err.Error())
			return
		}

		log.Printf("plugin onLoad() > os.Setenv(%v, %v)\n", key, value)
	}
}

var PluginOnLoad OnLoad
