package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mqtt_db/controller"
	"mqtt_db/database"
	"mqtt_db/models"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	_ "mqtt_db/docs"

	"github.com/gin-gonic/gin"
	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"
	"github.com/mochi-mqtt/server/v2/packets"

	swaggerFiles "github.com/swaggo/files"     // swagger embed files
	ginSwagger "github.com/swaggo/gin-swagger" // gin-swagger middleware
	"go.mongodb.org/mongo-driver/mongo"
)

type Message struct {
	Sender   string `json:"sender" bson:"sender"`
	Receiver string `json:"receiver" bson:"receiver" binding:"required"`
	Content  string `json:"message" bson:"message" binding:"required"`
}

// @title          	使用 MQTTX 聊天，並過濾敏感字
// @version         1.0
// @description     使用 MQTT(websocket) + MongoDB 基本操作
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /

// @securityDefinitions.basic  BasicAuth

// @externalDocs.description  OpenAPI
// @externalDocs.url          https://swagger.io/resources/open-api/
func apiServer(database *mongo.Database) {
	r := gin.Default()
	c := controller.Controller{Textobj: &models.TextObj{},
		Database: database}

	accounts := r.Group("/Chat")
	{
		accounts.POST("/AddSensativeWord/", c.AddSensativeWord)
		accounts.POST("/DeleteSensativeWord/", c.DeleteSensativeWord)
		accounts.GET("/Analytics/", c.AnalyticsEveryone)
		accounts.GET("/", c.Chat)
	}

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.Run(":8080")
}

func mqttServer(database *mongo.Database) {
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		done <- true
	}()

	topic := "chat"
	server := mqtt.New(&mqtt.Options{
		InlineClient: true, // you must enable inline client to use direct publishing and subscribing.
	})
	_ = server.AddHook(new(auth.AllowHook), nil)

	ws := listeners.NewWebsocket("ws1", "127.0.0.1:1882", nil)
	err := server.AddListener(ws)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		err := server.Serve()
		if err != nil {
			log.Fatal(err)
		}

		callbackFn := func(cl *mqtt.Client, sub packets.Subscription, pk packets.Packet) {
			//server.Log.Info("inline client received message from subscription", "client", cl.ID, "subscriptionId", sub.Identifier, "topic", pk.TopicName, "payload", string(pk.Payload))
			server.Log.Info("inline client received message from subscription", "client", pk.Origin, "subscriptionId", sub.Identifier, "topic", pk.TopicName, "payload", string(pk.Payload))
			var message Message
			if err := json.Unmarshal(pk.Payload, &message); err != nil {
				log.Printf("Error decoding JSON message: %v", err)
				return
			}

			message.Sender = pk.Origin
			collection := database.Collection("message")
			_, err := collection.InsertOne(context.TODO(), message)
			if err != nil {
				log.Printf("Error decoding JSON message: %v", err)
				return
			}
			fmt.Println("Document inserted into collection.")

			//call chat api
			apiUrl, _ := url.Parse("http://127.0.0.1:8080/Chat/")
			parmas := url.Values{}
			parmas.Add("message", message.Content)
			apiUrl.RawQuery = parmas.Encode()

			response, err := http.Get(apiUrl.String())
			if err != nil {
				log.Fatal(err)
				return
			}

			var editText Message
			body, err := io.ReadAll(response.Body)
			if err := json.Unmarshal(body, &editText); err != nil {
				log.Printf("Error decoding JSON message: %v", err)
				return
			}
			message.Sender = pk.Origin
			info := "sender: " + message.Sender + "\n" + "message: " + editText.Content
			err = server.Publish(topic+"/"+message.Receiver, []byte(info), false, 0)
			if err != nil {
				log.Fatal(err)
			}
		}
		topic := topic
		server.Subscribe(topic, 1, callbackFn)

	}()

	<-done
	server.Log.Warn("caught signal, stopping...")
	_ = server.Close()
	server.Log.Info("main.go finished")
}

func main() {
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		done <- true
	}()
	db := database.DBService()
	go apiServer(db)
	go mqttServer(db)
	<-done
}
