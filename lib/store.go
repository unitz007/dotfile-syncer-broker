package lib

import (
	"context"
	"fmt"
	"github.com/r3labs/sse/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"os"
)

type MachinesStore struct {
	store  *mongo.Collection
	Server *sse.Server
}

type Machines struct {
	ID       string   `bson:"_id"`
	Machines []string `bson:"machines"`
}

func NewStore() MachinesStore {
	defer fmt.Printf("successfully connected to MongoDB!\n")

	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	url := os.Getenv("D_MONGO_URL")
	if url == "" {
		fmt.Println("D_MONGO_URL environment variable not set")
		os.Exit(1)
	}

	//opts := options.Client().ApplyURI("mongodb+srv://unitz007:dFZmjO4G9EgVA6Dt@dotfile-syncer.6s2to.mongodb.net/?retryWrites=true&w=majority&appName=dotfile-syncer").SetServerAPIOptions(serverAPI)
	opts := options.Client().ApplyURI(url).SetServerAPIOptions(serverAPI)
	// Create a new client and connect to the server
	client, err := mongo.Connect(context.TODO(), opts)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	Store := client.Database("dotfile-syncer")

	if err := Store.RunCommand(context.TODO(), bson.D{{"ping", 1}}).Err(); err != nil {
		fmt.Println("could not connect to mongodb", err)
		os.Exit(1)
	}

	return MachinesStore{
		store: Store.Collection("machines"),
	}
}

func (m *MachinesStore) Add(n string) {

	filer := bson.M{"_id": "1"}
	exists := m.store.FindOne(context.Background(), filer)
	if exists.Err() != nil {
		s := Machines{
			Machines: []string{n},
			ID:       "1",
		}
		m.store.InsertOne(context.Background(), s)
	} else {
		var update Machines
		exists.Decode(&update)

		isExists := func() bool {
			e := false
			for _, m := range update.Machines {
				if m == n {
					e = true
				}
			}
			return e
		}()

		if !isExists {
			update.Machines = append(update.Machines, n)
			m.store.FindOneAndReplace(context.TODO(), filer, update)
		}
	}
}

func (m *MachinesStore) Get() []string {
	var response Machines
	err := m.store.FindOne(context.Background(), bson.M{"_id": "1"}).Decode(&response)
	if err != nil {
		fmt.Println(err)
		return []string{}
	}
	return response.Machines
}
