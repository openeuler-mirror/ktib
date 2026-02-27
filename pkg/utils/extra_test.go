/*
   Copyright (c) 2026 KylinSoft Co., Ltd.
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
	"testing"
)

func TestFormatBytes(t *testing.T) {
	testCases := []struct {
		name string
		args int64
		want string
	}{
		{
			name: "Bytes",
			args: 500,
			want: "500 B",
		},
		{
			name: "Kilobytes",
			args: 1024,
			want: "1.0 KiB",
		},
		{
			name: "Megabytes",
			args: 2621440, // 2.5 MiB
			want: "2.5 MiB",
		},
		{
			name: "Gigabytes",
			args: 1610612736, // 1.5 GiB
			want: "1.5 GiB",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatBytes(tt.args); got != tt.want {
				t.Errorf("FormatBytes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidImageType(t *testing.T) {
	testCases := []struct {
		name string
		args string
		want bool
	}{
		{
			name: "Micro",
			args: "micro",
			want: true,
		},
		{
			name: "Minimal",
			args: "minimal",
			want: true,
		},
		{
			name: "Platform",
			args: "platform",
			want: true,
		},
		{
			name: "Init",
			args: "init",
			want: true,
		},
		{
			name: "Invalid",
			args: "invalid",
			want: false,
		},
		{
			name: "Empty",
			args: "",
			want: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidImageType(tt.args); got != tt.want {
				t.Errorf("IsValidImageType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckErrInternal(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		wantCode int
		wantMsg  string
	}{
		{
			name:     "Nil error",
			err:      nil,
			wantCode: 0, // Should not call handler
			wantMsg:  "",
		},
		{
			name:     "ErrExit",
			err:      ErrExit,
			wantCode: DefaultErrorExitCode,
			wantMsg:  "",
		},
		{
			name:     "Invalid Subcommand",
			err:      errors.New("invalid subcommand"),
			wantCode: DefaultErrorExitCode,
			wantMsg:  "invalid subcommand",
		},
		{
			name:     "Generic Error",
			err:      errors.New("generic error"),
			wantCode: DefaultErrorExitCode,
			wantMsg:  "generic error",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			mockHandler := func(msg string, code int) {
				called = true
				if code != tt.wantCode {
					t.Errorf("checkErr() code = %v, want %v", code, tt.wantCode)
				}
				if msg != tt.wantMsg {
					t.Errorf("checkErr() msg = %q, want %q", msg, tt.wantMsg)
				}
			}

			checkErr(tt.err, mockHandler)

			if tt.err != nil && !called {
				t.Error("checkErr() expected handler to be called")
			}
			if tt.err == nil && called {
				t.Error("checkErr() expected handler NOT to be called")
			}
		})
	}
}
