package digest

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var indexPrefix = "mitum_digest_"

var accountIndexModels = []mongo.IndexModel{
	{
		Keys: bson.D{bson.E{Key: "address", Value: 1}, bson.E{Key: "height", Value: -1}},
		Options: options.Index().
			SetName("mitum_digest_account"),
	},
	{
		Keys: bson.D{bson.E{Key: "height", Value: -1}},
		Options: options.Index().
			SetName("mitum_digest_account_height"),
	},
}

var balanceIndexModels = []mongo.IndexModel{
	{
		Keys: bson.D{bson.E{Key: "address", Value: 1}, bson.E{Key: "height", Value: -1}},
		Options: options.Index().
			SetName("mitum_digest_balance"),
	},
	{
		Keys: bson.D{bson.E{Key: "height", Value: -1}},
		Options: options.Index().
			SetName("mitum_digest_balance_height"),
	},
}

var operationIndexModels = []mongo.IndexModel{
	{
		Keys: bson.D{bson.E{Key: "addresses", Value: 1}, bson.E{Key: "height", Value: 1}, bson.E{Key: "fact", Value: 1}},
		Options: options.Index().
			SetName("mitum_digest_operation"),
	},
	{
		Keys: bson.D{bson.E{Key: "height", Value: -1}},
		Options: options.Index().
			SetName("mitum_digest_operation_height"),
	},
}

var defaultIndexes = map[string] /* collection */ []mongo.IndexModel{
	defaultColNameAccount:   accountIndexModels,
	defaultColNameBalance:   balanceIndexModels,
	defaultColNameOperation: operationIndexModels,
}
