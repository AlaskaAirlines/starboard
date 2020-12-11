package operator

import (
	"context"
	"errors"
	"fmt"

	"github.com/AlaskaAirlines/starboard/pkg/trivy"

	"github.com/AlaskaAirlines/s/starboard/pkg/ext"
	"github.com/AlaskaAirlines/s/starboard/pkg/operator/aqua"
	"github.com/AlaskaAirlines/s/starboard/pkg/operator/controller"
	"github.com/AlaskaAirlines/s/starboard/pkg/operator/controller/job"
	"github.com/AlaskaAirlines/s/starboard/pkg/operator/controller/pod"
	"github.com/AlaskaAirlines/s/starboard/pkg/operator/etc"
	"github.com/AlaskaAirlines/s/starboard/pkg/operator/logs"
	"github.com/AlaskaAirlines/s/starboard/pkg/starboard"
	"github.com/AlaskaAirlines/s/starboard/pkg/vulnerabilityreport"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	setupLog = log.Log.WithName("operator")
)

func Run(buildInfo starboard.BuildInfo, operatorConfig etc.Config) error {
	setupLog.Info("Starting operator", "buildInfo", buildInfo)

	// Validate configured namespaces to resolve install mode.
	operatorNamespace, err := operatorConfig.Operator.GetOperatorNamespace()
	if err != nil {
		return fmt.Errorf("getting operator namespace: %w", err)
	}

	targetNamespaces := operatorConfig.Operator.GetTargetNamespaces()

	installMode, err := operatorConfig.Operator.GetInstallMode()
	if err != nil {
		return fmt.Errorf("getting install mode: %w", err)
	}
	setupLog.Info("Resolving install mode", "install mode", installMode,
		"operator namespace", operatorNamespace,
		"target namespaces", targetNamespaces)

	// Set the default manager options.
	options := manager.Options{
		Scheme:                 starboard.NewScheme(),
		MetricsBindAddress:     operatorConfig.Operator.MetricsBindAddress,
		HealthProbeBindAddress: operatorConfig.Operator.HealthProbeBindAddress,
	}

	switch installMode {
	case etc.InstallModeOwnNamespace:
		// Add support for OwnNamespace set in STARBOARD_NAMESPACE (e.g. marketplace) and STARBOARD_TARGET_NAMESPACES (e.g. marketplace)
		setupLog.Info("Constructing single-namespaced cache", "namespace", targetNamespaces[0])
		options.Namespace = targetNamespaces[0]
	case etc.InstallModeSingleNamespace:
		// Add support for SingleNamespace set in STARBOARD_NAMESPACE (e.g. marketplace) and STARBOARD_TARGET_NAMESPACES (e.g. foo)
		cachedNamespaces := append(targetNamespaces, operatorNamespace)
		setupLog.Info("Constructing multi-namespaced cache", "namespaces", cachedNamespaces)
		options.Namespace = targetNamespaces[0]
		options.NewCache = cache.MultiNamespacedCacheBuilder(cachedNamespaces)
	case etc.InstallModeMultiNamespace:
		// Add support for MultiNamespace set in STARBOARD_NAMESPACE (e.g. marketplace) and STARBOARD_TARGET_NAMESPACES (e.g. foo,bar).
		// Note that we may face performance issues when using this with a high number of namespaces.
		// More: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/cache#MultiNamespacedCacheBuilder
		cachedNamespaces := append(targetNamespaces, operatorNamespace)
		setupLog.Info("Constructing multi-namespaced cache", "namespaces", cachedNamespaces)
		options.Namespace = ""
		options.NewCache = cache.MultiNamespacedCacheBuilder(cachedNamespaces)
	case etc.InstallModeAllNamespaces:
		// Add support for AllNamespaces set in STARBOARD_NAMESPACE (e.g. marketplace) and STARBOARD_TARGET_NAMESPACES left blank.
		setupLog.Info("Watching all namespaces")
		options.Namespace = ""
	default:
		return fmt.Errorf("unrecognized install mode: %v", installMode)
	}

	kubernetesConfig, err := ctrl.GetConfig()
	if err != nil {
		return fmt.Errorf("getting kube client config: %w", err)
	}

	// The only reason we're using kubernetes.Clientset is that we need it to read Pod logs,
	// which is not supported by the client returned by the ctrl.Manager.
	kubernetesClientset, err := kubernetes.NewForConfig(kubernetesConfig)
	if err != nil {
		return fmt.Errorf("constructing kube client: %w", err)
	}

	mgr, err := ctrl.NewManager(kubernetesConfig, options)
	if err != nil {
		return fmt.Errorf("constructing controllers manager: %w", err)
	}

	err = mgr.AddReadyzCheck("ping", healthz.Ping)
	if err != nil {
		return err
	}

	err = mgr.AddHealthzCheck("ping", healthz.Ping)
	if err != nil {
		return err
	}

	configManager := starboard.NewConfigManager(kubernetesClientset, operatorNamespace)
	err = configManager.EnsureDefault(context.Background())
	if err != nil {
		return err
	}

	starboardConfig, err := configManager.Read(context.Background())
	if err != nil {
		return err
	}

	store := vulnerabilityreport.NewStore(mgr.GetClient(), mgr.GetScheme())
	idGenerator := ext.NewGoogleUUIDGenerator()

	scanner, err := getEnabledScanner(buildInfo, idGenerator, operatorConfig, starboardConfig)
	if err != nil {
		return err
	}

	analyzer := controller.NewAnalyzer(operatorConfig.Operator,
		store,
		mgr.GetClient())

	reconciler := controller.NewReconciler(mgr.GetScheme(),
		operatorConfig.Operator,
		mgr.GetClient(),
		store,
		idGenerator,
		scanner,
		logs.NewReader(kubernetesClientset))

	if err = (&pod.PodController{
		Operator:   operatorConfig.Operator,
		Client:     mgr.GetClient(),
		Analyzer:   analyzer,
		Reconciler: reconciler,
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create pod controller: %w", err)
	}

	if err = (&job.JobController{
		Operator:   operatorConfig.Operator,
		Client:     mgr.GetClient(),
		Analyzer:   analyzer,
		Reconciler: reconciler,
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create job controller: %w", err)
	}

	setupLog.Info("Starting controllers manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		return fmt.Errorf("starting controllers manager: %w", err)
	}

	return nil
}

func getEnabledScanner(
	buildInfo starboard.BuildInfo,
	idGenerator ext.IDGenerator,
	config etc.Config,
	starboardConfig starboard.ConfigData,
) (vulnerabilityreport.Plugin, error) {
	if config.ScannerTrivy.Enabled && config.ScannerAquaCSP.Enabled {
		return nil, fmt.Errorf("invalid configuration: multiple vulnerability scanners enabled")
	}
	if !config.ScannerTrivy.Enabled && !config.ScannerAquaCSP.Enabled {
		return nil, fmt.Errorf("invalid configuration: none vulnerability scanner enabled")
	}
	if config.ScannerTrivy.Enabled {
		setupLog.Info("Using Trivy as vulnerability scanner",
			"trivyImageRef", starboardConfig.GetTrivyImageRef(),
			"trivyMode", starboardConfig.GetTrivyMode(),
			"trivyServerURL", starboardConfig.GetTrivyServerURL())
		return trivy.NewScannerPlugin(idGenerator, starboardConfig), nil
	}
	if config.ScannerAquaCSP.Enabled {
		setupLog.Info("Using Aqua CSP as vulnerability scanner", "image", config.ScannerAquaCSP.ImageRef)
		return aqua.NewScanner(idGenerator, buildInfo, config.ScannerAquaCSP), nil
	}
	return nil, errors.New("invalid configuration: unhandled vulnerability scanners config")
}
