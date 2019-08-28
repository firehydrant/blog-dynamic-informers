package main

import (
	"os"
	"os/signal"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	cfg, err := restConfig()
	if err != nil {
		logrus.WithError(err).Fatal("could not get config")
	}

	// Grab a dynamic interface that we can create informers from
	dc, err := dynamic.NewForConfig(cfg)
	if err != nil {
		logrus.WithError(err).Fatal("could not generate dynamic client for config")
	}

	// Create a factory object that we can say "hey, I need to watch this resource"
	// and it will give us back an informer for it
	f := dynamicinformer.NewFilteredDynamicSharedInformerFactory(dc, 0, v1.NamespaceAll, nil)

	// Retrieve a "GroupVersionResource" type that we need when generating our informer from our dynamic factory
	gvr, _ := schema.ParseResourceArg("deployments.v1.apps")

	// Finally, create our informer for deployments!
	i := f.ForResource(*gvr)

	stopCh := make(chan struct{})
	go startWatching(stopCh, i.Informer())

	sigCh := make(chan os.Signal, 0)
	signal.Notify(sigCh, os.Kill, os.Interrupt)

	<-sigCh
	close(stopCh)
}

func restConfig() (*rest.Config, error) {
	kubeCfg, err := rest.InClusterConfig()
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		kubeCfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	}

	if err != nil {
		return nil, err
	}

	return kubeCfg, nil
}

func startWatching(stopCh <-chan struct{}, s cache.SharedIndexInformer) {
	handlers := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			u := obj.(*unstructured.Unstructured)

			logrus.WithFields(logrus.Fields{
				"name":      u.GetName(),
				"namespace": u.GetNamespace(),
				"labels":    u.GetLabels(),
			}).Info("received add event!")
		},
		UpdateFunc: func(oldObj, obj interface{}) {
			logrus.Info("received update event!")
		},
		DeleteFunc: func(obj interface{}) {
			logrus.Info("received update event!")
		},
	}

	s.AddEventHandler(handlers)
	s.Run(stopCh)
}
