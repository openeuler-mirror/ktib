package options

import (
	"github.com/aquasecurity/trivy-db/pkg/types"
	"github.com/aquasecurity/trivy/pkg/commands/artifact"
	"github.com/aquasecurity/trivy/pkg/commands/option"
	"github.com/aquasecurity/trivy/pkg/log"
	"go.uber.org/zap"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
)

var (
	Severity = strings.Join(types.SeverityNames, ",")
)

func NewGlobalOption(opt Option, ctx cli.Context) (option.GlobalOption, error) {
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

func NewConfigOption(opt Option) option.ConfigOption {
	return option.ConfigOption{
		PolicyNamespaces: opt.PolicyNamespaces,
	}
}

func NewReportOption(opt Option) option.ReportOption {
	return option.ReportOption{
		Format:       opt.Format,
		IgnorePolicy: ".ktibignore",
		Output:       os.Stdout,
		ListAllPkgs:  false,
	}
}

func InitScanOptions(opt Option, ctx cli.Context) (artifact.Option, error) {
	globalOption, err := NewGlobalOption(opt, ctx)
	if err != nil {
		return artifact.Option{}, err
	}
	return artifact.Option{
		GlobalOption:     globalOption,
		ArtifactOption:   option.NewArtifactOption(&ctx),
		DBOption:         option.NewDBOption(&ctx),
		ImageOption:      option.NewImageOption(&ctx),
		ReportOption:     NewReportOption(opt),
		CacheOption:      option.NewCacheOption(&ctx),
		ConfigOption:     NewConfigOption(opt),
		RemoteOption:     option.NewRemoteOption(&ctx),
		SbomOption:       option.NewSbomOption(&ctx),
		SecretOption:     option.NewSecretOption(&ctx),
		KubernetesOption: option.NewKubernetesOption(&ctx),
		OtherOption:      option.NewOtherOption(&ctx),
	}, nil
}

func GetSeverity(logger *zap.SugaredLogger, severity string) []types.Severity {
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
