# cloudsim_k8s_launcher

cloudsim_k8s_launcher is responsible for creating and closing k8s resources containing unreal engine pixel streaming application in the cluster

## Deploy for CloudSim

1. Edit the dockerfile to setup the environment variable (important)
  * PORT - the port the launcher listen to(no need to change)
  * HOST - the ip address of the host running the k8s cluster. The host runs the cloudsim_k8s_launcher.
  * MONGO_IP - ip address of the mongodb for keeping world state data, usually the same host \
  * MONGO_PORT - port of mongodb
  * IMAGE_REPO - when launcher create the resource, it will pull images from the docker hub. This specify where to pull the image. Default: robcog or xiaojunll 

2. Build the image
```
docker build -t robcog/cloudsim_k8s_launcher .
docker push robcog/cloudsim_k8s_launcher
```

3. Deploy the image, the launcher will listen to port 30002, you can change that in cloudsim_k8s_launcher.yaml
```
kubectl apply -f cloudsim_k8s_launcher.yaml
```

4. How to use the launcher
Send HTTP POST request to the launcher with JSON data, one does not need to send post request directly, use the Prolog query in `knowrob_ameva`:
```
ag_create_clients(Num, LevelName, ClientId) 
```
once the launcher receives the request from `knowrob_ameva`, it will create an unreal engine application in k8s, and pass the ip address of `knowrob`, so the unreal engine application can connect to `knowrob_ameva`


JSON data:
```
{
	'KRServerPort' : xxx,             // it depends on the port  that knowrob listen to
	                                  // ue_start_src(Port) knowrob use this to specify port	                                      
	'KRProtocol' : 'kr_websocket',   // default, you can change the value in knowrob_ameva
	'LevelName' : xxxx                 // the image name of the unreal engine image, it is usually named with level name
}
```
PS: there is no need to pass the ip of the knowrob_ameva

5. configuration in `knowrob_ameva`
in `knowrob_ameva`, `ag_create_clients(Num, LevelName, ClientId) ` is used to post request to the launcher to create k8s resources. After deploying the launcher, one needs to configure the ip address and port of the launcher in `ag_conn.cpp`
