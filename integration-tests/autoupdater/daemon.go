package autoupdater

import (
	_ "embed"
	"io/ioutil"
	"os/exec"
	"os"
	"path"
)

//go:embed configs.json
var configs string

func setupCerts(execDir string, caCert []byte, rootCert []byte, rootKey []byte) error {
	certsPath := path.Join(execDir, "certs")
	err := os.MkdirAll(certsPath, os.ModePerm)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path.Join(certsPath, "ca.pem"), caCert, 0644)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path.Join(certsPath, "root.pem"), rootCert, 0644)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path.Join(certsPath, "root.key"), rootKey, 0600)
	if err != nil {
		return err
	}

	return nil
}

func deleteCerts(execDir string) error {
    return os.RemoveAll(path.Join(execDir, "certs"))
}

func setupConfig(execDir string) error {
	return ioutil.WriteFile(path.Join(execDir, "configs.json"), []byte(configs), 0644)
}

func deleteConfig(execDir string) error {
	return os.Remove(path.Join(execDir, "configs.json"))
}

type LifecycleRequest struct {
	Type string
	Pid int
	Killed bool
}

//For strictly correct usage to avoid edge race conditions in a context requiring more rigor, a mutex would probably be needed 
//when restarting the process between getting the kill status, starting the process and setting the pid. The launch daemon would 
//need to respect that mutex. However, that race condition should never happen in those tests as the auto-updated is expected
//to restart only once in the middle of the tests
func daemon(binaryPath string, execDir string, lifecycleRequestChan chan LifecycleRequest, killStatusChan chan bool) {
	for {
		lifecycleRequestChan <- LifecycleRequest{Type: "killStatus"}
		killed := <- killStatusChan
		if killed {
			lifecycleRequestChan <- LifecycleRequest{Type: "finish"}
			break
		}

		cmd := exec.Command(binaryPath)
		cmd.Dir = execDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		
		err := cmd.Start()
		if err != nil {
			continue
		}
		lifecycleRequestChan <- LifecycleRequest{Type: "setPid", Pid: cmd.Process.Pid}
		cmd.Wait()
	}
}

func daemonLifecycle(lifecycleRequestChan chan LifecycleRequest, killStatusChan chan bool, finishChan chan struct{}) {
	var pid int
	killed := false
	for {
		message := <- lifecycleRequestChan
		if message.Type == "setPid" {
			pid = message.Pid
		} else if message.Type == "kill" {
			proc, err := os.FindProcess(pid)
			if err != nil {
				continue
			}
			proc.Kill()
			killed = true
		} else if message.Type == "killStatus" {
			killStatusChan <- killed
		} else if message.Type == "finish" {
			close(finishChan)
			return
		}
	}
}

func LaunchDaemon(binaryPath string, execDir string, caCert []byte, rootCert []byte, rootKey []byte, stop chan struct {}) error {
	err := setupCerts(execDir, caCert, rootCert, rootKey)
	if err != nil {
		return err
	}
	
	err = setupConfig(execDir)
	if err != nil {
		return err
	}

	lifecycleRequestChan := make(chan LifecycleRequest)
	killStatusChan := make(chan bool)
	finishChan := make(chan struct {})
	go daemonLifecycle(lifecycleRequestChan, killStatusChan, finishChan)
	go daemon(binaryPath, execDir, lifecycleRequestChan, killStatusChan)
	<- stop
	lifecycleRequestChan <- LifecycleRequest{Type: "kill"}
    <- finishChan

	err = deleteCerts(execDir)
	if err != nil {
		return err
	}

	return deleteConfig(execDir)
}