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
	"errors"
	"fmt"
	"os"
	"strings"
)

const (
	// DefaultErrorExitCode defines exit the code for failed action generally
	DefaultErrorExitCode = 1
	// PreFlightExitCode defines exit the code for preflight checks
	PreFlightExitCode = 2
	// ValidationExitCode defines the exit code validation checks
	ValidationExitCode = 3
)

var (
	ErrInvalidSubCommandMsg = "invalid subcommand"
	ErrExit                 = errors.New("exit")
)

func fatal(msg string, code int) {
	if len(msg) > 0 {
		// add newline if needed
		if !strings.HasSuffix(msg, "\n") {
			msg += "\n"
		}

		fmt.Fprint(os.Stderr, msg)
	}
	os.Exit(code)
}

func CheckErr(err error) {
	checkErr(err, fatal)
}

func checkErr(err error, handleErr func(string, int)) {
	if err == nil {
		return
	}
	switch {
	case err == ErrExit:
		handleErr("", DefaultErrorExitCode)
	case strings.Contains(err.Error(), ErrInvalidSubCommandMsg):
		handleErr(err.Error(), DefaultErrorExitCode)
	default:
		switch err.(type) {
		//case preflightError:
		//	handleErr(msg, PreFlightExitCode)
		//case errorsutil.Aggregate:
		//	handleErr(msg, ValidationExitCode)

		default:
			handleErr(err.Error(), DefaultErrorExitCode)
		}
	}
}
