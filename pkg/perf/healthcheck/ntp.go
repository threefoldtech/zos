package healthcheck

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/zinit"
)

const acceptableSkew = 10 * time.Minute

func ntpCheck(ctx context.Context) []error {
	var errs []error
	z := zinit.Default()

	localTime, err := getCurrentLocalTime()
	if err != nil {
		errs = append(errs, err)
	}

	if math.Abs(float64(time.Since(localTime))) > float64(acceptableSkew) {
		if err := z.Kill("ntp", zinit.SIGTERM); err != nil {
			errs = append(errs, errors.Wrapf(err, "failed to restart ntpd"))
		}
	}

	return errs
}

func getCurrentLocalTime() (time.Time, error) {
	timeRes, err := http.Get("https://worldtimeapi.org/api/timezone/UTC")
	if err != nil {
		return time.Time{}, errors.Wrapf(err, "failed to get date")
	}

	var result struct {
		DateTime string `json:"datetime"`
	}
	err = json.NewDecoder(timeRes.Body).Decode(&result)
	timeRes.Body.Close()
	if err != nil {
		return time.Time{}, errors.Wrapf(err, "failed to decode date response")
	}

	timeLayout := "2006-01-02T15:04:05+00:00"
	localTime, err := time.Parse(timeLayout, result.DateTime)
	if err != nil {
		return time.Time{}, errors.Wrapf(err, "failed to parse date")
	}

	return localTime, nil
}
