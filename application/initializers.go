package application

import (
	"fmt"
	"github.com/getsentry/raven-go"
	"github.com/thoas/gostorages"
	"github.com/thoas/picfit/engines"
	"github.com/thoas/picfit/util"
)

type Initializer func(app *Application) error

var Initializers = []Initializer{
	KVStoreInitializer,
	StorageInitializer,
	ShardInitializer,
	BasicInitializer,
	SentryInitializer,
}

var KVStores = map[string]KVStoreParameter{
	"redis": RedisKVStoreParameter,
	"cache": CacheKVStoreParameter,
}

var Storages = map[string]StorageParameter{
	"http+s3": HTTPS3StorageParameter,
	"s3":      S3StorageParameter,
	"http+fs": HTTPFileSystemStorageParameter,
	"fs":      FileSystemStorageParameter,
}

var SentryInitializer Initializer = func(app *Application) error {
	results := app.Config.GetStringMapString("sentry.tags")

	var tags map[string]string

	tags = util.MapInterfaceToMapString(results)

	client, err := raven.NewClient(dsn, tags)

	if err != nil {
		return err
	}

	app.Raven = client

	return nil
}

var BasicInitializer Initializer = func(app *Application) error {

	format := app.Config.GetString("options.format")
	quality := app.Config.GetInt("options.quality")

	if quality == 0 {
		quality = DefaultQuality
	}

	app.SecretKey = app.Config.GetString("secret_key")
	app.Engine = &engines.GoImageEngine{
		DefaultFormat:  DefaultFormat,
		Format:         format,
		DefaultQuality: quality,
	}

	enableUpload := app.Config.GetBool("options.enable_upload")
	enableDelete := app.Config.GetBool("options.enable_delete")

	return nil
}

var ShardInitializer Initializer = func(app *Application) error {
	width := app.Config.GetInt("shard.width")

	if width == 0 {
		width = DefaultShardWidth
	}

	depth := app.Config.GetInt("shard.depth")

	if depth == 0 {
		depth = DefaultShardDepth
	}

	app.Shard = Shard{Width: width, Depth: depth}

	return nil
}

var KVStoreInitializer Initializer = func(app *Application) error {
	key := app.Config.GetString("kvstore.type")

	parameter, ok := KVStores[key]

	if !ok {
		return fmt.Errorf("KVStore %s does not exist", key)
	}

	config := app.Config.GetStringMapString("kvstore")

	params := util.MapInterfaceToMapString(config)
	store, err := parameter(params)

	if err != nil {
		return err
	}

	app.Prefix = params["prefix"]
	app.KVStore = store

	return nil
}

func getStorageFromConfig(key string, app *Application) (gostorages.Storage, error) {
	storageType := app.Config.GetString("storage." + key + ".type")

	parameter, ok := Storages[storageType]

	if !ok {
		return nil, fmt.Errorf("Storage %s does not exist", key)
	}

	config, err := app.GetStringMapString("storage." + key)

	if err != nil {
		return nil, err
	}

	storage, err := parameter(util.MapInterfaceToMapString(config))

	if err != nil {
		return nil, err
	}

	return storage, err
}

var StorageInitializer Initializer = func(app *Application) error {
	sourceStorage := getStorageFromConfig("src", app)

	app.SourceStorage = sourceStorage

	destStorage, err := getStorageFromConfig("dst", app)

	if err != nil {
		app.DestStorage = sourceStorage
	} else {
		app.DestStorage = destStorage
	}

	return nil
}
