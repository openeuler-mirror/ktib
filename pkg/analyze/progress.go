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

package analyze

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

// NewAnalysisProgressBar creates a new progress bar for the analysis process.
// It returns a callback function to update progress and a wait function to ensure completion.
func NewAnalysisProgressBar(totalSteps int) (func(string, bool, time.Duration), func()) {
	// Only show progress bar if we are not outputting JSON to stdout (to avoid mixing output)
	// We send to stderr so it's fine.
	p := mpb.New(mpb.WithOutput(os.Stderr), mpb.WithWidth(60))

	var currentStep string
	var mu sync.Mutex

	stepDecor := decor.Any(func(s decor.Statistics) string {
		mu.Lock()
		defer mu.Unlock()
		return currentStep
	}, decor.WC{W: 25})

	bar := p.New(int64(totalSteps),
		mpb.BarStyle().Lbound("").Rbound("").Filler("█").Tip("█").Padding("░"),
		mpb.PrependDecorators(
			decor.Spinner(nil, decor.WC{W: 2, C: decor.DSyncSpace}),
			stepDecor,
		),
		mpb.AppendDecorators(
			decor.CurrentNoUnit(""),
			decor.Name("/", decor.WC{W: 1}),
			decor.TotalNoUnit(""),
			decor.Percentage(decor.WCSyncSpace),
		),
	)

	progressFunc := func(step string, done bool, duration time.Duration) {
		if !done {
			mu.Lock()
			currentStep = step
			mu.Unlock()
		} else {
			// Step finished
			msg := fmt.Sprintf("\x1b[32m✔ %s\x1b[0m (%v)\n", step, duration.Round(time.Millisecond))
			fmt.Fprintf(os.Stderr, "\r%s", msg)
			bar.Increment()
		}
	}

	waitFunc := func() {
		if p != nil {
			p.Wait()
		}
	}

	return progressFunc, waitFunc
}
