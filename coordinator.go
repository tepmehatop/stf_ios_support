package main

import (
	"encoding/json"
	"fmt"
	//"net"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
	"time"
	"html/template"
)

type Config struct {
	DeviceTrigger string `json:"device_trigger"`
	VideoEnabler string `json:"video_enabler"`
	SupportRoot string `json:"support_root"`
	MirrorFeedRoot string `json:"mirrorfeed_root"`
	WDARoot string `json:"wda_root"`
	CoordinatorHost string `json:"coordinator_host"`
	CoordinatorPort int `json:"coordinator_port"`
	WDAProxyPort string `json:"wda_proxy_port"`
	MirrorFeedPort int `json:"mirrorfeed_port"`
	Pipe string `json:"pipe"`
}

type DevEvent struct {
    action int
    uuid string
}

type RunningDev struct {
	uuid string
	name string
    mirror *os.Process
    ff     *os.Process
    proxy  *os.Process
}

type BaseProgs struct {
	trigger    *os.Process
	vidEnabler *os.Process
	stf        *os.Process
}

func main() {
	// Read in config
	configFile := "config.json"
	configFh, err := os.Open(configFile)   
    if err != nil {
        log.Panicf("failed reading file: %s", err)
    }
    defer configFh.Close()
    
	jsonBytes, _ := ioutil.ReadAll( configFh )
	var config Config
	json.Unmarshal( jsonBytes, &config )
	
	devEventCh := make( chan DevEvent, 2 )
	runningDevs := make( map [string] RunningDev )
	baseProgs := BaseProgs{}
	
	// start web server waiting for trigger http command for device connect and disconnect
	
	var listen_addr = fmt.Sprintf( "%s:%d", config.CoordinatorHost, config.CoordinatorPort ) // "localhost:8027"
	go startServer( devEventCh, listen_addr )
	
	// start the 'osx_ios_device_trigger'
	go func() {
		fmt.Printf("Starting osx_ios_device_trigger\n");
		triggerCmd := exec.Command( config.DeviceTrigger )
		
		//triggerOut, _ := triggerCmd.StdoutPipe()
		//triggerCmd.Stdout = os.Stdout
		//triggerCmd.Stderr = os.Stderr
		err := triggerCmd.Start()
		if err != nil {
			fmt.Println(err.Error())
		} else {
			baseProgs.trigger = triggerCmd.Process
		}
		/*for {
			line, err := ioutil.Read(triggerOut)
			if err != nil {
				break
			}
		}*/
		triggerCmd.Wait()
		fmt.Printf("Ended: osx_ios_device_trigger\n");
	}()
	
	// start the video enabler
	go func() {
		fmt.Printf("Starting video-enabler\n");
		enableCmd := exec.Command(config.VideoEnabler)
		err := enableCmd.Start()
		if err != nil {
			fmt.Println(err.Error())
			baseProgs.vidEnabler = nil
		} else {
			baseProgs.vidEnabler = enableCmd.Process 
		}
		enableCmd.Wait()
		fmt.Printf("Ended: video-enabler\n")
	}()
	
	// start stf and restart it when needed
	// TODO: if it doesn't restart / crashes again; give up
	go func() {
		for {
			fmt.Printf("Starting stf\n");
			stfCmd := exec.Command("/bin/bash", "run-stf.sh")
			err := stfCmd.Start()
			if err != nil {
				fmt.Println(err.Error())
				baseProgs.stf = nil
			} else {
				baseProgs.stf = stfCmd.Process
			}
			stfCmd.Wait()
			fmt.Printf("Ended:stf\n");
			// log out that it stopped
		}
	}()
	
	SetupCloseHandler( runningDevs, &baseProgs )
	
	/*go func() {
		// repeatedly check vpn connection
				
		// when vpn goes down
			// log an error
			// wait for it to come back up
			// restart the various things to use the new IP
	}*/

	//go func() {
		for {
			// receive message
			devEvent := <- devEventCh
			uuid := devEvent.uuid
			
			if devEvent.action == 0 { // device connect
				devd := RunningDev{}
				devd.uuid = uuid
				fmt.Printf("Setting up device uuid: %s\n", uuid)
				devd.name = getDeviceName( uuid )
				devName := devd.name
				fmt.Printf("Device name: %s\n", devName)
				
				// start mirrorfeed
				mirrorPort := config.MirrorFeedPort // 8000
				pipeName := config.Pipe
				fmt.Printf("Starting mirrorfeed\n");
				
				mirrorFeedBin := fmt.Sprintf( "%s/mirrorfeed/mirrorfeed", config.MirrorFeedRoot )
				
				mirrorCmd := exec.Command(mirrorFeedBin, strconv.Itoa( mirrorPort ), pipeName )
				mirrorCmd.Stdout = os.Stdout
				mirrorCmd.Stderr = os.Stderr
				go func() {
					err := mirrorCmd.Start()
					if err != nil {
						fmt.Println(err.Error())
						devd.mirror = nil
					} else {
						devd.mirror = mirrorCmd.Process
					}
					mirrorCmd.Wait()
					fmt.Printf("mirrorfeed ended\n")
					devd.mirror = nil
				}()
				
				// start ffmpeg
				fmt.Printf("Starting ffmpeg\n")
				
				halfresScript := fmt.Sprintf( "%s/mirrorfeed/halfres.sh", config.MirrorFeedRoot )
				
				ffCmd := exec.Command("/bin/bash", halfresScript, devName, pipeName )
				//ffCmd.Stdout = os.Stdout
				ffCmd.Stderr = os.Stderr
				go func() {
					err := ffCmd.Start()
					if err != nil {
						fmt.Println(err.Error())
						devd.ff = nil
					} else {
						devd.ff = ffCmd.Process
					}
					ffCmd.Wait()
					fmt.Printf("ffmpeg ended\n")
					devd.ff = nil
				}()
				
				time.Sleep( time.Second * 9 )
				
				// start wdaproxy
				wdaPort := config.WDAProxyPort // "8100"
				fmt.Printf("Starting wdaproxy\n")
				proxyCmd := exec.Command( "wdaproxy", "-p", wdaPort, "-d", "-W", config.WDARoot, "-u", uuid )
				proxyCmd.Stdout = os.Stdout
				proxyCmd.Stderr = os.Stderr
				go func() {
					err := proxyCmd.Start()
					if err != nil {
						fmt.Println(err.Error())
						devd.proxy = nil
					} else {
						devd.proxy = proxyCmd.Process
					}
					proxyCmd.Wait()
					fmt.Printf("wdaproxy ended\n")
				}()
				
				runningDevs[uuid] = devd
			}
			if devEvent.action == 1 { // device disconnect
				devd := runningDevs[uuid]
				closeRunningDev( devd )
			}
		}
	//}
}

func closeAllRunningDevs( runningDevs map [string] RunningDev ) {
	for _, devd := range runningDevs {
		closeRunningDev( devd )
	}
}

func closeRunningDev( devd RunningDev ) {
	// stop wdaproxy
	if devd.proxy != nil {
		fmt.Printf("Killing wdaproxy\n")
		devd.proxy.Kill()
	}
	
	// stop ffmpeg
	if devd.ff != nil {
		fmt.Printf("Killing ffmpeg\n")
		devd.ff.Kill()
	}
	
	// stop mirrorfeed
	if devd.mirror != nil {
		fmt.Printf("Killing mirrorfeed\n")
		devd.mirror.Kill()
	}
}

func closeBaseProgs( baseProgs *BaseProgs ) {
	if baseProgs.trigger != nil {
		fmt.Printf("Killing trigger\n")
		baseProgs.trigger.Kill()
	}
	if baseProgs.vidEnabler != nil {
		fmt.Printf("Killing vidEnabler\n")
		baseProgs.vidEnabler.Kill()
	}
	if baseProgs.stf != nil {
		fmt.Printf("Killing stf\n")
		baseProgs.stf.Kill()
	}
}

func SetupCloseHandler( runningDevs map [string] RunningDev, baseProgs *BaseProgs ) {
    c := make(chan os.Signal, 2)
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)
    go func() {
        <- c
        fmt.Println("\r- Ctrl+C pressed in Terminal")
        closeBaseProgs( baseProgs )
        closeAllRunningDevs( runningDevs )
        os.Exit(0)
    }()
}

func getDeviceName( uuid string ) (string) {
	name, _ := exec.Command( "idevicename", "-u", uuid ).Output()
	nameStr := string(name)
	nameStr = nameStr[:len(nameStr)-1]
	return nameStr
}
	
func startServer( devEventCh chan<- DevEvent, listen_addr string ) {
    fmt.Printf("Starting server\n");
    http.HandleFunc( "/", handleRoot )
    connectClosure := func( w http.ResponseWriter, r *http.Request ) {
    	deviceConnect( w, r, devEventCh )
    }
    disconnectClosure := func( w http.ResponseWriter, r *http.Request ) {
    	deviceDisconnect( w, r, devEventCh )
    }
    http.HandleFunc( "/dev_connect", connectClosure )
    http.HandleFunc( "/dev_disconnect", disconnectClosure )
    log.Fatal( http.ListenAndServe( listen_addr, nil ) )
}

func handleRoot( w http.ResponseWriter, r *http.Request ) {
    rootTpl.Execute( w, "ws://"+r.Host+"/echo" )
}

func deviceConnect( w http.ResponseWriter, r *http.Request, devEventCh chan<- DevEvent ) {
	// signal device loop of device connect
	devEvent := DevEvent{}
	devEvent.action = 0
	r.ParseForm()
	devEvent.uuid = r.Form.Get("uuid")
	devEventCh <- devEvent
}

func deviceDisconnect( w http.ResponseWriter, r *http.Request, devEventCh chan<- DevEvent ) {
	// signal device loop of device disconnect
	devEvent := DevEvent{}
	devEvent.action = 1
	r.ParseForm()
	devEvent.uuid = r.Form.Get("uuid")
	devEventCh <- devEvent
}

var rootTpl = template.Must(template.New("").Parse(`
<!DOCTYPE html>
<html>
	<head>
	</head>
	<body>
	test
	</body>
</html>
`))