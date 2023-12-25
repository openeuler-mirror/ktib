package options

import (
	"github.com/aquasecurity/trivy/pkg/commands/artifact"
	"github.com/aquasecurity/trivy/pkg/commands/option"
	"github.com/aquasecurity/trivy/pkg/log"
	"os"

	"github.com/urfave/cli/v2"
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
		ListAllPkgs:  true,
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
