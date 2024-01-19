package options

import (
	"context"
	"github.com/aquasecurity/trivy-db/pkg/types"
	"github.com/aquasecurity/trivy/pkg/commands/artifact"
	"github.com/aquasecurity/trivy/pkg/commands/option"
	"github.com/aquasecurity/trivy/pkg/log"

	"go.uber.org/zap"
	"os"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
)

var (
	Severity = strings.Join(types.SeverityNames, ",")
)

func InitScanOption(args []string, opt Option) (artifact.Option, error) {
	var ctx cli.Context
	ctx.Context = context.Background()
	ctx.App = cli.NewApp()
	scanOption, err := initScanOptions(opt, ctx)
	if err != nil {
		return artifact.Option{}, err
	}
	if scanOption.Input == "" {
		scanOption.Target = args[0]
	}
	scanOption.Severities = getSeverity(scanOption.Logger, Severity)
	return scanOption, nil
}

func initScanOptions(opt Option, ctx cli.Context) (artifact.Option, error) {
	globalOption, err := newGlobalOption(opt, ctx)
	artifactOption := option.NewArtifactOption(&ctx)
	artifactOption.Timeout = time.Second * 300
	if err != nil {
		return artifact.Option{}, err
	}
	return artifact.Option{
		GlobalOption:     globalOption,
		ArtifactOption:   artifactOption,
		DBOption:         option.NewDBOption(&ctx),
		ImageOption:      option.NewImageOption(&ctx),
		ReportOption:     newReportOption(opt),
		CacheOption:      option.NewCacheOption(&ctx),
		ConfigOption:     newConfigOption(opt),
		RemoteOption:     option.NewRemoteOption(&ctx),
		SbomOption:       option.NewSbomOption(&ctx),
		SecretOption:     option.NewSecretOption(&ctx),
		KubernetesOption: option.NewKubernetesOption(&ctx),
		OtherOption:      option.NewOtherOption(&ctx),
	}, nil
}

func getSeverity(logger *zap.SugaredLogger, severity string) []types.Severity {
	logger.Debugf("Severities: %s", severity)
	var severities []types.Severity
	for _, s := range strings.Split(severity, ",") {
		severity, err := types.NewSeverity(s)
		if err != nil {
			logger.Warnf("unknown severity option: %s", err)
		}
		severities = append(severities, severity)
	}
	return severities
}

func newGlobalOption(opt Option, ctx cli.Context) (option.GlobalOption, error) {
	logger, err := log.NewLogger(false, false)
	if err != nil {
		return option.GlobalOption{}, err
	}
	return option.GlobalOption{
		Context:    &ctx,
		Logger:     logger,
		AppVersion: ctx.App.Version,
		CacheDir:   opt.CacheDir,
	}, nil
}

func newConfigOption(opt Option) option.ConfigOption {
	return option.ConfigOption{
		PolicyNamespaces: opt.PolicyNamespaces,
	}
}

func newReportOption(opt Option) option.ReportOption {
	return option.ReportOption{
		Format:      opt.Format,
		Output:      os.Stdout,
		ListAllPkgs: false,
	}
}
