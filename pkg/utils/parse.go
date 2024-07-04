/*
   Copyright (c) 2023 KylinSoft Co., Ltd.
   Kylin trusted image builder(ktib) is licensed under Mulan PSL v2.
   You can use this software according to the terms and conditions of the Mulan PSL v2.
   You may obtain a copy of Mulan PSL v2 at:
            http://license.coscl.org.cn/MulanPSL2
   THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR IMPLIED, INCLUDING
   BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
   See the Mulan PSL v2 for more details.
*/

package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"gitee.com/openeuler/ktib/pkg/imagemanager"

	"gitee.com/openeuler/ktib/pkg/options"

	"gitee.com/openeuler/ktib/pkg/builder"
	ktype "gitee.com/openeuler/ktib/pkg/types"
	"github.com/containers/common/pkg/report"
	"github.com/containers/image/v5/types"
	container "github.com/containers/storage"
	"github.com/docker/go-units"
	"github.com/opencontainers/go-digest"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const unknownState = "<none>"

type imageReport struct {
	Name     string
	ID       string
	Digest   digest.Digest
	Size     string
	Created  string
	TopLayer string
}

type containerReport struct {
	ID      string
	Names   string
	LayerID string
	ImageID string
	Created string
}

func humanSize(s int64) string {
	if s < 1024 {
		return fmt.Sprintf("%.2fB", float64(s)/float64(1))
	} else if s < (1024 * 1024) {
		return fmt.Sprintf("%.2fKB", float64(s)/float64(1024))
	} else if s < (1024 * 1024 * 1024) {
		return fmt.Sprintf("%.2fMB", float64(s)/float64(1024*1024))
	} else if s < (1024 * 1024 * 1024 * 1024) {
		return fmt.Sprintf("%.2fGB", float64(s)/float64(1024*1024*1024))
	} else {
		return fmt.Sprintf("%.2fTB", float64(s)/float64(1024*1024*1024*1024))
	}
}

func sortImages(imgs []imagemanager.Image) ([]imageReport, error) {
	var imgReport []imageReport
	for _, img := range imgs {
		size := img.Size
		createdAgo := units.HumanDuration(time.Since(img.OriImage.Created)) + " ago"
		topLayer := img.OriImage.TopLayer[0:10]
		imgID := img.OriImage.ID[:10]

		if len(img.OriImage.Names) > 0 {
			for _, name := range append(img.OriImage.Names, unknownState)[:len(img.OriImage.Names)] {
				imgReport = append(imgReport, imageReport{
					Name:     name,
					ID:       imgID,
					Digest:   img.OriImage.Digest,
					TopLayer: topLayer,
					Created:  createdAgo,
					Size:     humanSize(size),
				})
			}
		} else {
			imgReport = append(imgReport, imageReport{
				Name:     unknownState,
				ID:       imgID,
				Digest:   img.OriImage.Digest,
				TopLayer: topLayer,
				Created:  createdAgo,
				Size:     humanSize(size),
			})
		}
	}
	return imgReport, nil
}

func sortContainers(containers []container.Container) ([]containerReport, error) {
	var containerReports []containerReport
	for _, c := range containers {
		var containerName string
		if len(c.Names) > 0 {
			containerName = c.Names[0]
		} else {
			containerName = ""
		}
		containerReports = append(containerReports, containerReport{
			ID:      c.ID[:10],
			Names:   containerName,
			LayerID: c.LayerID,
			ImageID: c.ImageID[:10],
			Created: units.HumanDuration(time.Since(c.Created)) + " ago",
		})
	}
	return containerReports, nil
}

func FormatImages(images []imagemanager.Image, ops options.ImagesOption) error {
	//TODO 参考docker以image table format 输出
	defaultImageTableFormat := "table {{.Name}} {{.ID}}  {{.Size}} {{.TopLayer}}   {{.Created}}"
	defaultImageTableFormatWithDigest := "table {{.Name}} {{.ID}} {{.Digest}} {{.Size}} {{.TopLayer}} {{.Created}}"
	defaultQuietFormat := "table {{.ID}}"
	// defaultImageTableFormatWithDigest = "table {{.Repository}}\t{{.Tag}}\t{{.Digest}}\t{{.ID}}\t{{.CreatedSince}}\t{{.Size}}"
	// 构造所需的image结构=>sortImage
	imagesReport, err := sortImages(images)
	if err != nil {
		return err
	}
	headers := report.Headers(imageReport{}, map[string]string{
		"Name": "Name",
	})
	if ops.Quiet {
		defaultImageTableFormat = defaultQuietFormat
	} else if ops.Digests {
		defaultImageTableFormat = defaultImageTableFormatWithDigest
	} else if ops.Format != "" {
		defaultImageTableFormat = "table " + ops.Format
	}
	formater, err := report.New(os.Stdout, "format").Parse(report.OriginPodman, defaultImageTableFormat)
	if err != nil {
		return err
	}
	defer func() {
		err = formater.Flush()
		if err != nil {
			logrus.Error(err)
		}
	}()
	if !ops.Quiet {
		err = formater.Execute(headers)
		if err != nil {
			return err
		}
	}
	err = formater.Execute(imagesReport)
	if err != nil {
		return err
	}
	return nil
}

func SystemContextFromFlagSet(c *cobra.Command) (types.SystemContext, error) {

	return types.SystemContext{}, nil
}

func JsonFormatImages(images []imagemanager.Image, ops options.ImagesOption) error {
	var jsonImages []ktype.JsonImage

	for _, image := range images {
		jsonImages = append(jsonImages,
			ktype.JsonImage{
				Name:    image.OriImage.Names,
				Digest:  image.OriImage.Digest,
				ImageID: image.OriImage.ID,
				Created: image.OriImage.Created,
				Size:    image.Size,
			})
	}
	data, err := json.MarshalIndent(jsonImages, "", "    ")
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", data)
	return nil
}

func FormatBuilders(containers []container.Container, ops options.BuildersOption) error {
	// TODO 参考docker输出
	defaultBuilderTableFormat := "table {{.ID}}  {{.Names}} {{.LayerID}} {{.ImageID}}   {{.Created}}"
	containerReports, err := sortContainers(containers)
	if err != nil {
		return err
	}
	headers := report.Headers(containerReport{}, map[string]string{
		"Name": "Name",
	})
	formater, err := report.New(os.Stdout, "format").Parse(report.OriginPodman, defaultBuilderTableFormat)
	if err != nil {
		return err
	}
	defer func() {
		err = formater.Flush()
		if err != nil {
			logrus.Error(err)
		}
	}()
	err = formater.Execute(headers)
	if err != nil {
		return err
	}
	err = formater.Execute(containerReports)
	if err != nil {
		return err
	}
	return nil
}

func JsonFormatBuilders(containers []container.Container, ops options.BuildersOption) error {
	var jsonBuilders []ktype.JsonBuilder
	for _, b := range containers {
		jsonBuilders = append(jsonBuilders,
			ktype.JsonBuilder{
				ID:      b.ID,
				Names:   b.Names,
				ImageID: b.ImageID,
				Created: b.Created,
			})
	}
	data, err := json.MarshalIndent(jsonBuilders, "", "    ")
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", data)
	return nil
}

func JsonFormatMountInfo(builders []*builder.Builder) error {
	var jsonBuilders []ktype.JsonBuilder
	for _, b := range builders {
		if b.MountPoint != "" {
			jsonBuilders = append(jsonBuilders,
				ktype.JsonBuilder{
					ID:      b.ID,
					Mount:   b.MountPoint,
					ImageID: b.FromImageID,
				})
		}
	}
	data, err := json.MarshalIndent(jsonBuilders, "", "    ")
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", data)
	return nil
}

func FormatMountInfo(builders []*builder.Builder) error {
	return nil
}
