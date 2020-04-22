package builders

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/crypto"
	"github.com/threefoldtech/zos/pkg/identity"
	"github.com/threefoldtech/zos/pkg/schema"
	"github.com/threefoldtech/zos/tools/client"
	"github.com/threefoldtech/zos/tools/explorer/models/generated/workloads"
	wrklds "github.com/threefoldtech/zos/tools/explorer/pkg/workloads"
)

var (
	day             = time.Hour * 24
	defaultDuration = day * 30
)

// ReservationBuilder is a struct that can build reservations
type ReservationBuilder struct {
	Reservation workloads.Reservation
	Bcdb        *client.Client
	Mainui      *identity.UserIdentity
	assets      []string
	seedPath    string
	dryRun      bool
}

// NewReservationBuilder creates a new ReservationBuilder
func NewReservationBuilder(bcdb *client.Client, mainui *identity.UserIdentity) *ReservationBuilder {
	reservation := workloads.Reservation{}
	reservation.Epoch = schema.Date{Time: time.Now()}
	return &ReservationBuilder{
		Reservation: reservation,
		Bcdb:        bcdb,
		Mainui:      mainui,
	}
}

// LoadReservationBuilder loads a reservation builder based on a file path
func LoadReservationBuilder(reader io.Reader) (*ReservationBuilder, error) {
	reservation := workloads.Reservation{}
	err := json.NewDecoder(reader).Decode(&reservation)
	if err != nil {
		return &ReservationBuilder{}, err
	}

	return &ReservationBuilder{Reservation: reservation}, nil
}

// Save saves the reservation builder to an IO.Writer
func (r *ReservationBuilder) Save(writer io.Writer) error {
	err := json.NewEncoder(writer).Encode(r.Reservation)
	if err != nil {
		return err
	}
	return err
}

// Build returns the reservation
func (r *ReservationBuilder) Build() workloads.Reservation {
	return r.Reservation
}

// Deploy deploys the reservation
func (r *ReservationBuilder) Deploy() (wrklds.ReservationCreateResponse, error) {
	userID := int64(r.Mainui.ThreebotID)
	signer, err := client.NewSigner(r.Mainui.Key().PrivateKey.Seed())
	if err != nil {
		return wrklds.ReservationCreateResponse{}, errors.Wrapf(err, "could not find seed file at %s", r.seedPath)
	}

	r.Reservation.CustomerTid = userID
	// we always allow user to delete his own reservations
	r.Reservation.DataReservation.SigningRequestDelete.QuorumMin = 1
	r.Reservation.DataReservation.SigningRequestDelete.Signers = []int64{userID}

	// set allowed the currencies as provided by the user
	r.Reservation.DataReservation.Currencies = r.assets

	bytes, err := json.Marshal(r.Reservation.DataReservation)
	if err != nil {
		return wrklds.ReservationCreateResponse{}, err
	}

	r.Reservation.Json = string(bytes)
	_, signature, err := signer.SignHex(r.Reservation.Json)
	if err != nil {
		return wrklds.ReservationCreateResponse{}, errors.Wrap(err, "failed to sign the reservation")
	}

	r.Reservation.CustomerSignature = signature

	if r.dryRun {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return wrklds.ReservationCreateResponse{}, enc.Encode(r.Reservation)
	}

	response, err := r.Bcdb.Workloads.Create(r.Reservation)
	if err != nil {
		return wrklds.ReservationCreateResponse{}, errors.Wrap(err, "failed to send reservation")
	}

	return response, nil
}

// DeleteReservation deletes a reservation by id
func (r *ReservationBuilder) DeleteReservation(resID int64) error {
	userID := int64(r.Mainui.ThreebotID)

	reservation, err := r.Bcdb.Workloads.Get(schema.ID(resID))
	if err != nil {
		return errors.Wrap(err, "failed to get reservation info")
	}

	signer, err := client.NewSigner(r.Mainui.Key().PrivateKey.Seed())
	if err != nil {
		return errors.Wrapf(err, "failed to load signer")
	}

	_, signature, err := signer.SignHex(resID, reservation.Json)
	if err != nil {
		return errors.Wrap(err, "failed to sign the reservation")
	}

	if err := r.Bcdb.Workloads.SignDelete(schema.ID(resID), schema.ID(userID), signature); err != nil {
		return errors.Wrapf(err, "failed to sign deletion of reservation: %d", resID)
	}

	fmt.Printf("Reservation %v marked as to be deleted\n", resID)
	return nil
}

// WithDryRun sets if dry run to the reservation
func (r *ReservationBuilder) WithDryRun(dryRun bool) *ReservationBuilder {
	r.dryRun = dryRun
	return r
}

// WithDuration sets the duration to the reservation
func (r *ReservationBuilder) WithDuration(duration string) (*ReservationBuilder, error) {
	if duration == "" {
		timein := time.Now().Local().Add(defaultDuration)
		r.Reservation.DataReservation.ExpirationReservation = schema.Date{Time: timein}
		return r, nil
	}
	d, err := time.ParseDuration(duration)
	if err != nil {
		nrDays, err := strconv.Atoi(duration)
		if err != nil {
			return r, errors.Wrap(err, "unsupported duration format")
		}
		d = time.Duration(nrDays) * day
	}
	timein := time.Now().Local().Add(d)
	r.Reservation.DataReservation.ExpirationReservation = schema.Date{Time: timein}
	return r, nil
}

// WithAssets sets the assets to the reservation
func (r *ReservationBuilder) WithAssets(assets []string) *ReservationBuilder {
	r.assets = assets
	return r
}

// WithSeedPath sets the seed to the reservation
func (r *ReservationBuilder) WithSeedPath(seedPath string) *ReservationBuilder {
	r.seedPath = seedPath
	return r
}

// AddVolume adds a volume builder to the reservation builder
func (r *ReservationBuilder) AddVolume(volume VolumeBuilder) *ReservationBuilder {
	r.Reservation.DataReservation.Volumes = append(r.Reservation.DataReservation.Volumes, volume.Volume)
	return r
}

// AddNetwork adds a network builder to the reservation builder
func (r *ReservationBuilder) AddNetwork(network NetworkBuilder) *ReservationBuilder {
	r.Reservation.DataReservation.Networks = append(r.Reservation.DataReservation.Networks, network.Network)
	return r
}

// AddZdb adds a zdb builder to the reservation builder
func (r *ReservationBuilder) AddZdb(zdb ZdbBuilder) *ReservationBuilder {
	r.Reservation.DataReservation.Zdbs = append(r.Reservation.DataReservation.Zdbs, zdb.ZDB)
	return r
}

// AddContainer adds a container builder to the reservation builder
func (r *ReservationBuilder) AddContainer(container ContainerBuilder) *ReservationBuilder {
	r.Reservation.DataReservation.Containers = append(r.Reservation.DataReservation.Containers, container.Container)
	return r
}

// AddK8s adds a k8s builder to the reservation builder
func (r *ReservationBuilder) AddK8s(k8s K8sBuilder) *ReservationBuilder {
	r.Reservation.DataReservation.Kubernetes = append(r.Reservation.DataReservation.Kubernetes, k8s.K8S)
	return r
}

func encryptSecret(plain, nodeID string) (string, error) {
	if len(plain) == 0 {
		return "", nil
	}

	pubkey, err := crypto.KeyFromID(pkg.StrIdentifier(nodeID))
	if err != nil {
		return "", err
	}

	encrypted, err := crypto.Encrypt([]byte(plain), pubkey)
	return hex.EncodeToString(encrypted), err
}
