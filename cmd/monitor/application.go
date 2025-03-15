package main

import (
	"context"
	"fmt"

	"github.com/MaurUppi/disk-health-monitor/internal/collector"
	"github.com/MaurUppi/disk-health-monitor/internal/model"
	"github.com/MaurUppi/disk-health-monitor/internal/output"
	"github.com/MaurUppi/disk-health-monitor/internal/storage"
	"github.com/MaurUppi/disk-health-monitor/internal/system"
)

// Application represents the main application structure
type Application struct {
	Config         *model.Config
	Logger         system.Logger
	CommandRunner  system.CommandRunner
	DiskCollector  *collector.DiskCollector
	CtrlCollector  *collector.ControllerCollector
	HistoryStorage *storage.DiskHistoryStorage
	ExitOnWarning  bool
	OnlyWarnings   bool
	Quiet          bool
	CompactMode    bool
}

// NewApplication creates and initializes a new application instance
func NewApplication(config *model.Config, options map[string]interface{}) (*Application, error) {
	// Define log level
	var logLevel system.LogLevel
	if config.Debug {
		logLevel = system.LogLevelDebug
	} else if config.Verbose {
		logLevel = system.LogLevelInfo
	} else {
		logLevel = system.LogLevelError
	}

	// Initialize logger
	logger := system.NewLogger(config.LogFile, logLevel, config.Verbose)

	if config.Debug {
		logger.Debug("Debug mode enabled")
	}
	logger.Info("Initializing application")

	// Initialize command runner
	cmdRunner := &system.DefaultCommandRunner{}

	// Initialize history storage
	historyStorage := storage.NewDiskHistoryStorage(config.DataFile, logger)

	// Set storage path to ensure directory exists
	if err := historyStorage.SetStoragePath(config.DataFile); err != nil {
		return nil, fmt.Errorf("failed to set storage path: %w", err)
	}

	// Create application with options
	app := &Application{
		Config:         config,
		Logger:         logger,
		CommandRunner:  cmdRunner,
		HistoryStorage: historyStorage,
		// Set additional options from options map
		ExitOnWarning: getBoolOption(options, "exit_on_warning", false),
		OnlyWarnings:  getBoolOption(options, "only_warnings", false),
		Quiet:         getBoolOption(options, "quiet", false),
		CompactMode:   getBoolOption(options, "compact", false),
	}

	// Initialize collectors
	// Note: We initialize them here after creating the app because they depend on the config
	// which may have additional options
	app.DiskCollector = collector.NewDiskCollector(config, logger, cmdRunner)
	app.CtrlCollector = collector.NewControllerCollector(cmdRunner, logger)

	logger.Info("Application initialization complete")
	return app, nil
}

// getBoolOption safely extracts a boolean option from the options map
func getBoolOption(options map[string]interface{}, key string, defaultValue bool) bool {
	if options == nil {
		return defaultValue
	}

	if val, ok := options[key]; ok {
		if boolVal, ok := val.(bool); ok {
			return boolVal
		}
	}
	return defaultValue
}

// Run executes the main application workflow
func (app *Application) Run() int {
	app.Logger.Info("Starting disk health monitor")

	// Check required tools
	if err := checkRequiredTools(app.Logger, app.CommandRunner); err != nil {
		app.Logger.Error("Required tools check failed: %v", err)
		createDummyOutput(app.Config, fmt.Sprintf("Required tools not found: %v", err))
		return 2 // Initialization error
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), app.Config.CommandTimeout)
	defer cancel()

	// Collect controller data (if needed)
	var ctrlData *model.ControllerData
	var ctrlErr error

	if !app.Config.NoController {
		app.Logger.Info("Collecting controller information")
		ctrlData, ctrlErr = app.CtrlCollector.Collect(ctx)
		if ctrlErr != nil {
			app.Logger.Error("Failed to collect controller information: %v", ctrlErr)
			// Continue with other operations even if controller collection fails
		} else {
			app.Logger.Info("Found %d controllers (%d LSI, %d NVMe)",
				ctrlData.GetTotalControllerCount(),
				ctrlData.GetLSIControllerCount(),
				ctrlData.GetNVMeControllerCount())
		}
	}

	// Skip disk collection if only controller info is requested
	if app.Config.ControllerOnly {
		if ctrlErr != nil {
			app.Logger.Error("Controller data collection failed and --controller-only was specified")
			return 3 // Data collection error
		}

		// Generate output with only controller data
		if err := app.generateOutput(nil, ctrlData); err != nil {
			app.Logger.Error("Failed to generate output: %v", err)
			return 4 // Output generation error
		}

		return 0 // Success
	}

	// Collect disk data
	app.Logger.Info("Collecting disk information")
	diskData, diskErr := app.DiskCollector.Collect(ctx)
	if diskErr != nil {
		app.Logger.Error("Failed to collect disk information: %v", diskErr)

		// If both controller and disk collection failed, return error
		if ctrlErr != nil {
			app.Logger.Error("All data collection operations failed")
			createDummyOutput(app.Config, "Failed to collect any disk or controller data")
			return 3 // Data collection error
		}
	} else {
		// Log summary of disk data
		app.Logger.Info("Found %d disks (SSD: %d, HDD: %d, warnings: %d, errors: %d)",
			diskData.GetDiskCount(),
			diskData.GetSSDCount(),
			diskData.GetHDDCount(),
			diskData.GetWarningCount(),
			diskData.GetErrorCount())
	}

	// Filter only warning/error disks if requested
	if app.OnlyWarnings && diskData != nil {
		// Create a new filtered disk data object
		filteredData := model.NewDiskData()

		// Copy only disks with warnings or errors
		for _, disk := range diskData.Disks {
			status := disk.GetStatus()
			if status == model.DiskStatusWarning || status == model.DiskStatusError {
				filteredData.AddDisk(disk)
			}
		}

		// Use filtered data for output
		diskData = filteredData
		app.Logger.Info("Filtered to %d disks with warnings or errors", diskData.GetDiskCount())
	}

	// Generate output
	if err := app.generateOutput(diskData, ctrlData); err != nil {
		app.Logger.Error("Failed to generate output: %v", err)
		createDummyOutput(app.Config, fmt.Sprintf("Failed to generate output: %v", err))
		return 4 // Output generation error
	}

	// Check for warnings if --exit-on-warning is enabled
	if app.ExitOnWarning && diskData != nil {
		if diskData.GetWarningCount() > 0 || diskData.GetErrorCount() > 0 {
			app.Logger.Info("Exiting with status 5 due to warnings or errors detected")
			return 5 // Warning found
		}
	}

	app.Logger.Info("Disk health monitor completed successfully")
	return 0 // Success
}

// generateOutput creates formatted output based on collected data
func (app *Application) generateOutput(diskData *model.DiskData, ctrlData *model.ControllerData) error {
	// Determine output format
	format := string(app.Config.OutputFormat)

	// Get formatter options
	options := formatFormatterOptions(app)

	// Create formatter
	formatter, err := output.NewFormatter(format, options)
	if err != nil {
		return fmt.Errorf("failed to create output formatter: %w", err)
	}

	// Set data in formatter
	if app.Config.ControllerOnly {
		// Special handling for controller-only mode
		if err := formatter.FormatControllerInfo(ctrlData); err != nil {
			app.Logger.Error("Failed to format controller information: %v", err)
			return fmt.Errorf("controller formatting failed: %w", err)
		}
	} else {
		// Format controller data if available
		if ctrlData != nil && !app.Config.NoController {
			if err := formatter.FormatControllerInfo(ctrlData); err != nil {
				app.Logger.Error("Failed to format controller information: %v", err)
			}
		}

		// Format disk data if available
		if diskData != nil {
			if err := formatter.FormatDiskInfo(diskData); err != nil {
				app.Logger.Error("Failed to format disk information: %v", err)
			}
		}
	}

	// Verify buffer has content
	if textFormatter, ok := formatter.(fmt.Stringer); ok {
		if len(textFormatter.String()) == 0 {
			app.Logger.Error("Formatter produced empty output")
			return fmt.Errorf("empty output generated")
		}
	}

	if app.Config.Debug {
		if ctrlData != nil {
			app.Logger.Debug("Controller Data: LSI=%d, NVMe=%d",
				len(ctrlData.LSIControllers),
				len(ctrlData.NVMeControllers))
		}
		if textFormatter, ok := formatter.(fmt.Stringer); ok {
			output := textFormatter.String()
			app.Logger.Debug("Formatted Output Length: %d bytes", len(output))
			if len(output) < 100 {
				app.Logger.Debug("Formatted Output: %s", output)
			} else if len(output) == 0 {
				app.Logger.Debug("WARNING: Output is empty!")
			}
		} else {
			app.Logger.Debug("Formatter doesn't implement Stringer interface")
		}
	}

	// Save to file if output file is specified
	if app.Config.OutputFile != "" {
		if err := formatter.SaveToFile(app.Config.OutputFile); err != nil {
			return fmt.Errorf("failed to save output to file: %w", err)
		}

		if !app.Quiet {
			fmt.Printf("Output saved to %s\n", app.Config.OutputFile)
		}
		app.Logger.Info("Output saved to %s", app.Config.OutputFile)
	} else {
		// Print to console if no output file is specified
		if !app.Quiet {
			if textFormatter, ok := formatter.(fmt.Stringer); ok {
				fmt.Println(textFormatter.String())
			} else {
				// Fallback to simple output
				fmt.Println("Data collection complete.")
				// Print summary information
			}
		}
	}

	return nil
}
