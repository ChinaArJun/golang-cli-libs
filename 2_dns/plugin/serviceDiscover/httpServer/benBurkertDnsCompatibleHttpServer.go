/*
JSon config format:
{
  "port": 8080,
  "servicePrefix": "service.discover.",
  "register": [
    {
      "schema": "http",
      "endpoint": "service",
      "name": "http.service.discover"
    }
  ]
}

url format:
http[s]://{server}:{port}/service/{service_name}

[POST] localhost:8080/service/node
[PUT]  localhost:8080/service/node

Raw JSon data format to send
{
  "port":     int,
  "target":   string ended in point. ex.:"192.169.0.1." or "mongodb." [optional - when this value is omitted, there is the remote address of the client]
}

JSon return format
{
    "Meta": {
        "TotalCount": 1,
        "Success": true,
        "Error": ""
    },
    "Objects": [
        {
            "Priority": 10,
            "Weight": 10,
            "Port": 8080,
            "Target": "192.168.10.1."
        }
    ]
}

[GET]  localhost:8080/service/node

JSon return format
{
    "Meta": {
        "TotalCount": 2,
        "Success": true,
        "Error": ""
    },
    "Objects": [
        {
            "Priority": 10,
            "Weight": 10,
            "Port": 8080,
            "Target": "192.168.10.1."
        },
        {
            "Priority": 10,
            "Weight": 10,
            "Port": 8080,
            "Target": "192.168.10.2."
        }
    ]
}
*/
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/helmutkemper/communsTypesForGolangPlugin"
	"github.com/helmutkemper/dns"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// used to organize the http server
type handle struct {
	Method string
	Type   string
	Func   func(w http.ResponseWriter, r *http.Request)
}

// the list of endpoints and their respective functions
type handleList []handle

// data input format from endpoint name service
type service struct {
	Port   int
	Target string
}

// meta object compliant with http://json-schema.org/
type MetaJSonOut struct {
	TotalCount int    `json:"TotalCount"`
	Success    bool   `json:"Success"`
	Error      string `json:"Error"`
}

type configJSonRegister struct {
	Schema   string
	Endpoint string
	Name     string
}

type configJSon struct {
	Port          int
	ServicePrefix string
	Register      []configJSonRegister
}

// output object compliant with http://json-schema.org/
type JSonOut struct {
	Meta    MetaJSonOut `json:"Meta"`
	Objects interface{} `json:"Objects"`
}

// data to output object compliant with http://json-schema.org/
func (el *JSonOut) ToOutput(totalCountAInt int, errorAErr error, dataATfc interface{}, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json;")

	if errorAErr != nil {
		w.WriteHeader(http.StatusInternalServerError)

		el.Meta = MetaJSonOut{
			Error:      fmt.Sprint(errorAErr),
			Success:    false,
			TotalCount: 0,
		}
		el.Objects = make([]int, 0)
	} else {
		if totalCountAInt == 0 {
			w.WriteHeader(http.StatusOK)

			el.Meta = MetaJSonOut{
				Error:      "",
				Success:    true,
				TotalCount: totalCountAInt,
			}
			el.Objects = make([]int, 0)
		} else {
			el.Meta = MetaJSonOut{
				Error:      "",
				Success:    true,
				TotalCount: totalCountAInt,
			}
			el.Objects = dataATfc
		}
	}

	if err := json.NewEncoder(w).Encode(el); err != nil {
		log.Printf("[Ben Burkert DNS compatible http server plugin log] %v", err)
	}
}

// plugin external interface
type PluginHttpServerInterface interface {
	Connect() error
	OnLoad(conf ...interface{}) error
	SetDataGet(v func([]byte, []communsTypes.KeyValueType) (error, int))
	SetDataPut(v func(communsTypes.KeyValueType) error)
	SetDataDelete(v func([]byte) error)
	SelfRegister() error
	Register(name, target string, port int) error
	GetServiceKeyPrefix() string
}

// plugin main struct
type HttpServer struct {
	handleList    handleList
	port          int
	servicePrefix string
	dataGet       func([]byte) (error, int, []communsTypes.KeyValueType)
	dataPut       func(communsTypes.KeyValueType) error
	dataDelete    func([]byte) error
	register      []configJSonRegister
}

// plugin on load function
// this is a first function to run after plugin loaded
// conf[0] - string containing a json file path of configuration file
//
//   json file example:
//   {
//     "addressAndPort": ":8080",
//     "servicePrefix": "service"
//   }
func (el *HttpServer) OnLoad(conf ...interface{}) error {
	var err error
	var fileContent []byte
	var jsonData configJSon
	var filePath = conf[0].([]interface{})[0].(string)

	if _, err = os.Stat(filePath); os.IsNotExist(err) {
		el.handleError(err)
		return err
	}

	fileContent, err = ioutil.ReadFile(filePath)
	if err != nil {
		el.handleError(err)
		return err
	}

	err = json.Unmarshal(fileContent, &jsonData)
	if err != nil {
		el.handleError(err)
		return err
	}

	el.port = jsonData.Port
	if el.port == 0 {
		err = errors.New("json port key not found")
		el.handleError(err)
		return err
	}

	el.servicePrefix = "." + jsonData.ServicePrefix
	if el.servicePrefix == "." {
		el.servicePrefix = "service.discover."
		el.log("servicePrefix set to service.discover.")
	}

	el.register = jsonData.Register
	if len(el.register) == 0 {
		err = errors.New("json register key not found")
		el.handleError(err)
		return err
	}

	return nil
}

// set this http server functions on DNS registers by self http interface
func (el *HttpServer) SelfRegister() error {
	var selfAddress string
	var err error
	selfAddress, err = el.externalIP()
	if err != nil {
		el.handleError(err)
		return err
	}

	for _, register := range el.register {
		url := register.Schema + "://" + selfAddress + ":" + strconv.Itoa(el.port) + "/" + register.Endpoint + "/" + register.Name
		req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(`{ "port": `+strconv.Itoa(el.port)+`, "target": "`+register.Schema+"://"+selfAddress+":"+strconv.Itoa(el.port)+"/"+register.Endpoint+"/"+`." }`)))
		if err != nil {
			el.handleError(err)
			return err
		}

		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			el.handleError(err)
			return err
		}

		body, _ := ioutil.ReadAll(resp.Body)
		err = resp.Body.Close()
		if err != nil {
			el.handleError(err)
			return err
		}

		el.log(fmt.Sprintf("self register url %v\n", url))
		el.log(fmt.Sprintf("self register response %s\n", body))
	}

	return nil
}

// allows the registration of a new service by the http interface
func (el *HttpServer) Register(name, target string, port int) error {
	var selfAddress string
	var err error
	selfAddress, err = el.externalIP()
	if err != nil {
		el.handleError(err)
		return err
	}

	for _, register := range el.register {
		url := register.Schema + "://" + selfAddress + ":" + strconv.Itoa(el.port) + "/" + register.Endpoint + "/" + name
		req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(`{ "port": `+strconv.Itoa(port)+`, "target": "`+target+`" }`)))
		if err != nil {
			el.handleError(err)
			return err
		}

		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			el.handleError(err)
			return err
		}

		body, _ := ioutil.ReadAll(resp.Body)
		err = resp.Body.Close()
		if err != nil {
			el.handleError(err)
			return err
		}

		el.log(fmt.Sprintf("self register url %v\n", url))
		el.log(fmt.Sprintf("self register response %s\n", body))
	}

	return nil
}

// plugin connect function
func (el *HttpServer) Connect() error {
	server := http.NewServeMux()
	server.HandleFunc("/", el.handleFunc)

	newServer := &http.Server{
		Addr:    ":" + strconv.Itoa(el.port),
		Handler: server,
	}

	return newServer.ListenAndServe()
}

func (el HttpServer) GetServiceKeyPrefix() string {
	return el.servicePrefix
}

// plugin set dataGet from external plugin data function
func (el *HttpServer) SetDataGet(v func([]byte) (error, int, []communsTypes.KeyValueType)) {
	el.dataGet = v
}

// plugin set dataPut from external plugin data function
func (el *HttpServer) SetDataPut(v func(communsTypes.KeyValueType) error) {
	el.dataPut = v
}

// plugin set dataDelete from external plugin data function
func (el *HttpServer) SetDataDelete(v func([]byte) error) {
	el.dataDelete = v
}

// shown critical erros in log with file and line numbers
func (el *HttpServer) handleError(err error) {
	if err != nil {
		_, fn, line, _ := runtime.Caller(1)
		log.Printf("[Ben Burkert DNS compatible http server plugin error] %s:%d %v", fn, line, err)
	}
}

// shows the log identifying the plugin generator information
func (el *HttpServer) log(info string) {
	log.Printf("[Ben Burkert DNS compatible http server plugin log] %v", info)
}

// http handle function
func (el *HttpServer) handleFunc(w http.ResponseWriter, r *http.Request) {
	el.handleList = []handle{
		{
			Method: http.MethodPost,
			Type:   "service",
			Func:   el.handlePutService,
		},
		{
			Method: http.MethodPut,
			Type:   "service",
			Func:   el.handlePutService,
		},
		{
			Method: http.MethodGet,
			Type:   "service",
			Func:   el.handleGetService,
		},
		{
			Method: http.MethodDelete,
			Type:   "service",
			Func:   el.handleDeleteService,
		},
	}

	urlElements := strings.Split(r.Method+r.URL.Path, "/")

	method := urlElements[0]
	endpointType := urlElements[1]

	for _, handleData := range el.handleList {

		if handleData.Method == method && handleData.Type == endpointType {
			handleData.Func(w, r)
		}
	}
}

// http delete method function
// this method delete the DNS record that has the same port and the same target contained in the original list of
// records from service name.
// If there are no more records in the list record the whole service is deleted from the database.
//
//  Raw JSon data format
//  {
//    "port":     int,
//    "target":   string ended in point. ex.:"192.169.0.1." or "mongodb."
//  }
//
//  JSon output format:
//  {
//    "Meta": {
//        "TotalCount": 2,
//        "Success": true,
//        "Error": ""
//    },
//    "Objects": [
//        {
//            "Priority": 10,
//            "Weight": 10,
//            "Port": 8080,
//            "Target": "192.168.10.1."
//        },
//        {
//            "Priority": 10,
//            "Weight": 10,
//            "Port": 8080,
//            "Target": "192.168.10.2."
//        }
//    ]
//  }
func (el *HttpServer) handleDeleteService(w http.ResponseWriter, r *http.Request) {
	var err error
	var inData service
	var records []dns.SRV
	var jsonData []byte
	var dataToSave communsTypes.KeyValueType
	var output JSonOut
	var found int
	var dataFromDataSource []communsTypes.KeyValueType

	w.Header().Add("Content-Type", "application/json")

	if el.dataGet == nil {
		output.ToOutput(0, errors.New("ben burkert dns plugin config error. please, define a getData function"), nil, w)
		return
	}

	if el.dataPut == nil {
		output.ToOutput(0, errors.New("ben burkert dns plugin config error. please, define a putData function"), nil, w)
		return
	}

	urlElements := strings.Split(r.Method+r.URL.Path, "/")
	serviceName := urlElements[2]

	jsonData, err = ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(503)
		el.handleError(err)
		output.ToOutput(0, errors.New("internal server error"), nil, w)
		return
	}

	err = json.Unmarshal(jsonData, &inData)
	if err != nil {
		w.WriteHeader(503)
		output.ToOutput(0, errors.New("unmarshal incoming json from client side error: "+err.Error()), nil, w)
		return
	}

	err, found, dataFromDataSource = el.dataGet([]byte(el.servicePrefix + serviceName))
	if err != nil {
		w.WriteHeader(503)
		el.handleError(err)
		output.ToOutput(0, errors.New("internal server error"), nil, w)
		return
	}

	if found == 0 {
		output.ToOutput(0, nil, nil, w)
		return
	}

	for dataSourceKey := range dataFromDataSource {
		err = json.Unmarshal(dataFromDataSource[dataSourceKey].V, &records)
		if err != nil {
			w.WriteHeader(503)
			el.handleError(err)
			output.ToOutput(0, errors.New("internal server error"), nil, w)
			return
		}

		for k, record := range records {
			rTest := record.Get()
			if rTest.(*dns.SRV).Port == inData.Port && rTest.(*dns.SRV).Target == inData.Target {

				records = append(records[:k], records[k+1:]...)

				dataToSave.V, err = json.Marshal(&records)
				dataToSave.K = []byte(el.servicePrefix + serviceName)
				if err != nil {
					w.WriteHeader(503)
					el.handleError(err)
					output.ToOutput(0, errors.New("internal server error"), nil, w)
					return
				}

				if len(records) == 0 {
					err = el.dataDelete([]byte(el.servicePrefix + serviceName))
					if err != nil {
						w.WriteHeader(503)
						el.handleError(err)
						output.ToOutput(0, errors.New("internal server error"), nil, w)
						return
					}
				} else {
					err = el.dataPut(dataToSave)
					if err != nil {
						w.WriteHeader(503)
						el.handleError(err)
						output.ToOutput(0, errors.New("internal server error"), nil, w)
						return
					}
				}
				break
			}
		}
	}

	output.ToOutput(len(records), nil, records, w)
}

// http get method function
// this method get the DNS record list from service name
//
//  JSon output format:
//  {
//    "Meta": {
//        "TotalCount": 2,
//        "Success": true,
//        "Error": ""
//    },
//    "Objects": [
//        {
//            "Priority": 10,
//            "Weight": 10,
//            "Port": 8080,
//            "Target": "192.168.10.1."
//        },
//        {
//            "Priority": 10,
//            "Weight": 10,
//            "Port": 8080,
//            "Target": "192.168.10.2."
//        }
//    ]
//  }
func (el *HttpServer) handleGetService(w http.ResponseWriter, r *http.Request) {
	var err error
	var records []dns.SRV
	//var recordsAsJSonString string
	var output JSonOut
	var found int
	var dataFromDataSource []communsTypes.KeyValueType

	w.Header().Add("Content-Type", "application/json")

	if el.dataGet == nil {
		output.ToOutput(0, errors.New("ben burkert dns plugin config error. please, define a getData function"), nil, w)
		return
	}

	if el.dataPut == nil {
		output.ToOutput(0, errors.New("ben burkert dns plugin config error. please, define a putData function"), nil, w)
		return
	}

	urlElements := strings.Split(r.Method+r.URL.Path, "/")

	serviceName := urlElements[2]

	err, found, dataFromDataSource = el.dataGet([]byte(el.servicePrefix + serviceName))
	if err != nil {
		w.WriteHeader(503)
		el.handleError(err)
		output.ToOutput(0, errors.New("internal server error"), nil, w)
		return
	}

	if found == 0 {
		output.ToOutput(0, nil, nil, w)
		return
	}

	err = json.Unmarshal([]byte(dataFromDataSource[0].V), &records)

	output.ToOutput(len(records), err, records, w)
}

// http post/put method function
// this method creates a new DNS record list ou append a new record in list.
//
//  Raw JSon data format
//  {
//    "port":     int,
//    "target":   string ended in point. ex.:"192.169.0.1." or "mongodb."
//  }
//
//  JSon output format:
//  {
//    "Meta": {
//        "TotalCount": 2,
//        "Success": true,
//        "Error": ""
//    },
//    "Objects": [
//        {
//            "Priority": 10,
//            "Weight": 10,
//            "Port": 8080,
//            "Target": "192.168.10.1."
//        },
//        {
//            "Priority": 10,
//            "Weight": 10,
//            "Port": 8080,
//            "Target": "192.168.10.2."
//        }
//    ]
//  }
func (el *HttpServer) handlePutService(w http.ResponseWriter, r *http.Request) {
	var err error
	var inData service
	var records []dns.SRV
	var found int
	var jsonData []byte
	var dataToSave []byte
	var output JSonOut
	var re *regexp.Regexp
	var dataFromDataSource []communsTypes.KeyValueType

	w.Header().Add("Content-Type", "application/json")

	if el.dataGet == nil {
		output.ToOutput(0, errors.New("ben burkert dns plugin config error. please, define a getData function"), nil, w)
		return
	}

	if el.dataPut == nil {
		output.ToOutput(0, errors.New("ben burkert dns plugin config error. please, define a putData function"), nil, w)
		return
	}

	urlElements := strings.Split(r.Method+r.URL.Path, "/")
	serviceName := urlElements[2]

	jsonData, err = ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(503)
		el.handleError(err)
		output.ToOutput(0, errors.New("internal server error"), nil, w)
		return
	}

	err = json.Unmarshal(jsonData, &inData)
	if err != nil {
		w.WriteHeader(503)
		output.ToOutput(0, errors.New("unmarshal incoming json from client side error: "+err.Error()), nil, w)
		return
	}

	err, found, dataFromDataSource = el.dataGet([]byte(el.servicePrefix + serviceName))
	if err != nil {
		w.WriteHeader(503)
		el.handleError(err)
		output.ToOutput(0, errors.New("internal server error"), nil, w)
		return
	}

	if (inData.Target == "" && inData.Port == 0) || inData.Target == "." {
		w.WriteHeader(503)
		el.handleError(err)
		output.ToOutput(0, errors.New("register data error. please, don't send blank data"), nil, w)
		return
	}

	if inData.Target == "" && inData.Port != 0 {
		addr := r.RemoteAddr
		if strings.HasPrefix(addr, "[::1]") {
			addr = "127.0.0.1"
		} else {
			re, err = regexp.Compile("^([0-9]{1,3}.[0-9]{1,3}.[0-9]{1,3}.[0-9]{1,3})(:.*)$")
			if err != nil {
				w.WriteHeader(503)
				el.handleError(err)
				output.ToOutput(0, errors.New("internal server error"), nil, w)
				return
			}
			addr = string(re.ReplaceAll([]byte(addr), []byte("$1")))

		}
		inData.Target = addr + "."
	}

	if found == 0 {
		records = []dns.SRV{
			{Target: inData.Target, Port: inData.Port, Priority: 10, Weight: 10},
		}

		dataToSave, err = json.Marshal(&records)
		if err != nil {
			w.WriteHeader(503)
			el.handleError(err)
			output.ToOutput(0, errors.New("internal server error"), nil, w)
			return
		}

		dataToDataSource := communsTypes.KeyValueType{}
		dataToDataSource.K = []byte(el.servicePrefix + serviceName)
		dataToDataSource.V = dataToSave
		err = el.dataPut(dataToDataSource)
		if err != nil {
			w.WriteHeader(503)
			el.handleError(err)
			output.ToOutput(0, errors.New("internal server error"), nil, w)
			return
		}
	} else {

		err = json.Unmarshal(dataFromDataSource[0].V, &records)
		if err != nil {
			w.WriteHeader(503)
			el.handleError(err)
			output.ToOutput(0, errors.New("internal server error"), nil, w)
			return
		}

		pass := true
		for _, record := range records {
			rTest := record.Get()
			if rTest.(*dns.SRV).Port == inData.Port && rTest.(*dns.SRV).Target == inData.Target {
				pass = false
			}
		}

		if pass == true {
			records = append(records, dns.SRV{Target: inData.Target, Port: inData.Port, Priority: 10, Weight: 10})

			dataToSave, err = json.Marshal(&records)
			if err != nil {
				w.WriteHeader(503)
				el.handleError(err)
				output.ToOutput(0, errors.New("internal server error"), nil, w)
				return
			}

			dataToDataSource := communsTypes.KeyValueType{}
			dataToDataSource.K = []byte(el.servicePrefix + serviceName)
			dataToDataSource.V = dataToSave
			err = el.dataPut(dataToDataSource)
			if err != nil {
				w.WriteHeader(503)
				el.handleError(err)
				output.ToOutput(0, errors.New("internal server error"), nil, w)
				return
			}
		}
	}

	output.ToOutput(len(records), nil, records, w)
}

func (el *HttpServer) externalIP() (string, error) {
	iFaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iFace := range iFaces {
		if iFace.Flags&net.FlagUp == 0 {
			continue
		}
		if iFace.Flags&net.FlagLoopback != 0 {
			continue
		}
		address, err := iFace.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range address {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String(), nil
		}
	}
	return "", errors.New("internet connection not found")
}

var PluginData HttpServer
