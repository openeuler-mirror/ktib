/*
   Copyright (c) 2025 KylinSoft Co., Ltd.
   Kylin trusted image builder(ktib) is licensed under Mulan PSL v2.
   You can use this software according to the terms and conditions of the Mulan PSL v2.
   You may obtain a copy of Mulan PSL v2 at:
            http://license.coscl.org.cn/MulanPSL2
   THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR IMPLIED, INCLUDING
   BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
   See the Mulan PSL v2 for more details.
*/

package logging

import (
    "os"
    "strings"
    "github.com/sirupsen/logrus"
)

func Setup(level string, format string) {
    if level == "" {
        level = strings.ToLower(os.Getenv("KTIB_LOG_LEVEL"))
    }
    if format == "" {
        format = strings.ToLower(os.Getenv("KTIB_LOG_FORMAT"))
    }
    switch level {
    case "trace":
        logrus.SetLevel(logrus.TraceLevel)
    case "debug":
        logrus.SetLevel(logrus.DebugLevel)
    case "warn":
        logrus.SetLevel(logrus.WarnLevel)
    case "error":
        logrus.SetLevel(logrus.ErrorLevel)
    case "fatal":
        logrus.SetLevel(logrus.FatalLevel)
    case "panic":
        logrus.SetLevel(logrus.PanicLevel)
    default:
        logrus.SetLevel(logrus.InfoLevel)
    }
    switch format {
    case "json":
        logrus.SetFormatter(&logrus.JSONFormatter{})
    default:
        logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true, DisableColors: true})
    }
    logrus.SetOutput(os.Stdout)
}
