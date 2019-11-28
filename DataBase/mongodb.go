package DataBase

import (
	"GoShort/Config"
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Record struct {
	Url 	string `json:"url"`
	Mapping string `json:"maps"`
}

// Boilerplate to connect to MongoDB and return a client and collection ready to use
// TODO: Create 1 and reuse ?
func newClient(a *Config.MongoDB) (client *mongo.Client, collection *mongo.Collection, err error) {
	clientOptions := options.Client().ApplyURI(a.URI)
	client, err = mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		return
	}
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		return
	}
	collection = client.Database(a.DataBase).Collection(a.Collection)
	return
}

// Create a new mapping in the DB
func Insert(a *Config.MongoDB, url string, mapping string) (result *mongo.InsertOneResult, err error) {
	client, collection, err := newClient(a)
	if err != nil {
		return
	}
	rb := Record{Url:url, Mapping:mapping}
	result, err = collection.InsertOne(context.TODO(), rb)
	if err != nil {
		return
	}
	err = client.Disconnect(context.TODO())
	if err != nil {
		return
	}
	return
}

// Looks for the URL using a Mapping
func FilterFromMapping(a *Config.MongoDB, mapping string) (result string, err error) {
	client, collection, err := newClient(a)
	if err != nil {
		return
	}
	r := Record{}
	filter := bson.D{{"mapping", mapping}}

	err = collection.FindOne(context.TODO(), filter).Decode(&r)
	if err != nil {
		return
	}
	err = client.Disconnect(context.TODO())
	if err != nil {
		return
	}
	result = r.Url
	return
}

// Looks for the Mapping using a URL
func FilterFromURL(a *Config.MongoDB, url string) (result string, err error) {
	client, collection, err := newClient(a)
	if err != nil {
		return
	}
	r := Record{}
	filter := bson.D{{"url", url}}

	err = collection.FindOne(context.TODO(), filter).Decode(&r)
	if err != nil {
		return
	}
	err = client.Disconnect(context.TODO())
	if err != nil {
		return
	}
	result = r.Mapping
	return
}
