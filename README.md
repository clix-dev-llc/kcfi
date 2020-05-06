# `kcfi` - Codefresh Installer for Kubernetes  

### Download
https://github.com/codefresh-io/kcfi/releases

### Usage
Create configuration directory
```
kcfi init <product> [-d /path/to/stage-dir]
```
Edit configuration in config.yaml and deploy to Kubernetes
```
kcfi deploy [ -c config.yaml ] [ --kube-context <kube-context-name> ] [ --atomic ] [ --debug ] [ helm upgrade parameters ]
```

### Example - Codefresh onprem installation
```
kcfi init codefresh
```
It creates `codefresh` directory with config.yaml and other files

- Edit `codefresh/config.yaml` - set global.appUrl and other parameters  
- Set docker registry credentials - obtain sa.json from Codefesh or set your private registry address and credentials  
- Set tls certifcates (optional) - set tls.selfSigned=false put ssl.crt and private.key into certs/ directory  

Deploy Codefresh
```
kcfi deploy -c codefresh/config.yaml [ --kube-context <kube-context-name> ] [ --atomic ] [ --debug ] [ helm upgrade parameters ]
```

