package database

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func DBService() *mongo.Database {
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")

	// 连接到 MongoDB（通过 WSL）
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		panic(err)
	}

	// 检查连接
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		panic(err)
	}

	database := client.Database("sensative_db")
	return database

	//collection := database.Collection("test") // 替换为你的集合名称

}
