package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

func get_downloads_dir_loc() (string, error) {
	curr_user, curr_user_err := user.Current()
	if curr_user_err != nil {
		return "", errors.New("failed to get the current user object")
	}
	return filepath.Join(curr_user.HomeDir, "Downloads"), nil
}

func open_file_w_default_app(app_loc string) error {
	var cmd *exec.Cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", app_loc) // loading the app location into the starter
	var run_err error = cmd.Run()                                                        // running the app (.nfp)
	return run_err                                                                       // will be nil if no error occurs
}

func check_if_file_is_nfp(file_loc string) bool {
	var file_name string = filepath.Base(file_loc)
	return filepath.Ext(file_name) == ".nfp"
}

func main() {
	// basing the logfile location off of the .exe's relative location
	main_exe_path, exe_err := os.Executable()
	if exe_err != nil {
		log.Println("error getting executable path:", exe_err.Error()) // can't yet print to logfile
		return
	}
	var main_exe_dir string = filepath.Dir(main_exe_path)

	// opening (or creating) the local logfile
	var logfile_loc string = filepath.Join(main_exe_dir, "logfile.log")
	logfile, log_open_err := os.OpenFile(logfile_loc, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if log_open_err != nil {
		log.Println("failed to open logfile:", log_open_err.Error())
		return
	}
	defer logfile.Close()
	log.SetOutput(logfile) // setting the output of the log to the provided logfile

	// getting the current users downloads folder
	downloads_loc, down_get_err := get_downloads_dir_loc()
	if down_get_err != nil {
		log.Println("failed to get the location of the downloads folder:", down_get_err.Error())
		return
	}

	// Create a new watcher
	watcher, watch_err := fsnotify.NewWatcher()
	if watch_err != nil {
		log.Println("failed to create the watcher obj:", watch_err.Error())
		return
	}
	defer watcher.Close()

	// Start a goroutine to handle incoming watcher events
	var last_opened_file string = ""
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					log.Println("incoming watcher event '!ok'")
					return
				}
				if (event.Op & fsnotify.Create) == fsnotify.Create {
					if last_opened_file != event.Name { // ensuring that file not opened twice due to temporary files in downloads
						if check_if_file_is_nfp(event.Name) { // returns true if the event name is an nfp
							log.Println("new file created:", event.Name)
							last_opened_file = event.Name
							var run_file_err error = open_file_w_default_app(event.Name)
							if run_file_err != nil {
								log.Println("failed to run file:", run_file_err.Error())
							} else {
								log.Println("ran file:", event.Name)
							}
						}
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					log.Println("incoming watcher event '!ok'")
					return
				}
				log.Println("error:", err.Error())
			}
		}
	}()

	// Add the Downloads folder to the watcher
	watch_add_err := watcher.Add(downloads_loc)
	if watch_add_err != nil {
		log.Println("failed to add the downloads folder to the watcher obj:", watch_add_err.Error())
		return
	}

	// program ready to respond to incoming files in watcher dir
	fmt.Println("SUCCESS: watcher active")
	log.Println("SUCCESS: watcher active")
	select {} // infinite loop
}
