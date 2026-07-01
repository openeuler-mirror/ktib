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

	"gitee.com/openeuler/ktib/pkg/options"
	ktype "gitee.com/openeuler/ktib/pkg/types"
	"github.com/containers/common/pkg/report"
	container "github.com/containers/storage"
	"github.com/docker/go-units"
	"github.com/sirupsen/logrus"
)

type containerReport struct {
	ID      string
	Names   string
	LayerID string
	ImageID string
	Created string
}

func sortContainers(containers []container.Container) ([]containerReport, error) {
	var containerReports []containerReport
	for _, c := range containers {
		var containerName string
		if len(c.Names) > 0 {
			containerName = c.Names[0]
		}
		containerReports = append(containerReports, containerReport{
			ID:      c.ID[:10],
			Names:   containerName,
			LayerID: c.LayerID,
			ImageID: c.ImageID,
			Created: units.HumanDuration(time.Since(c.Created)) + " ago",
		})
	}
	return containerReports, nil
}

func FormatBuilders(containers []container.Container, ops options.BuildersOption) error {
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
		jsonBuilders = append(jsonBuilders, ktype.JsonBuilder{
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
