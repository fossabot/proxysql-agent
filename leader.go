package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/klog/v2"
)

func Leader(leaseName string) {
	//FIXME: figure out how to make this dynamic
	leaseNamespace := "proxysql"

	clientset := getKubeClient()

	run := func(ctx context.Context) {
		//FIXME: run whatever code in this loop
		slog.Info("Controller loop...")

		select {}
	}

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ch
		klog.Info("Received termination, signaling shutdown")
		cancel()
	}()

	id := os.Getenv("HOSTNAME")

	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock: &resourcelock.LeaseLock{
			LeaseMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-lock", leaseName),
				Namespace: leaseNamespace,
			},
			Client: clientset.CoordinationV1(),
			LockConfig: resourcelock.ResourceLockConfig{
				Identity: id,
			},
		},
		// IMPORTANT: you MUST ensure that any code you have that
		// is protected by the lease must terminate **before**
		// you call cancel. Otherwise, you could have a background
		// loop still running and another process could
		// get elected before your background loop finished, violating
		// the stated goal of the lease.
		ReleaseOnCancel: true,
		LeaseDuration:   60 * time.Second,
		RenewDeadline:   15 * time.Second,
		RetryPeriod:     5 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				// we're notified when we start - this is where you would
				// usually put your code
				run(ctx)
			},
			OnStoppedLeading: func() {
				// we can do cleanup here
				slog.Info("No longer leader", slog.String("id", id))
			},
			OnNewLeader: func(identity string) {
				// we're notified when new leader elected
				if identity == id {
					slog.Info("This pod was elected leader", slog.String("lease", leaseName))
				} else {
					slog.Info("New leader elected", slog.String("lease", leaseName), slog.String("leader", identity))
				}
			},
		},
	})
}

func getKubeClient() *kubernetes.Clientset {
	config, err := rest.InClusterConfig()
	if err != nil {
		slog.Error("Error detected", slog.Any("err", err))
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		slog.Error("Error detected", slog.Any("err", err))
	}

	return clientset
}
