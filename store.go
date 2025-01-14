package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
	"net/url"
	"os"
)

type MachinesStore struct {
	store *mongo.Collection
}

type GitHttpCommitResponse struct {
	Sha    string `json:"sha"`
	Commit Commit `json:"commit"`
}

type Commit struct {
	Id     string `json:"id"`
	Author struct {
		Date string `json:"date"`
	} `json:"author"`
}

type SyncStatus struct {
	//RemoteCommit string `bson:"remote_commit" json:"remote_commit"`
	LocalCommit string `bson:"local_commit" json:"local_commit"`
	IsSync      bool   `json:"is_sync"`
}

type Machine struct {
	Id         string     `bson:"_id" json:"_id"`
	SyncStatus SyncStatus `bson:"sync_details" json:"sync_details"`
}

func NewStore() MachinesStore {
	defer fmt.Printf("successfully connected to Database!\n")

	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	mongoUrl := os.Getenv("D_MONGO_URL")
	if mongoUrl == "" {
		fmt.Println("D_MONGO_URL environment variable not set")
		os.Exit(1)
	}

	opts := options.Client().ApplyURI(mongoUrl).SetServerAPIOptions(serverAPI)

	// Create a new client and connect to the server
	client, err := mongo.Connect(context.TODO(), opts)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	Store := client.Database("dotfile-syncer")

	if err := Store.RunCommand(context.TODO(), bson.D{{"ping", 1}}).Err(); err != nil {
		fmt.Println("could not connect to Database", err)
		os.Exit(1)
	}

	return MachinesStore{
		store: Store.Collection("machines"),
	}
}

func (m *MachinesStore) Add(machine *Machine) error {
	_, err := m.Get(machine.Id)
	if err != nil {
		_, err = m.store.InsertOne(context.TODO(), machine)
		if err != nil {
			return err
		}
		return nil
	} else {
		_, err = m.store.UpdateOne(context.TODO(), bson.M{"_id": machine.Id}, bson.M{"$set": machine})
		if err != nil {
			fmt.Println(err)
			return err
		}

		return nil
	}
}

func (m *MachinesStore) Get(id string) (*Machine, error) {
	machine := &Machine{}
	err := m.store.FindOne(context.Background(), bson.M{"_id": id}).Decode(machine)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("machine not found")
		}

		return nil, err
	}

	remoteCommit, err := GitRemoteCommit()
	if err == nil {
		machine.SyncStatus.IsSync = machine.SyncStatus.LocalCommit == remoteCommit.Id
	}

	return machine, nil
}

func (m *MachinesStore) GetAll() *[]Machine {
	cursor, err := m.store.Find(context.TODO(), bson.D{})
	if err != nil {
		return nil
	}

	var machines []Machine
	remoteCommit, err := GitRemoteCommit()
	for cursor.Next(context.Background()) {
		var machine Machine
		_ = cursor.Decode(&machine)
		if err == nil {
			machine.SyncStatus.IsSync = machine.SyncStatus.LocalCommit == remoteCommit.Id
		}
		machines = append(machines, machine)
	}

	return &machines
}

func GitRemoteCommit() (*Commit, error) {
	gitUrl, err := url.Parse(os.Getenv("GIT_URL"))
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequest(http.MethodGet, gitUrl.String(), nil)
	if err != nil {
		return nil, err
	}

	gitToken := os.Getenv("GITHUB_TOKEN")
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Authorization", "Bearer "+gitToken)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}

	statusCode := response.StatusCode

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("unable to fetch remote commit: %v", statusCode)
	}

	var responseBody []GitHttpCommitResponse

	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response body: %v", err)
	}

	headCommit := responseBody[0]

	commit := &Commit{
		Id: headCommit.Sha,
	}

	return commit, nil
}
