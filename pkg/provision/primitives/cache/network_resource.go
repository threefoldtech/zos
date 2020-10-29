package cache

import (
	"encoding/json"
	"fmt"

	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/provision/primitives"
)

// UpgradeNetworkResource walks over all the reservation in cache and remove any network resource
// that has the network name equal to networkID
// it returns true if a matchine network was fund, false otherwise
func (s *Fs) UpgradeNetworkResource(newRes provision.Reservation) error {
	if newRes.Type != primitives.NetworkResourceReservation && newRes.Type != primitives.NetworkReservation {
		return nil
	}

	newNR := pkg.NetResource{}
	if err := json.Unmarshal(newRes.Data, &newNR); err != nil {
		return fmt.Errorf("failed to unmarshal network from reservation: %w", err)
	}

	cb := func(r provision.Reservation) error {
		if newRes.ID == r.ID || r.Type != primitives.NetworkResourceReservation && r.Type != primitives.NetworkReservation {
			return nil
		}

		nr := pkg.NetResource{}
		if err := json.Unmarshal(r.Data, &nr); err != nil {
			return fmt.Errorf("failed to unmarshal network from reservation: %w", err)
		}

		if provision.NetworkID(r.User, nr.Name) != provision.NetworkID(newRes.User, newNR.Name) {
			return nil
		}

		// at this point we have identified a network resource from the same network in cache
		// we need to remove it
		return s.remove(r.ID)
	}

	return s.walk(cb)
}
