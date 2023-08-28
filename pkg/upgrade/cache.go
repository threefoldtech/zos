package upgrade

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"os"
	"path"
	"syscall"

	"github.com/golang/snappy"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/0-fs/meta"
	"github.com/threefoldtech/0-fs/storage"
	"github.com/xxtea/xxtea-go/xxtea"
	"golang.org/x/crypto/blake2b"

	"golang.org/x/sync/errgroup"
)

const (
	// DefaultDownloadWorkers define the default number of workload to use to downloads data blocks
	DefaultDownloadWorkers = 4
	//DefaultBlockSize is the default block size
	DefaultBlockSize = 512 //KB
)

// Downloader allows to get some data blocks using a pool of workers
type Downloader struct {
	cache     string
	workers   int
	storage   storage.Storage
	blocks    []meta.BlockInfo
	blockSize uint64
}

// DownloaderOption interface
type DownloaderOption interface {
	apply(d *Downloader)
}

type WorkersOpt struct {
	workers uint
}

func (o WorkersOpt) apply(d *Downloader) {
	d.workers = int(o.workers)
}

// NewDownloader creates a downloader for this meta from this storage
func NewDownloader(cache string, storage storage.Storage, m meta.Meta, opts ...DownloaderOption) *Downloader {
	downloader := &Downloader{
		cache:     cache,
		storage:   storage,
		blockSize: m.Info().FileBlockSize,
		blocks:    m.Blocks(),
	}

	for _, opt := range opts {
		opt.apply(downloader)
	}

	return downloader
}

// OutputBlock is the result of a Dowloader worker
type OutputBlock struct {
	Raw   []byte
	Index int
}

// getBlock tries to get block from cache, if not there, downloads it from storage
func (d *Downloader) getBlock(block meta.BlockInfo) ([]byte, error) {
	// check if block is cached already
	hash := make([]byte, 32)
	n := hex.Encode(hash, block.Decipher)
	hash = hash[:n]
	if len(hash) < 4 {
		return nil, fmt.Errorf("invalid chunk hash")
	}

	log.Debug().Msgf("checking cache for block %s", string(hash))
	blockDirPath := path.Join(d.cache, "blocks", string(hash[0:2]), string(hash[2:4]))
	if err := os.MkdirAll(blockDirPath, os.ModePerm); err != nil {
		return nil, errors.Wrapf(err, "failed to create directories for %s", blockDirPath)
	}

	blockFilePath := path.Join(blockDirPath, string(hash))
	blockFile, err := os.OpenFile(blockFilePath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open file %s", blockFilePath)
	}

	if err := syscall.Flock(int(blockFile.Fd()), syscall.LOCK_EX); err != nil {
		return nil, err
	}

	defer func() {
		if err := syscall.Flock(int(blockFile.Fd()), syscall.LOCK_UN); err != nil {
			log.Err(err).Msgf("failed to release file %s", blockFilePath)
		}
	}()

	info, err := blockFile.Stat()
	if err != nil {
		return nil, err
	}

	if info.Size() > 0 {
		// already cached, read from file
		log.Debug().Msgf("block cache hit: %s", string(hash))
		data := make([]byte, DefaultBlockSize*1024)
		n, err := blockFile.Read(data)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read from cache file %s", blockDirPath)
		}

		return data[:n], nil
	}

	data, err := d.downloadBlock(block)
	if err != nil {
		return nil, errors.Wrap(err, "failed to download block")
	}

	// write block data to cache file
	_, err = blockFile.Write(data)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to write block data to cache file %s", blockDirPath)
	}

	return data, nil
}

// downloadBlock downloads a data block identified by block
func (d *Downloader) downloadBlock(block meta.BlockInfo) ([]byte, error) {
	log.Debug().Msgf("downloading block %x", block.Key)
	body, err := d.storage.Get(block.Key)
	if err != nil {
		return nil, err
	}

	defer body.Close()

	data, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}

	data, err = snappy.Decode(nil, xxtea.Decrypt(data, block.Decipher))
	if err != nil {
		return nil, err
	}

	hasher, err := blake2b.New(16, nil)
	if err != nil {
		return nil, err
	}

	if _, err := hasher.Write(data); err != nil {
		return nil, err
	}

	hash := hasher.Sum(nil)
	if !bytes.Equal(hash, block.Decipher) {
		return nil, fmt.Errorf("block key(%x), cypher(%x) hash is wrong hash(%x)", block.Key, block.Decipher, hash)
	}

	return data, nil
}

func (d *Downloader) worker(ctx context.Context, feed <-chan int, out chan<- *OutputBlock) error {
	for index := range feed {
		info := d.blocks[index]
		raw, err := d.getBlock(info)
		if err != nil {
			log.Err(err).Msgf("error downloading block %d", index+1)
			return err
		}

		result := &OutputBlock{
			Index: index,
			Raw:   raw,
		}

		select {
		case out <- result:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

// Download download the file into this output file
func (d *Downloader) Download(output *os.File) error {
	if len(d.blocks) == 0 {
		return fmt.Errorf("no blocks provided")
	}

	if d.blockSize == 0 {
		return fmt.Errorf("block size is not set")
	}

	workers := int(math.Min(float64(d.workers), float64(len(d.blocks))))
	if workers == 0 {
		workers = int(math.Min(float64(DefaultDownloadWorkers), float64(len(d.blocks))))
	}
	group, ctx := errgroup.WithContext(context.Background())

	feed := make(chan int)
	results := make(chan *OutputBlock)

	//start workers.
	for i := 1; i <= workers; i++ {
		group.Go(func() error {
			return d.worker(ctx, feed, results)
		})
	}

	log.Debug().Msgf("downloading %d blocks", len(d.blocks))

	//feed the workers
	group.Go(func() error {
		defer close(feed)
		for index := range d.blocks {
			select {
			case feed <- index:
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		return nil
	})

	go func() {
		if err := group.Wait(); err != nil {
			log.Err(err).Msg("error while waiting for upgrader download workeres")
		}

		close(results)
	}()

	count := 1
	for result := range results {
		log.Debug().Msgf("writing block %d/%d of %s", count, len(d.blocks), output.Name())
		if _, err := output.Seek(int64(result.Index)*int64(d.blockSize), io.SeekStart); err != nil {
			return err
		}

		if _, err := output.Write(result.Raw); err != nil {
			return err
		}

		count++
	}

	return group.Wait()
}
