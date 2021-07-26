package main

import (
	"fmt"
	yml "github.com/helmutkemper/reverseProxyMicroService/yaml"
	"log"
)

func main() {

	var err error

	fmt.Printf("template_0.yml\n")
	c := yml.Compose{}
	err = c.Unmarshal("Libraries/src/github.com/helmutkemper/reverseProxyMicroService/templates/template_0.yml")
	if err != nil {
		log.Fatalf(err.Error())
	}

	fmt.Printf("template_1.yml\n")
	c = yml.Compose{}
	err = c.Unmarshal("Libraries/src/github.com/helmutkemper/reverseProxyMicroService/templates/template_1.yml")
	if err != nil {
		log.Fatalf(err.Error())
	}

	fmt.Printf("template_2.yml\n")
	c = yml.Compose{}
	err = c.Unmarshal("Libraries/src/github.com/helmutkemper/reverseProxyMicroService/templates/template_2.yml")
	if err != nil {
		log.Fatalf(err.Error())
	}
}
