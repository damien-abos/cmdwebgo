package main

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
)

func createHandler() (http.HandlerFunc, error) {
	fileBytes, err := ioutil.ReadFile("config.json")
	if err != nil {
		return nil, err
	} else {
		var config map[string]interface{}
		err = json.Unmarshal(fileBytes, &config)
		if err != nil {
			return nil, err
		} else {
			return func(w http.ResponseWriter, r *http.Request) {
				path := r.URL.Path[1:]
				command, ok := config[path].(map[string]interface{})
				if ok {

					log.Printf("Command %s", command)

					cmd, ok := command["cmd"].(string)
					if !ok {
						cmd = r.URL.Path[1:]
					}

					var arg []string = []string{}
					argv, ok := command["argv"].([]interface{})
					if ok {
						for _, a := range argv {
							v, ok := a.(string)
							if ok {
								arg = append(arg, v)
							}
						}
					}

					execCommand := exec.Command(cmd, arg...)

					var pipeReader, pipeWriter = io.Pipe()
					execCommand.Stdout = pipeWriter

					go writeCmdOutput(w, pipeReader)

					err := execCommand.Run()
					if err != nil {
						log.Fatal(err)
						http.Error(w, err.Error(), http.StatusInternalServerError)
					}
					pipeWriter.Close()
				} else {
					http.NotFound(w, r)
				}
			}, nil
		}
	}

}

func writeCmdOutput(w http.ResponseWriter, pipeReader *io.PipeReader) {
	var data []byte = make([]byte, 512)
	var n int
	var err error
	n, err = pipeReader.Read(data)
	if err == nil {
		w.Header().Add("Content-Type", "text/plain")
		for err == nil && n > 0 {
			w.Write(data[0:n])
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			} else {
				log.Println("Damn, no flush")
			}
			n, err = pipeReader.Read(data)
		}
		log.Println("End of Read")
		if err != io.EOF {
			log.Fatal(err)
		} else {
			log.Println("EOF reached")
		}
		pipeReader.Close()
	}
}

func main() {
	handler, err := createHandler()
	if err != nil {
		log.Fatal(err)
	} else {
		http.HandleFunc("/", handler)
		err := http.ListenAndServe(":9999", nil)
		if err != nil {
			log.Fatal(err)
		}
	}
}
