package main

import (
	"io/ioutil"
	"os"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func getClient(pathToConfig string) (*kubernetes.Clientset, error) {
	var config *rest.Config
	var err error
	if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		config, err = rest.InClusterConfig()

	} else {
		if pathToConfig != "" {
			config, err = clientcmd.BuildConfigFromFlags("", pathToConfig)
		}
	}
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}
func getNamespace(ns string) string {
	log.Debugf("namespace: %s", ns)
	if ns != "" {
		return ns
	}
	// get namespace from within the pod. if found, return that
	if namespacePod, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		return string(namespacePod)

	}
	return "default"

}

func watchConfigMap(clientset kubernetes.Clientset, namespace string, listOptions metav1.ListOptions, ev chan Event) {
	//tmi test
	watcher, err := clientset.CoreV1().ConfigMaps(namespace).Watch(listOptions)
	if err != nil {
		panic(err)
	}

	for event := range watcher.ResultChan() {

		cm := event.Object.(*v1.ConfigMap)
		cmid := cm.Namespace + "/" + cm.GetName()
		log.Debug(cmid)
		for key, val := range cm.Labels {
			log.Debugf("   Labels: %s=%s", key, val)
		}
		e := Event{}
		e.cmid = cmid
		e.namespace = cm.Namespace
		switch event.Type {
		case watch.Deleted:
			e.action = "deleted"
		case watch.Added:
			e.action = "added"
		case watch.Modified:
			e.action = "modified"

		default:
			panic("unexpected event type " + event.Type)
		}
		var output []Entry
		for dataKey, dataValue := range cm.Data {
			log.Debugf("      dataKey: %s", dataKey)
			var ent Entry

			ent.data = string(dataValue)
			ent.name = dataKey
			output = append(output, ent)

		}
		e.entry = output
		ev <- e

	}
}

func watchSecret(clientset kubernetes.Clientset, namespace string, listOptions metav1.ListOptions, ev chan Event) {
	watcher, err := clientset.CoreV1().Secrets(namespace).Watch(listOptions)
	if err != nil {
		panic(err)
	}

	for event := range watcher.ResultChan() {

		cm := event.Object.(*v1.Secret)
		cmid := cm.Namespace + "/" + cm.GetName()
		log.Debug(cmid)
		for key, val := range cm.Labels {
			log.Debugf("   Labels: %s=%s", key, val)
		}
		e := Event{}
		e.cmid = cmid
		e.namespace = cm.Namespace
		switch event.Type {
		case watch.Deleted:
			e.action = "deleted"
		case watch.Added:
			e.action = "added"
		case watch.Modified:
			e.action = "modified"

		default:
			panic("unexpected event type " + event.Type)
		}

		var output []Entry
		for dataKey, dataValue := range cm.Data {
			log.Debugf("      dataKey: %s", dataKey)
			var ent Entry

			ent.data = string(dataValue)
			ent.name = dataKey
			output = append(output, ent)

		}
		e.entry = output
		ev <- e

	}
}
func writeToSecret(clientset kubernetes.Clientset, ns string, name string, stringData map[string]string) {

	_, err := clientset.CoreV1().Secrets(ns).Get(name, metav1.GetOptions{})
	if err == nil {
		_, err := clientset.CoreV1().Secrets(ns).Update(&v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			StringData: stringData,
		})
		if err != nil {
			log.Error(err)
			return
		}
		log.Infof("Updated Secret: %s/%s", ns, name)
		return
	}

	_, err = clientset.CoreV1().Secrets(ns).Create(&v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		StringData: stringData,
	})
	if err != nil {
		log.Error(err)
		return
	}
	log.Infof("Created Secret: %s/%s", ns, name)

	return

}

func writeToConfigMap(clientset kubernetes.Clientset, ns string, name string, stringData map[string]string) {

	_, err := clientset.CoreV1().ConfigMaps(ns).Get(name, metav1.GetOptions{})
	if err == nil {
		_, err := clientset.CoreV1().ConfigMaps(ns).Update(&v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			Data: stringData,
		})
		if err != nil {
			log.Error(err)
			return
		}
		log.Infof("Updated ConfigMap: %s/%s", ns, name)
		return
	}

	_, err = clientset.CoreV1().ConfigMaps(ns).Create(&v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Data: stringData,
	})
	if err != nil {
		log.Error(err)
		return
	}
	log.Infof("Created ConfigMap: %s/%s", ns, name)
	return

}
