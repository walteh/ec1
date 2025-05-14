// This file is safe to edit. Once it exists it will not be overwritten

package restapi

import (
	"crypto/tls"
	"net/http"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"

	"github.com/walteh/ec1/gen/firecracker-swagger-go/restapi/operations"
)

//go:generate swagger generate server --target ../../firecracker-swagger-go --name FirecrackerAPI --spec ../../../docs/firecracker/firecracker.swagger.yaml --template-dir ./docs/firecracker/templates --principal interface{} --strict-responders

func configureFlags(api *operations.FirecrackerAPIAPI) {
	// api.CommandLineOptionsGroups = []swag.CommandLineOptionsGroup{ ... }
}

func configureAPI(api *operations.FirecrackerAPIAPI) http.Handler {
	// configure the api here
	api.ServeError = errors.ServeError

	// Set your custom logger if needed. Default one is log.Printf
	// Expected interface func(string, ...interface{})
	//
	// Example:
	// api.Logger = log.Printf

	api.UseSwaggerUI()
	// To continue using redoc as your UI, uncomment the following line
	// api.UseRedoc()

	api.JSONConsumer = runtime.JSONConsumer()

	api.JSONProducer = runtime.JSONProducer()

	if api.CreateSnapshotHandler == nil {
		api.CreateSnapshotHandler = operations.CreateSnapshotHandlerFunc(func(params operations.CreateSnapshotParams) operations.CreateSnapshotResponder {
			return operations.CreateSnapshotNotImplemented()
		})
	}
	if api.CreateSyncActionHandler == nil {
		api.CreateSyncActionHandler = operations.CreateSyncActionHandlerFunc(func(params operations.CreateSyncActionParams) operations.CreateSyncActionResponder {
			return operations.CreateSyncActionNotImplemented()
		})
	}
	if api.DescribeBalloonConfigHandler == nil {
		api.DescribeBalloonConfigHandler = operations.DescribeBalloonConfigHandlerFunc(func(params operations.DescribeBalloonConfigParams) operations.DescribeBalloonConfigResponder {
			return operations.DescribeBalloonConfigNotImplemented()
		})
	}
	if api.DescribeBalloonStatsHandler == nil {
		api.DescribeBalloonStatsHandler = operations.DescribeBalloonStatsHandlerFunc(func(params operations.DescribeBalloonStatsParams) operations.DescribeBalloonStatsResponder {
			return operations.DescribeBalloonStatsNotImplemented()
		})
	}
	if api.DescribeInstanceHandler == nil {
		api.DescribeInstanceHandler = operations.DescribeInstanceHandlerFunc(func(params operations.DescribeInstanceParams) operations.DescribeInstanceResponder {
			return operations.DescribeInstanceNotImplemented()
		})
	}
	if api.GetExportVMConfigHandler == nil {
		api.GetExportVMConfigHandler = operations.GetExportVMConfigHandlerFunc(func(params operations.GetExportVMConfigParams) operations.GetExportVMConfigResponder {
			return operations.GetExportVMConfigNotImplemented()
		})
	}
	if api.GetFirecrackerVersionHandler == nil {
		api.GetFirecrackerVersionHandler = operations.GetFirecrackerVersionHandlerFunc(func(params operations.GetFirecrackerVersionParams) operations.GetFirecrackerVersionResponder {
			return operations.GetFirecrackerVersionNotImplemented()
		})
	}
	if api.GetMachineConfigurationHandler == nil {
		api.GetMachineConfigurationHandler = operations.GetMachineConfigurationHandlerFunc(func(params operations.GetMachineConfigurationParams) operations.GetMachineConfigurationResponder {
			return operations.GetMachineConfigurationNotImplemented()
		})
	}
	if api.GetMmdsHandler == nil {
		api.GetMmdsHandler = operations.GetMmdsHandlerFunc(func(params operations.GetMmdsParams) operations.GetMmdsResponder {
			return operations.GetMmdsNotImplemented()
		})
	}
	if api.LoadSnapshotHandler == nil {
		api.LoadSnapshotHandler = operations.LoadSnapshotHandlerFunc(func(params operations.LoadSnapshotParams) operations.LoadSnapshotResponder {
			return operations.LoadSnapshotNotImplemented()
		})
	}
	if api.PatchBalloonHandler == nil {
		api.PatchBalloonHandler = operations.PatchBalloonHandlerFunc(func(params operations.PatchBalloonParams) operations.PatchBalloonResponder {
			return operations.PatchBalloonNotImplemented()
		})
	}
	if api.PatchBalloonStatsIntervalHandler == nil {
		api.PatchBalloonStatsIntervalHandler = operations.PatchBalloonStatsIntervalHandlerFunc(func(params operations.PatchBalloonStatsIntervalParams) operations.PatchBalloonStatsIntervalResponder {
			return operations.PatchBalloonStatsIntervalNotImplemented()
		})
	}
	if api.PatchGuestDriveByIDHandler == nil {
		api.PatchGuestDriveByIDHandler = operations.PatchGuestDriveByIDHandlerFunc(func(params operations.PatchGuestDriveByIDParams) operations.PatchGuestDriveByIDResponder {
			return operations.PatchGuestDriveByIDNotImplemented()
		})
	}
	if api.PatchGuestNetworkInterfaceByIDHandler == nil {
		api.PatchGuestNetworkInterfaceByIDHandler = operations.PatchGuestNetworkInterfaceByIDHandlerFunc(func(params operations.PatchGuestNetworkInterfaceByIDParams) operations.PatchGuestNetworkInterfaceByIDResponder {
			return operations.PatchGuestNetworkInterfaceByIDNotImplemented()
		})
	}
	if api.PatchMachineConfigurationHandler == nil {
		api.PatchMachineConfigurationHandler = operations.PatchMachineConfigurationHandlerFunc(func(params operations.PatchMachineConfigurationParams) operations.PatchMachineConfigurationResponder {
			return operations.PatchMachineConfigurationNotImplemented()
		})
	}
	if api.PatchMmdsHandler == nil {
		api.PatchMmdsHandler = operations.PatchMmdsHandlerFunc(func(params operations.PatchMmdsParams) operations.PatchMmdsResponder {
			return operations.PatchMmdsNotImplemented()
		})
	}
	if api.PatchVMHandler == nil {
		api.PatchVMHandler = operations.PatchVMHandlerFunc(func(params operations.PatchVMParams) operations.PatchVMResponder {
			return operations.PatchVMNotImplemented()
		})
	}
	if api.PutBalloonHandler == nil {
		api.PutBalloonHandler = operations.PutBalloonHandlerFunc(func(params operations.PutBalloonParams) operations.PutBalloonResponder {
			return operations.PutBalloonNotImplemented()
		})
	}
	if api.PutCPUConfigurationHandler == nil {
		api.PutCPUConfigurationHandler = operations.PutCPUConfigurationHandlerFunc(func(params operations.PutCPUConfigurationParams) operations.PutCPUConfigurationResponder {
			return operations.PutCPUConfigurationNotImplemented()
		})
	}
	if api.PutEntropyDeviceHandler == nil {
		api.PutEntropyDeviceHandler = operations.PutEntropyDeviceHandlerFunc(func(params operations.PutEntropyDeviceParams) operations.PutEntropyDeviceResponder {
			return operations.PutEntropyDeviceNotImplemented()
		})
	}
	if api.PutGuestBootSourceHandler == nil {
		api.PutGuestBootSourceHandler = operations.PutGuestBootSourceHandlerFunc(func(params operations.PutGuestBootSourceParams) operations.PutGuestBootSourceResponder {
			return operations.PutGuestBootSourceNotImplemented()
		})
	}
	if api.PutGuestDriveByIDHandler == nil {
		api.PutGuestDriveByIDHandler = operations.PutGuestDriveByIDHandlerFunc(func(params operations.PutGuestDriveByIDParams) operations.PutGuestDriveByIDResponder {
			return operations.PutGuestDriveByIDNotImplemented()
		})
	}
	if api.PutGuestNetworkInterfaceByIDHandler == nil {
		api.PutGuestNetworkInterfaceByIDHandler = operations.PutGuestNetworkInterfaceByIDHandlerFunc(func(params operations.PutGuestNetworkInterfaceByIDParams) operations.PutGuestNetworkInterfaceByIDResponder {
			return operations.PutGuestNetworkInterfaceByIDNotImplemented()
		})
	}
	if api.PutGuestVsockHandler == nil {
		api.PutGuestVsockHandler = operations.PutGuestVsockHandlerFunc(func(params operations.PutGuestVsockParams) operations.PutGuestVsockResponder {
			return operations.PutGuestVsockNotImplemented()
		})
	}
	if api.PutLoggerHandler == nil {
		api.PutLoggerHandler = operations.PutLoggerHandlerFunc(func(params operations.PutLoggerParams) operations.PutLoggerResponder {
			return operations.PutLoggerNotImplemented()
		})
	}
	if api.PutMachineConfigurationHandler == nil {
		api.PutMachineConfigurationHandler = operations.PutMachineConfigurationHandlerFunc(func(params operations.PutMachineConfigurationParams) operations.PutMachineConfigurationResponder {
			return operations.PutMachineConfigurationNotImplemented()
		})
	}
	if api.PutMetricsHandler == nil {
		api.PutMetricsHandler = operations.PutMetricsHandlerFunc(func(params operations.PutMetricsParams) operations.PutMetricsResponder {
			return operations.PutMetricsNotImplemented()
		})
	}
	if api.PutMmdsHandler == nil {
		api.PutMmdsHandler = operations.PutMmdsHandlerFunc(func(params operations.PutMmdsParams) operations.PutMmdsResponder {
			return operations.PutMmdsNotImplemented()
		})
	}
	if api.PutMmdsConfigHandler == nil {
		api.PutMmdsConfigHandler = operations.PutMmdsConfigHandlerFunc(func(params operations.PutMmdsConfigParams) operations.PutMmdsConfigResponder {
			return operations.PutMmdsConfigNotImplemented()
		})
	}

	api.PreServerShutdown = func() {}

	api.ServerShutdown = func() {}

	return setupGlobalMiddleware(api.Serve(setupMiddlewares))
}

// The TLS configuration before HTTPS server starts.
func configureTLS(tlsConfig *tls.Config) {
	// Make all necessary changes to the TLS configuration here.
}

// As soon as server is initialized but not run yet, this function will be called.
// If you need to modify a config, store server instance to stop it individually later, this is the place.
// This function can be called multiple times, depending on the number of serving schemes.
// scheme value will be set accordingly: "http", "https" or "unix".
func configureServer(s *http.Server, scheme, addr string) {
}

// The middleware configuration is for the handler executors. These do not apply to the swagger.json document.
// The middleware executes after routing but before authentication, binding and validation.
func setupMiddlewares(handler http.Handler) http.Handler {
	return handler
}

// The middleware configuration happens before anything, this middleware also applies to serving the swagger.json document.
// So this is a good place to plug in a panic handling middleware, logging and metrics.
func setupGlobalMiddleware(handler http.Handler) http.Handler {
	return handler
}
