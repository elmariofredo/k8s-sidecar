/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Note: the example only works with the code within the same release/branch.
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sysincz/k8s-sidecar/cmd/sidecar/config"
	"github.com/sysincz/k8s-sidecar/cmd/sidecar/template"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	logrus "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/homedir"
	//
	// Uncomment to load all auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth
	//
	// Or uncomment to load specific auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/openstack"
)

//Version from build
var Version string

//Commit from build
var Commit string

//Branch from build
var Branch string

//BuildDate from build
var BuildDate string

var (
	//conf       config.Config
	log        = logrus.WithFields(logrus.Fields{"logger": "main"})
	configFile = flag.String("config", "/config/sidecar.yaml", "The Snmptrapper configuration file")
	debug      = flag.Bool("debug", false, "Set Log to debug level and print as text")
)

func main() {
	//logrus.SetLevel(logrus.InfoLevel)
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})
	log.Infof("Start  Version: %s, Commit %s, Branch %s,BuildDate %s", Version, Commit, Branch, BuildDate)
	var tmpOut string
	var lastOut string
	var kubeconfig *string

	//var mMap = make(map[string]map[string]string)
	var eMap = make(map[string]Event)

	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()
	var conf *config.Config
	log.Infof("Load config ... ")

	conf, _, err := config.LoadConfigFile(*configFile)
	if err != nil {
		log.Errorf("Error loading configuration: %s", err)
		panic(err)
	}
	if conf.CheckSelfConfig {
		go checkConfig(*configFile)
	}

	//monitoring start
	http.Handle(conf.PrometheusMetricsURL, promhttp.Handler())
	port := fmt.Sprintf(":%d", conf.PrometheusMetricsPort)
	log.Infof("Start http server for Prometheus '0.0.0.0:%d%s'", conf.PrometheusMetricsPort, conf.PrometheusMetricsURL)
	go http.ListenAndServe(port, nil)

	// create the clientset
	clientset, err := getClient(*kubeconfig) //kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	event := make(chan Event)
	for _, selector := range conf.Selectors {
		sel := strings.Split(selector, "/")
		if len(sel) != 2 {
			panic("wrong config for selector" + selector)
		}
		kind := sel[0]
		labelSelector := sel[1]
		listOptions := metav1.ListOptions{
			LabelSelector: labelSelector,
			Limit:         100,
		}

		fromNamespace := ""
		if conf.FromNamespace != "ALL" {
			fromNamespace = getNamespace(conf.FromNamespace)
		}

		switch kind {
		case "configmap":
			go watchConfigMap(*clientset, fromNamespace, listOptions, event)
		case "secret":
			go watchSecret(*clientset, fromNamespace, listOptions, event)
		default:
			panic("uknow kind:" + kind)
		}

		//sleep (download main first-  problem with download additional info before main)
		time.Sleep(5 * time.Second)
	}
	for {
		event, CMok := <-event
		log.Debugln("Received ", event.cmid, CMok)
		cmid := event.cmid
		eMap[cmid] = event

		if conf.Template != "" {
			tmpOut = validOutput(*conf, cmid, eMap)
			if lastOut != tmpOut {

				tmpDir := conf.ToDirectory
				fileName := conf.ToFileName
				createDir(tmpDir)
				writeToFile(tmpDir+fileName, tmpOut)
				log.Infof("Changed write to File %s", tmpDir+fileName)

				namespace := getNamespace(conf.ToNamespace)

				stringData := map[string]string{
					conf.ToFileName: tmpOut,
				}
				if conf.ToSecretName != "" {
					log.Infof("Changed write to Secret %s/%s", namespace, conf.ToSecretName)
					writeToSecret(*clientset, namespace, conf.ToSecretName, stringData)
				}
				if conf.ToConfigMapName != "" {
					log.Infof("Changed write to ConfigMap %s/%s", namespace, conf.ToSecretName)
					writeToConfigMap(*clientset, namespace, conf.ToConfigMapName, stringData)
				}
				urlReloads(*conf)
				lastOut = tmpOut
			}

		} else {

			dir := conf.ToDirectory

			for cmid := range eMap {

				in := make(map[string]string)
				in["namespace"] = eMap[cmid].namespace
				finDir := RunTemplate(dir, in)
				if finDir == "" {
					panic("wrong template 'ToDirectory' ")
				}
				log.Infof("Rename dir: %s to %s", dir, finDir)
				for _, ent := range eMap[cmid].entry {

					if eMap[cmid].action == "deleted" {
						log.Infof("Delete file %s (cmid:%s deleted)", finDir+ent.name, cmid)
						deleteFile(finDir + ent.name)
					} else {
						createDir(finDir)
						log.Debugf("cmid: '%s' name: '%s' len: %d ", cmid, ent.name, len(ent.data))
						if validData(*conf, eMap, cmid, ent.data) {
							writeToFile(finDir+ent.name, ent.data)
						}

					}
				}
				urlReloads(*conf)

				if eMap[cmid].action == "deleted" {
					delete(eMap, cmid)
				}

			}
		}

	}

}
func urlReloads(myConfig config.Config) {
	for _, u := range myConfig.URLRealoads {
		MakeHTTPRequest(u)
	}
}
func e2map(eMap map[string]Event) map[string]map[string]string {

	var mMap = make(map[string]map[string]string)

	for cmid, event := range eMap {
		if event.action != "deleted" {
			if mMap[cmid] == nil {
				mMap[cmid] = map[string]string{}
			}

			for _, ent := range event.entry {
				if mMap[cmid][ent.name] != ent.data {
					mMap[cmid][ent.name] = ent.data
				}
			}
		}
	}
	return mMap
}
func validOutput(myConfig config.Config, cmid string, eMap map[string]Event) (tmpOut string) {

	mMap := e2map(eMap)

	tmpOut = createOutput(myConfig, mMap)
	if checkSyntax(myConfig, tmpOut) {
		log.Info("Syntax OK ", cmid)
		sidecarSyntaxOk.WithLabelValues(eMap[cmid].namespace, cmid).Set(1)
	} else {
		log.Warn("INVALID syntax: ", cmid)
		sidecarSyntaxOk.WithLabelValues(eMap[cmid].namespace, cmid).Set(0)
		delete(eMap, cmid)
		delete(mMap, cmid)
		tmpOut = createOutput(myConfig, mMap)
	}
	return
}

func validData(myConfig config.Config, eMap map[string]Event, cmid string, tmpIn string) (valid bool) {

	if checkSyntax(myConfig, tmpIn) {
		log.Info("Syntax OK ", cmid)
		sidecarSyntaxOk.WithLabelValues(eMap[cmid].namespace, cmid).Set(1)
		return true
	}
	log.Warn("INVALID syntax: ", cmid)
	sidecarSyntaxOk.WithLabelValues(eMap[cmid].namespace, cmid).Set(0)
	return false

}

func checkSyntax(myConfig config.Config, tmpOut string) bool {

	if myConfig.CheckYaml {
		log.Debug("checkSyntax - CheckYaml")
		val := checkYaml(tmpOut)
		if !val {
			return false
		}
	}

	if myConfig.CheckJSON {
		log.Debug("checkSyntax - CheckJSON")
		val := checkJSON(tmpOut)
		if !val {
			return false
		}
	}

	if myConfig.CheckCommand != "" {
		log.Debug("checkSyntax - CheckCommand")
		val := false
		f := myConfig.TmpDirectory + myConfig.ToFileName
		writeToFile(f, tmpOut)
		exitCode := RunCommand(myConfig.CheckCommand)
		deleteFile(f)
		for _, code := range myConfig.CheckCommandOKExitCode {
			log.Debugf("Test exit codes compare %d (CheckCommandOKExitCode) vs %d (exitCode)", code, exitCode)
			if code == exitCode {
				val = true
			}
		}
		if !val {
			return false
		}
	}

	return true
}

func createOutput(myConfig config.Config, data interface{}) string {
	tmpl := myConfig.Template
	tmplOut := RunTemplate(tmpl, data)
	if myConfig.RemoveComment {
		tmplOut = removeComments(tmplOut)
	}
	if myConfig.RemoveEmptyLines {
		tmplOut = removeEmptyLines(tmplOut)
	}

	return tmplOut
}
func createDir(dirname string) {
	_ = os.MkdirAll(dirname, os.ModePerm)
}

//RunTemplate translate template string to string + trimSpace
func RunTemplate(text string, data interface{}) string {
	tmpl := template.Init()

	value, err := tmpl.Execute(text, data)
	if err != nil {
		log.Errorf("Error loading templates from %s: %s", text, err)
		return ""
	}
	//value = strings.TrimSpace(value)
	return value
}
