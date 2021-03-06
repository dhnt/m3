package daemon

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/dhnt/m3/internal"
	"github.com/takama/daemon"
)

const (
	// name of the service
	name        = "dhnt.m3"
	description = "M3 Service"
)

// dependencies that are NOT required by the service, but might be used
var dependencies = []string{}

var logger = internal.Logger()

// Service has embedded daemon
type Service struct {
	daemon.Daemon
}

// Install the service into the system
func (service *Service) Install(args ...string) (string, error) {
	logger.Printf("calling super Install os.Args: %v len: %v args: %v", os.Args, len(os.Args), args)

	return service.Daemon.Install(args...)
}

// Remove uninstalls the service and all corresponding files from the system
func (service *Service) Remove() (string, error) {
	logger.Printf("calling super Remove os.Args: %v len: %v", os.Args, len(os.Args))
	_, err := service.Daemon.Status()
	if err != nil {
		service.Daemon.Stop()
	}
	return service.Daemon.Remove()
}

// Start the service
func (service *Service) Start() (string, error) {
	logger.Printf("calling super Start os.Args: %v len: %v", os.Args, len(os.Args))
	return service.Daemon.Start()
}

// // Stop the service
// func (service *Service) Stop() (string, error) {
// 	logger.Printf("calling super Stop os.Args: %v len: %v", os.Args, len(os.Args))
// 	return service.Daemon.Stop()
// }

// // Status - check the service status
// func (service *Service) Status() (string, error) {
// 	logger.Printf("calling super status os.Args: %v len: %v", os.Args, len(os.Args))
// 	return service.Daemon.Status()
// }

// Manage by daemon commands or run the daemon
func (service *Service) Manage() (string, error) {
	logger.Printf("Manage args: %v len: %v", os.Args, len(os.Args))

	usage := "Usage: m3 install --base <dhnt_base>| uninstall | start | stop | status"
	if len(os.Args) < 2 {
		return usage, nil
	}
	installCmd := flag.NewFlagSet("install", flag.ExitOnError)
	runCmd := flag.NewFlagSet("run", flag.ExitOnError)

	command := os.Args[1]
	switch command {
	case "install":
		bp := installCmd.String("base", "", "dhnt base")
		installCmd.Parse(os.Args[2:])
		if *bp == "" {
			return usage, nil
		}
		os.Args[1] = "run"
		return service.Install(os.Args[1:]...)
	case "uninstall":
		return service.Remove()
	case "start":
		return service.Start()
	case "stop":
		return service.Stop()
	case "status":
		return service.Status()
	case "run":
		break
	default:
		return usage, nil
	}

	logger.Printf("Manage set up args: %v len: %v", os.Args, len(os.Args))

	//
	bp := runCmd.String("base", "", "dhnt base")
	runCmd.Parse(os.Args[2:])
	if *bp == "" {
		return usage, nil
	}
	base := *bp
	logger.Println("dhnt base:", base)
	//
	if err := os.Chdir(base); err != nil {
		logger.Println(err)
		os.Exit(1)
	}
	//
	signal.Ignore(syscall.SIGHUP)

	// Set up channel on which to send signal notifications.
	// We must use a buffered channel or risk missing the signal
	// if we're not ready to receive when the signal is sent.

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, os.Kill, syscall.SIGTERM)

	//
	script := filepath.Join(base, "etc/init.sh")
	done := make(chan error, 1)
	go func() {
		done <- internal.Execute(base, "bin/gsh", script)
	}()

	select {
	case err := <-done:
		return fmt.Sprintf("script exited: %v", script), err
	case killSignal := <-interrupt:
		logger.Println("Got signal:", killSignal)
		// logger.Println("Stoping listening on ", s.Addr())
		// s.Stop()
		// es.Stop()

		if killSignal == os.Interrupt {
			return "Daemon was interupted by system signal", nil
		}
		return "Daemon was killed", nil
	}
}

// Run daemon service
func Run() {
	logger.Printf("Run args: %v len: %v", os.Args, len(os.Args))

	srv, err := daemon.New(name, description, dependencies...)
	if err != nil {
		logger.Println("Error: ", err)
		os.Exit(1)
	}
	service := &Service{
		Daemon: srv,
	}

	logger.Printf("Calling Manage service: %v", service)

	status, err := service.Manage()
	if err != nil {
		logger.Println(status, "\nError: ", err)
		os.Exit(1)
	}
	logger.Println(status)
}
