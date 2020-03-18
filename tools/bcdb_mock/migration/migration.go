package main

import (
	"context"
	"flag"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/tools/bcdb_mock/mw"
	"gopkg.in/yaml.v2"

	"go.mongodb.org/mongo-driver/mongo"

	directory "github.com/threefoldtech/zos/tools/bcdb_mock/pkg/directory/types"
)

func foreach(root string, f func(p string, r io.Reader) error) error {
	files, err := ioutil.ReadDir(root)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if filepath.Ext(file.Name()) != ".yaml" {
			log.Debug().Str("file", file.Name()).Msgf("skipping file")
			continue
		}

		p := filepath.Join(root, file.Name())
		fd, err := os.Open(p)
		if err != nil {
			return err
		}

		if err := f(p, fd); err != nil {
			fd.Close()
			return err
		}

		fd.Close()
	}

	return nil
}

type Migrator func(root string, db *mongo.Database) error

func migrateFarms(root string, db *mongo.Database) error {
	col := db.Collection(directory.FarmCollection)
	return foreach(root, func(p string, r io.Reader) error {
		var farm directory.Farm
		if err := yaml.NewDecoder(r).Decode(&farm); err != nil {
			return err
		}

		// if err := farm.Validate(); err != nil {
		// 	return errors.Wrapf(err, "file '%s'", p)
		// }

		_, err := col.InsertOne(context.TODO(), farm)
		if err != nil {
			log.Error().Err(err).Msgf("failed to insert option '%s'", p)
		}

		return nil
	})
}

func migrateNodes(root string, db *mongo.Database) error {
	col := db.Collection(directory.FarmCollection)
	return foreach(root, func(p string, r io.Reader) error {
		var farm directory.Farm
		if err := yaml.NewDecoder(r).Decode(&farm); err != nil {
			return err
		}

		// if err := farm.Validate(); err != nil {
		// 	return errors.Wrapf(err, "file '%s'", p)
		// }

		_, err := col.InsertOne(context.TODO(), farm)
		if err != nil {
			log.Error().Err(err).Msgf("failed to insert option '%s'", p)
		}

		return nil
	})
}

func main() {
	app.Initialize()

	var (
		root   string
		dbConf string
		name   string
	)

	flag.StringVar(&dbConf, "mongo", "mongodb://localhost:27017", "connection string to mongo database")
	flag.StringVar(&name, "name", "explorer", "database name")
	flag.StringVar(&root, "root", "", "root directory of the bcdb exported data")

	flag.Parse()
	if strings.EqualFold(root, "") {
		log.Fatal().Msg("root option is required")
	}

	db, err := mw.NewDatabaseMiddleware(name, dbConf)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}

	if err := directory.Setup(context.TODO(), db.Database()); err != nil {
		log.Fatal().Err(err).Msg("failed to setup directory indexes")
	}

	types := map[string]Migrator{
		"tfgrid_directory/tfgrid.directory.farm.1/yaml": migrateFarms,
	}

	for typ, migrator := range types {
		if err := migrator(filepath.Join(root, typ), db.Database()); err != nil {
			log.Error().Err(err).Str("root", typ).Msg("migration failed")
		}
	}
}
