# Service Discover

```
[POST] localhost:8080/typeAAAA/service
```
Data:
```json
{
	"AAAA": "192.169.0.1"
}
```

Resposta:
```json
{
    "Type": "typeAAAA",
    "Data": [
        {
            "AAAA": "192.169.0.1"
        }
    ]
}
```



```
[POST] localhost:8080/typeSRV/services
```

Data:
```json
{
	"priority": 10,
	"weight":   10,
	"port":     8080,
	"target":   "192.168.10.1."
}
```

Resposta:
```json
{
    "Type": "typeSRV",
    "Data": [
        {
            "Priority": 0,
            "Weight": 0,
            "Port": 0,
            "Target": ""
        },
        {
            "Priority": 0,
            "Weight": 0,
            "Port": 0,
            "Target": ""
        },
        {
            "Priority": 0,
            "Weight": 0,
            "Port": 0,
            "Target": ""
        },
        {
            "Priority": 10,
            "Weight": 10,
            "Port": 8080,
            "Target": "192.168.10.1."
        }
    ]
}
```

Exemplo de saída:
```json
{
    "hosts": {
        "database": {
            "address": "10.0.0.1",
            "port": 3306
        },
        "cache": {
            "address": "10.0.0.2",
            "port": 6379
        }
    }
}
```

Código node:
```javascript
const { Resolver } = require('dns');

const resolver = new Resolver();
resolver.setServers(['127.0.0.1:53535']);
resolver.resolveSrv('service.tld.', function (err, addresses, family) {
    console.log('addresses:',addresses);
    console.log('family:',family);
});
```

Resposta:

addresses: 
```json
[ { "name": "192.168.0.1", "port": 8080, "priority": 0, "weight": 10 },
  { "name": "192.168.0.2", "port": 8081, "priority": 0, "weight": 50 } ]
```

family: undefined

```
```
```
```

> Você é livre para copiar e modificar este documento, desde que indique a fonte original da informação e 
submeta suas mudanças para que este documento seja melhorado



loading pluginData() at path ./plugin/dataPlugin/etcd/etcd.so
panic: codecgen version mismatch: current: 8, need 10. Re-generate file: /home/hkemper/Dropbox/gRPC/Libraries/src/github.com/coreos/etcd/client/keys.generated.go

cd /gRPC/Libraries/src/github.com/coreos/etcd$

export GO111MODULE=on

go get -v -u github.com/ugorji/go/codec/codecgen

pushd client
codecgen -o ./keys.generated.go ./keys.go
popd