package main

import (
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/provision/primitives/cache"
)

// UpdateReservationsResults is a backward compatability fix to make sure
// all cached reservations has 'results' sets
func UpdateReservationsResults(cache *cache.Fs) error {
	log.Info().Msg("updating reservation results")
	reservations, err := cache.List()
	if err != nil {
		return err
	}

	client, err := app.ExplorerClient()
	if err != nil {
		return err
	}

	for _, reservation := range reservations {
		if !reservation.Result.IsNil() {
			continue
		}

		log.Info().Msgf("updating reservation result for %s", reservation.ID)

		result, err := client.Workloads.NodeWorkloadGet(reservation.ID)
		if err != nil {
			log.Error().Err(err).Msgf("error occurred while requesting reservation result for %s", reservation.ID)
			continue
		}

		provisionResult := result.GetResult()
		reservation.Result = provision.Result{
			Type:      reservation.Type,
			Created:   provisionResult.Epoch.Time,
			State:     provision.ResultState(provisionResult.State),
			Data:      provisionResult.DataJson,
			Error:     provisionResult.Message,
			ID:        provisionResult.WorkloadId,
			Signature: provisionResult.Signature,
		}

		err = cache.Add(reservation, true)
		if err != nil {
			log.Error().Err(err).Msg("error while updating reservation in cache")
			continue
		}
	}

	return nil
}
