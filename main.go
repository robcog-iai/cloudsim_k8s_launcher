package main 

import (
	"fmt"
	"os"
	"log"
	"strings"
	"time"
	"net/http"
	"encoding/json"
	"io/ioutil"
	"github.com/gorilla/mux"
	guuid "github.com/google/uuid"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	"agones.dev/agones/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

// Store client token
var clientToken map[string]string

// Knowrob server info
type KRClient struct {
	Port int     `json:"KRServerPort"`
	Protocol   string  `json:"KRProtocol"`
	LevelName  string  `json:"LevelName"`
}

func getAllGameServers(w http.ResponseWriter, r *http.Request) {
	// Check if token for the client already exists
	var clientIP string
	if r.RemoteAddr[:strings.Index(r.RemoteAddr, ":")] == "127.0.0.1" {
		hostIP := os.Getenv("HOST")
		clientIP = hostIP
	} else {
		clientIP = r.RemoteAddr[:strings.Index(r.RemoteAddr, ":")]
	}
	if _, ok := clientToken[clientIP]; !ok {
		fmt.Fprintf(w, "No server available\n")
	}

	config, err := rest.InClusterConfig()
	logger := runtime.NewLoggerWithSource("main")
	if err != nil {
		logger.WithError(err).Fatal("Could not create in cluster config")
	}

	// Access to the Agones resources through the Agones Clientset
	// Note that we reuse the same config as we used for the Kubernetes Clientset
	agonesClient, err := versioned.NewForConfig(config)
	if err != nil {
		fmt.Fprintf(w, fmt.Sprintf("Could not create the agones api clientset: %v\n", err))
	}
	selectorSet := make(map[string]string)
	selectorSet["client"] = clientToken[clientIP]
	labelSelector := labels.SelectorFromSet(selectorSet)
	
	options := metav1.ListOptions{ LabelSelector: labelSelector.String()}
	gsList, err := agonesClient.AgonesV1().GameServers("default").List(options)

	var ipList string
	for _, gs := range gsList.Items {
		ipList += gs.Status.Address + ":" + fmt.Sprint(int64(gs.Status.Ports[0].Port)) + "\n"
	}
	fmt.Fprintf(w, ipList)
}

//
func createGameServer(w http.ResponseWriter, r *http.Request) {
	
	mongoIP := os.Getenv("MONGO_IP")
	mongoPort := os.Getenv("MONGO_PORT")
	imageRepo := os.Getenv("IMAGE_REPO")

	// Check if token for the client already exists	
	var clientIP string
	if r.RemoteAddr[:strings.Index(r.RemoteAddr, ":")] == "127.0.0.1" {
		hostIP := os.Getenv("HOST")
		clientIP = hostIP
	} else {
		clientIP = r.RemoteAddr[:strings.Index(r.RemoteAddr, ":")]
	}
	reqBody, _ := ioutil.ReadAll(r.Body)
	var krClient KRClient 
	json.Unmarshal(reqBody, &krClient)

	// Check if token for the client already exists
	if _, ok := clientToken[clientIP]; !ok {
		clientToken[clientIP] = guuid.New().String()
	}
	
	config, err := rest.InClusterConfig()
	logger := runtime.NewLoggerWithSource("main")
	if err != nil {
		logger.WithError(err).Fatal("Could not create in cluster config")
	}

	// Access to the Agones resources through the Agones Clientset
	// Note that we reuse the same config as we used for the Kubernetes Clientset
	agonesClient, err := versioned.NewForConfig(config)
	if err != nil {
		fmt.Fprintf(w, fmt.Sprintf("Could not create the agones api clientset: %v\n", err))
	}

	// // Create a GameServer
	gs := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "ue-gs-", Namespace: "default",
			Labels: map[string]string { "client" : clientToken[clientIP] },
		},
		Spec: agonesv1.GameServerSpec{
			Container: "server",
			Ports: []agonesv1.GameServerPort{
				{
					ContainerPort: 80,
					Name:          "default",
					PortPolicy:    agonesv1.Dynamic,
					Protocol:      corev1.ProtocolTCP,
				},
			},
			Health: agonesv1.Health{
				InitialDelaySeconds : 120,
				PeriodSeconds       : 30,
			},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "server", Image: imageRepo+"/ue-signal-srv",
						},
						{
							Name: "proxy", Image: imageRepo+"/ue-proxy",
						},
						{
							Name: "ue-app", Image: imageRepo+"/"+strings.ToLower(krClient.LevelName), 
							Command: []string{"/home/ue4/project/RobCoG.sh"}, 
							Args: []string{"-AudioMixer", "-opengl4", "-NvEncFrameRateNum=1", "-KRServerIP="+clientIP, "KRServerPort="+fmt.Sprint(int64(krClient.Port)), "KRProtocol="+krClient.Protocol, "MongoServerIP="+mongoIP, "MongoServerPort="+mongoPort},
						},
					},
				},
			},
		},
	}
	fmt.Println("-KRServerIP="+clientIP+":"+fmt.Sprint(int64(krClient.Port)))
	newGS, err := agonesClient.AgonesV1().GameServers("default").Create(gs)
	if err != nil {
		fmt.Fprintf(w, "Unable to create GameServer: %v\n", err)
	}

	name := newGS.ObjectMeta.Name
	options := metav1.GetOptions{}
	for {
		gs, err := agonesClient.AgonesV1().GameServers("default").Get(name, options)
		if err != nil {
			fmt.Fprintf(w, fmt.Sprintf("Error updating gameserver: %v\n", err))
			return
		}
		switch gs.Status.State {
		case "Scheduled", "Ready", "Allocated":
			fmt.Fprintf(w, gs.Status.Address + ":" + fmt.Sprint(int64(gs.Status.Ports[0].Port)) + "\n")
			return
		case "Error", "Unhealthy", "Shutdown":
			fmt.Fprintf(w, "Error creating gameserver.\n")
			return
		default:
			time.Sleep(time.Second * 5)
		}
	}

}

func main() {
	// port := "9090"

	// if fromEnv := os.Getenv("PORT"); fromEnv != "" {
	// 	port = fromEnv
	// }
	port := os.Getenv("PORT")

	clientToken = make(map[string]string)

	// register hello function to handle all requests
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/game-server", createGameServer).Methods("POST")
	router.HandleFunc("/game-servers", getAllGameServers)

	// start the web server on port and accept requests
	log.Printf("Server listening on port %s", port)
	err := http.ListenAndServe(":"+port, router)
	log.Fatal(err)
}