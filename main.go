package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:      "spanch",
		Usage:     "Calls command when filesystem at specified path has been changed",
		Version:   "1.0",
		UsageText: `spanch help | -p . "system command with args"`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "path",
				Aliases: []string{"p"},
				Value:   ".",
				Usage:   "filesystem path to watch",
			},
		},
		Action: func(c *cli.Context) error {
			command := ""
			if c.NArg() > 0 {
				command = c.Args().Get(0)
			} else {
				cli.ShowAppHelp(c)
				return nil
			}

			path := c.String("path")

			err := watchAndExec(path, command)
			if err != nil {
				log.Println(err)
			}
			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func watchAndExec(path, command string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer watcher.Close()

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if !event.Has(fsnotify.Write) {
					continue
				}

				cmd := strings.Split(command, " ")[0]
				args := strings.Split(command, " ")

				out := exec.Command(cmd, strings.Join(args[1:], " "))
				outBuff, _ := out.CombinedOutput()
				log.Println(fmt.Sprintf(string(outBuff)))

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	walkErr := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			return nil
		}
		absPath, err := filepath.Abs(path)
		if err != nil {
			log.Fatal(err)
			return err
		}

		if strings.Contains(absPath, "/.") {
			return nil
		}

		err = watcher.Add(absPath)
		if err != nil {
			log.Fatal(err)
			return err
		}
		return nil
	})
	die(walkErr)

	log.Println("info: started watch at", path)

	<-make(chan struct{})
	return nil
}

func die(err error) {
	if err != nil {
		panic(err)
	}
}
