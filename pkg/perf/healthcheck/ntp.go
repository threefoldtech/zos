package healthcheck

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos4/pkg/zinit"
)

const acceptableSkew = 10 * time.Minute

// TimeServer represents a time server with its name and fetching function
type TimeServer struct {
	Name string
	Func func() (time.Time, error)
}

// List of time servers
var timeServers = []TimeServer{
	{
		Name: "worldtimeapi",
		Func: getWorldTimeAPI,
	},
	{
		Name: "worldclockapi",
		Func: getWorldClockAPI,
	},
	{
		Name: "timeapi.io",
		Func: getTimeAPI,
	},
}

func RunNTPCheck(ctx context.Context) {
	go func() {
		for {
			if err := ntpCheck(); err != nil {
				log.Error().Err(err).Msg("failed to run ntp check")
				continue
			}

			select {
			case <-ctx.Done():
				if ctx.Err() != nil {
					log.Error().Err(ctx.Err()).Msg("cli context done with error")
				}
				return
			case <-time.After(time.Minute):
			}
		}
	}()
}

func ntpCheck() error {
	z := zinit.Default()

	utcTime, err := getCurrentUTCTime()
	if err != nil {
		return err
	}

	if math.Abs(float64(time.Since(utcTime))) > float64(acceptableSkew) {
		if err := z.Kill("ntp", zinit.SIGTERM); err != nil {
			return errors.Wrapf(err, "failed to restart ntpd")
		}
	}

	return nil
}

func getCurrentUTCTime() (time.Time, error) {
	for _, server := range timeServers {
		utcTime, err := server.Func()
		if err == nil {
			return utcTime, nil
		}
		log.Error().Err(err).Str("server", server.Name).Msg("failed to get time from server")
	}
	return time.Time{}, errors.New("failed to get time from all servers")
}

func getWorldTimeAPI() (time.Time, error) {
	timeRes, err := http.Get("https://worldtimeapi.org/api/timezone/UTC")
	if err != nil {
		return time.Time{}, errors.Wrapf(err, "failed to get date from worldtimeapi")
	}
	defer timeRes.Body.Close()

	var utcTime struct {
		DateTime time.Time `json:"datetime"`
	}
	if err := json.NewDecoder(timeRes.Body).Decode(&utcTime); err != nil {
		return time.Time{}, errors.Wrapf(err, "failed to decode date response from worldtimeapi")
	}

	return utcTime.DateTime, nil
}

func getWorldClockAPI() (time.Time, error) {
	timeRes, err := http.Get("http://worldclockapi.com/api/json/utc/now")
	if err != nil {
		return time.Time{}, errors.Wrapf(err, "failed to get date from worldclockapi")
	}
	defer timeRes.Body.Close()

	var utcTime struct {
		CurrentDateTime time.Time `json:"currentDateTime"`
	}
	if err := json.NewDecoder(timeRes.Body).Decode(&utcTime); err != nil {
		return time.Time{}, errors.Wrapf(err, "failed to decode date response from worldclockapi")
	}

	return utcTime.CurrentDateTime, nil
}

func getTimeAPI() (time.Time, error) {
	timeRes, err := http.Get("https://timeapi.io/api/Time/current/zone?timeZone=UTC")
	if err != nil {
		return time.Time{}, errors.Wrapf(err, "failed to get date from timeapi.io")
	}
	defer timeRes.Body.Close()

	var utcTime struct {
		DateTime time.Time `json:"dateTime"`
	}
	if err := json.NewDecoder(timeRes.Body).Decode(&utcTime); err != nil {
		return time.Time{}, errors.Wrapf(err, "failed to decode date response from timeapi.io")
	}

	return utcTime.DateTime, nil
}
