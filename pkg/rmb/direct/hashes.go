package direct

import (
	"crypto/md5"
	"fmt"
	"io"

	"github.com/threefoldtech/zos/pkg/rmb/direct/types"
)

func Challenge(env *types.Envelope) ([]byte, error) {
	hash := md5.New()
	if err := challenge(hash, env); err != nil {
		return nil, err
	}

	return hash.Sum(nil), nil
}

func challenge(w io.Writer, env *types.Envelope) error {
	if _, err := fmt.Fprintf(w, "%s", env.Uid); err != nil {
		return err
	}
	if env.Tags != nil {
		if _, err := fmt.Fprintf(w, "%s", *env.Tags); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(w, "%d", env.Timestamp); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%d", env.Expiration); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%d", env.Source); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "%d", env.Destination); err != nil {
		return err
	}

	if request := env.GetRequest(); request != nil {
		return challengeRequest(w, request)
	} else if response := env.GetResponse(); response != nil {
		return challengeResponse(w, response)
	}

	return nil
}

func challengeRequest(w io.Writer, request *types.Request) error {
	if _, err := fmt.Fprintf(w, "%s", request.Command); err != nil {
		return err
	}

	n, err := w.Write(request.Data)
	if err != nil {
		return err
	}
	if len(request.Data) != n {
		return fmt.Errorf("partial write")
	}
	return nil
}

func challengeResponse(w io.Writer, response *types.Response) error {
	if errResp := response.GetError(); errResp != nil {
		if _, err := fmt.Fprintf(w, "%d", errResp.Code); err != nil {
			return err
		}

		if _, err := fmt.Fprintf(w, "%s", errResp.Message); err != nil {
			return err
		}

	} else if reply := response.GetReply(); reply != nil {
		n, err := w.Write(reply.Data)
		if err != nil {
			return err
		}
		if len(reply.Data) != n {
			return fmt.Errorf("partial write")
		}
	}

	return nil
}
