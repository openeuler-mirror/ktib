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
	"sort"
	"strings"
	"time"

	"gitee.com/openeuler/ktib/pkg/imagemanager"
	"gitee.com/openeuler/ktib/pkg/options"
	ktype "gitee.com/openeuler/ktib/pkg/types"
	"github.com/containers/common/pkg/report"
	"github.com/containers/image/v5/docker/reference"
	"github.com/docker/go-units"
	"github.com/opencontainers/go-digest"
	"github.com/sirupsen/logrus"
)

const unknownState = "<none>"

type imageReport struct {
	Repository string
	Tag        string
	ID         string
	Digest     digest.Digest
	Size       string
	Created    string
	TopLayer   string
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
	}
	return fmt.Sprintf("%.2fTB", float64(s)/float64(1024*1024*1024*1024))
}

func parseImageName(fullName string) (repository, tag string) {
	if fullName == "" {
		return unknownState, unknownState
	}

	parsed, err := reference.ParseNormalizedNamed(fullName)
	if err != nil {
		return manualParseImageName(fullName)
	}

	repository = reference.FamiliarName(parsed)
	if tagged, ok := parsed.(reference.Tagged); ok {
		tag = tagged.Tag()
	} else if digested, ok := parsed.(reference.Digested); ok {
		digestStr := digested.Digest().String()
		if len(digestStr) > 12 {
			tag = digestStr[:12] + "..."
		} else {
			tag = digestStr
		}
	} else {
		tag = unknownState
	}

	return repository, tag
}

func manualParseImageName(fullName string) (repository, tag string) {
	fullName = strings.TrimSpace(fullName)
	if fullName == "" {
		return unknownState, unknownState
	}

	lastColon := strings.LastIndex(fullName, ":")
	if lastColon <= 0 {
		return fullName, unknownState
	}

	if lastColon > 0 {
		charBeforeColon := fullName[lastColon-1]
		if charBeforeColon >= '0' && charBeforeColon <= '9' {
			prevColon := strings.LastIndex(fullName[:lastColon], ":")
			if prevColon > 0 {
				repository = fullName[:prevColon]
				tag = fullName[prevColon+1:]
				return repository, tag
			}
		}
	}

	repository = fullName[:lastColon]
	tag = fullName[lastColon+1:]
	if tag == "" {
		tag = unknownState
	}

	return repository, tag
}

func sortImages(imgs []*imagemanager.Image, ops options.ImagesOption) ([]imageReport, error) {
	var imgReport []imageReport

	for _, img := range imgs {
		size := img.Size
		createdAgo := units.HumanDuration(time.Since(img.OriImage.Created)) + " ago"

		topLayer := img.OriImage.TopLayer
		if len(topLayer) > 10 {
			topLayer = topLayer[:10]
		}

		imgID := img.OriImage.ID
		if !ops.NoTrunc {
			if len(imgID) > 10 {
				imgID = imgID[:10]
			}
		} else if len(imgID) > 12 {
			imgID = imgID[:12]
		}

		if len(img.OriImage.Names) > 0 {
			for _, name := range img.OriImage.Names {
				repository, tag := parseImageName(name)
				imgReport = append(imgReport, imageReport{
					Repository: repository,
					Tag:        tag,
					ID:         imgID,
					Digest:     img.OriImage.Digest,
					TopLayer:   topLayer,
					Created:    createdAgo,
					Size:       humanSize(size),
				})
			}
		} else {
			imgReport = append(imgReport, imageReport{
				Repository: unknownState,
				Tag:        unknownState,
				ID:         imgID,
				Digest:     img.OriImage.Digest,
				TopLayer:   topLayer,
				Created:    createdAgo,
				Size:       humanSize(size),
			})
		}
	}

	sort.Slice(imgReport, func(i, j int) bool {
		return imgReport[i].Repository < imgReport[j].Repository
	})
	return imgReport, nil
}

func FormatImages(images []*imagemanager.Image, ops options.ImagesOption) error {
	defaultImageTableFormat := "table {{.Repository}} {{.Tag}} {{.ID}} {{.Size}} {{.Created}}"
	defaultImageTableFormatWithDigest := "table {{.Repository}} {{.Tag}} {{.ID}} {{.Digest}} {{.Size}} {{.Created}}"
	defaultQuietFormat := "table {{.ID}}"

	imagesReport, err := sortImages(images, ops)
	if err != nil {
		return err
	}

	headers := report.Headers(imageReport{}, map[string]string{
		"Repository": "REPOSITORY",
		"Tag":        "TAG",
		"ID":         "IMAGE ID",
		"Size":       "SIZE",
		"Created":    "CREATED",
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

func JsonFormatImages(images []*imagemanager.Image, ops options.ImagesOption) error {
	var jsonImages []ktype.JsonImage

	for _, image := range images {
		jsonImages = append(jsonImages, ktype.JsonImage{
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
