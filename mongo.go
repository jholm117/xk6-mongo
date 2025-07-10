package xk6_mongo

import (
	"context"
	"log"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	k6modules "go.k6.io/k6/js/modules"
)

// Register the extension on module initialization, available to
// import from JS as "k6/x/mongo".
func init() {
	k6modules.Register("k6/x/mongo", new(Mongo))
}

// Mongo is the k6 extension for a Mongo client.
type Mongo struct{}

// Client is the Mongo client wrapper.
type Client struct {
	client *mongo.Client
}

type UpsertOneModel struct {
	Query  interface{} `json:"query"`
	Update interface{} `json:"update"`
}

// NewClient represents the Client constructor (i.e. `new mongo.Client()`) and
// returns a new Mongo client object.
// connURI -> mongodb://username:password@address:port/db?connect=direct
func (m *Mongo) NewClient(connURI string) *Client {
	return m.NewClientWithOptions(connURI, options.Client())
}

func (*Mongo) NewClientWithOptions(connURI string, clientOptions *options.ClientOptions) *Client {
	log.Print("start creating new client")

	clientOptions.ApplyURI(connURI)

	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Printf("Error while establishing a connection to MongoDB: %v", err)
		return nil
	}

	log.Print("created new client")
	return &Client{client: client}
}

func (c *Client) Insert(database string, collection string, doc interface{}) error {
	db := c.client.Database(database)
	col := db.Collection(collection)
	_, err := col.InsertOne(context.Background(), doc)
	if err != nil {
		log.Printf("Error while inserting document: %v", err)
		return err
	}
	log.Print("Document inserted successfully")
	return nil
}

func (c *Client) InsertMany(database string, collection string, docs []interface{}) error {
	db := c.client.Database(database)
	col := db.Collection(collection)
	_, err := col.InsertMany(context.Background(), docs)
	if err != nil {
		log.Printf("Error while inserting multiple documents: %v", err)
		return err
	}
	return nil
}

func (c *Client) Upsert(database string, collection string, filter interface{}, upsert interface{}) error {
	db := c.client.Database(database)
	col := db.Collection(collection)
	opts := options.Update().SetUpsert(true)
	_, err := col.UpdateOne(context.Background(), filter, upsert, opts)
	if err != nil {
		log.Printf("Error while performing upsert: %v", err)
		return err
	}
	return nil
}

func (c *Client) Find(database string, collection string, filter string, sort interface{}, limit int64, canonical bool, discardResponseBodies bool) ([]bson.M, error) {
	db := c.client.Database(database)
	col := db.Collection(collection)
	opts := options.Find().SetSort(sort).SetLimit(limit)

	var bsonFilter bson.D
	if filter != "" {
		trimmedFilter := strings.TrimSpace(filter)

		bsonErr := bson.UnmarshalExtJSON([]byte(trimmedFilter), canonical, &bsonFilter)
		if bsonErr != nil {
			log.Printf("Error while unmarshalling filter: %v", bsonErr)
			return nil, bsonErr
		}

	} else {
		bsonFilter = bson.D{}
	}

	cur, err := col.Find(context.Background(), bsonFilter, opts)
	if err != nil {
		log.Printf("Error while finding documents: %v", err)
		return nil, err
	}

	defer cur.Close(context.Background())

	if discardResponseBodies {
		// If discardResponseBodies is true, we still need to iterate through the
		// cursor to ensure the network data is consumed and the cursor is properly
		// closed. Without this, the server might hold resources, or subsequent
		// operations might misbehave if the client doesn't fully drain the cursor.
		// However, we just discard each result without decoding.
		log.Print("discarding response bodies")
		for cur.Next(context.Background()) {
			// Do nothing with cur.Current or cur.Decode, effectively discarding
			// the document data without decoding it into 'results'.
		}
		if err := cur.Err(); err != nil {
			log.Printf("Error during cursor iteration while discarding body: %v", err)
			return nil, err
		}

		return nil, nil
	}

	var results []bson.M
	if err = cur.All(context.Background(), &results); err != nil {
		log.Printf("Error while decoding documents: %v", err)
		return nil, err
	}
	return results, nil
}

func (c *Client) Aggregate(database string, collection string, pipeline interface{}) ([]bson.M, error) {
	db := c.client.Database(database)
	col := db.Collection(collection)
	cur, err := col.Aggregate(context.Background(), pipeline)
	if err != nil {
		log.Printf("Error while aggregating: %v", err)
		return nil, err
	}
	var results []bson.M
	if err = cur.All(context.Background(), &results); err != nil {
		log.Printf("Error while decoding documents: %v", err)
		return nil, err
	}
	return results, nil
}

func (c *Client) FindOne(database string, collection string, filter map[string]string) (bson.M, error) {
	db := c.client.Database(database)
	col := db.Collection(collection)
	var result bson.M
	err := col.FindOne(context.Background(), filter).Decode(&result)
	if err != nil {
		log.Printf("Error while finding the document: %v", err)
		return nil, err
	}

	return result, nil
}

func (c *Client) UpdateOne(database string, collection string, filter interface{}, data bson.D) error {
	db := c.client.Database(database)
	col := db.Collection(collection)

	_, err := col.UpdateOne(context.Background(), filter, data)
	if err != nil {
		log.Printf("Error while updating the document: %v", err)
		return err
	}

	return nil
}

func (c *Client) UpdateMany(database string, collection string, filter interface{}, data bson.D) error {
	db := c.client.Database(database)
	col := db.Collection(collection)

	update := bson.D{{"$set", data}}

	_, err := col.UpdateMany(context.Background(), filter, update)
	if err != nil {
		log.Printf("Error while updating the documents: %v", err)
		return err
	}

	return nil
}

func (c *Client) FindAll(database string, collection string) ([]bson.M, error) {
	db := c.client.Database(database)
	col := db.Collection(collection)
	cur, err := col.Find(context.Background(), bson.D{{}})
	if err != nil {
		log.Printf("Error while finding documents: %v", err)
		return nil, err
	}

	var results []bson.M
	if err = cur.All(context.Background(), &results); err != nil {
		log.Printf("Error while decoding documents: %v", err)
		return nil, err
	}

	return results, nil
}

func (c *Client) DeleteOne(database string, collection string, filter map[string]string) error {
	db := c.client.Database(database)
	col := db.Collection(collection)
	_, err := col.DeleteOne(context.Background(), filter)
	if err != nil {
		log.Printf("Error while deleting the document: %v", err)
		return err
	}

	return nil
}

func (c *Client) DeleteMany(database string, collection string, filter map[string]string) error {
	db := c.client.Database(database)
	col := db.Collection(collection)
	_, err := col.DeleteMany(context.Background(), filter)
	if err != nil {
		log.Printf("Error while deleting the documents: %v", err)
		return err
	}

	return nil
}

func (c *Client) Distinct(database string, collection string, field string, filter interface{}) ([]interface{}, error) {
	db := c.client.Database(database)
	col := db.Collection(collection)
	result, err := col.Distinct(context.Background(), field, filter)
	if err != nil {
		log.Printf("Error while getting distinct values: %v", err)
		return nil, err
	}

	return result, nil
}

func (c *Client) DropCollection(database string, collection string) error {
	db := c.client.Database(database)
	col := db.Collection(collection)
	err := col.Drop(context.Background())
	if err != nil {
		log.Printf("Error while dropping the collection: %v", err)
		return err
	}

	return nil
}

func (c *Client) CountDocuments(database string, collection string, filter interface{}) (int64, error) {
	db := c.client.Database(database)
	col := db.Collection(collection)
	count, err := col.CountDocuments(context.Background(), filter)
	if err != nil {
		log.Printf("Error while counting documents: %v", err)
		return 0, err
	}
	return count, nil
}

func (c *Client) FindOneAndUpdate(database string, collection string, filter interface{}, update interface{}) (*mongo.SingleResult, error) {
	db := c.client.Database(database)
	col := db.Collection(collection)
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	result := col.FindOneAndUpdate(context.Background(), filter, update, opts)
	if result.Err() != nil {
		log.Printf("Error while finding and updating document: %v", result.Err())
		return nil, result.Err()
	}
	return result, nil
}

func (c *Client) Disconnect() error {
	err := c.client.Disconnect(context.Background())
	if err != nil {
		log.Printf("Error while disconnecting from the database: %v", err)
		return err
	}

	return nil
}
