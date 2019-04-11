package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"syscall"

	"gopkg.in/yaml.v2"
)

func checkYaml(s string) bool {
	var ya interface{}
	err := yaml.Unmarshal([]byte(s), &ya)
	if err != nil {
		log.Warn(err)
		return false
	}
	return true
}

func checkJSON(s string) bool {
	var js interface{}
	err := json.Unmarshal([]byte(s), &js)
	if err != nil {
		log.Warn(err)
		return false
	}
	return true

}

func removeEmptyLines(str string) string {
	regex, err := regexp.Compile("\n[\t\n\f\r ]+\n")
	if err != nil {
		return str
	}
	str = regex.ReplaceAllString(str, "\n")
	return str
}

func removeComments(str string) string {
	regex, err := regexp.Compile("#.*")
	if err != nil {
		return str
	}
	str = regex.ReplaceAllString(str, "\n")
	return str
}

func deleteFile(path string) {
	// delete file
	var err = os.Remove(path)
	if err != nil {
		log.Error(err)
		return
	}

	log.Debug("Deleted ", path)
}
func writeToFile(filepath, data string) {
	log.Debugf("Write to file %s", filepath)
	f, err := os.Create(filepath)
	if err != nil {
		log.Error(err)
		return
	}
	l, err := f.WriteString(data)
	if err != nil {
		log.Error(err)
		f.Close()
		return
	}
	log.Debugf("%d bytes written successfully", l)
}

// RunCommand param command return exit code
func RunCommand(command string) (exitCode int) {
	args, err := parseCommandLine(command)
	if err != nil {
		log.Error(err)
	}
	cmd := exec.Command(args[0], args[1:]...)
	cmdOutput := &bytes.Buffer{}
	cmd.Stdout = cmdOutput
	cmdErrOutput := &bytes.Buffer{}
	cmd.Stderr = cmdErrOutput

	var waitStatus syscall.WaitStatus
	log.Debugf("Command: %v", cmd.Args)
	if err := cmd.Run(); err != nil {
		if err != nil {
			log.Debugf("Error: '%s'\n", err.Error())
		}
		if exitError, ok := err.(*exec.ExitError); ok {
			waitStatus = exitError.Sys().(syscall.WaitStatus)
			exitCode = waitStatus.ExitStatus()
			log.Debugf("exitCode: %s\n", []byte(fmt.Sprintf("%d", waitStatus.ExitStatus())))
		}
	} else {
		// Success
		waitStatus = cmd.ProcessState.Sys().(syscall.WaitStatus)
		exitCode = waitStatus.ExitStatus()
		log.Debugf("exitCode: %s\n", []byte(fmt.Sprintf("%d", waitStatus.ExitStatus())))
	}

	errO := string(cmdErrOutput.Bytes())
	if errO != "" {
		log.Infof("StdOut: %v", string(cmdOutput.Bytes()))
		log.Infof("ErrOut: %v", errO)
		return
	}

	log.Debugf("StdOut: %v", string(cmdOutput.Bytes()))
	return
}

func parseCommandLine(command string) ([]string, error) {
	var args []string
	state := "start"
	current := ""
	quote := "\""
	escapeNext := true
	for i := 0; i < len(command); i++ {
		c := command[i]

		if state == "quotes" {
			if string(c) != quote {
				current += string(c)
			} else {
				args = append(args, current)
				current = ""
				state = "start"
			}
			continue
		}

		if escapeNext {
			current += string(c)
			escapeNext = false
			continue
		}

		if c == '\\' {
			escapeNext = true
			continue
		}

		if c == '"' || c == '\'' {
			state = "quotes"
			quote = string(c)
			continue
		}

		if state == "arg" {
			if c == ' ' || c == '\t' {
				args = append(args, current)
				current = ""
				state = "start"
			} else {
				current += string(c)
			}
			continue
		}

		if c != ' ' && c != '\t' {
			state = "arg"
			current += string(c)
		}
	}

	if state == "quotes" {
		return []string{}, fmt.Errorf("Unclosed quote in command line: %s", command)
	}

	if current != "" {
		args = append(args, current)
	}

	return args, nil
}

func hashFileMd5(filePath string) (string, error) {
	var returnMD5String string
	file, err := os.Open(filePath)
	if err != nil {
		return returnMD5String, err
	}
	defer file.Close()
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return returnMD5String, err
	}
	hashInBytes := hash.Sum(nil)[:16]
	returnMD5String = hex.EncodeToString(hashInBytes)
	return returnMD5String, nil

}

//MakeHTTPRequest make HTTP request to url
func MakeHTTPRequest(url string) string {
	log.Infoln("Call HTTP:", url)
	resp, err := http.Get(url)
	if err != nil {
		log.Warnln(err)
		return ""
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Warnln(err)
		return ""
	}

	log.Infoln(string(body))
	return string(body)
}
