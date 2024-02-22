package controller

import (
	"context"
	"fmt"
	"log"
	"mqtt_db/models"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type Controller struct {
	Textobj  *models.TextObj
	Database *mongo.Database
}

type HttpObj struct {
	Success string `json:"success"`
	Error   string `json:"error`
	Msg     string `json: "message"`
}

// AddSensativeWord godoc
// @Summary    	 新增敏感詞
// @Description  新增敏感詞，且用 詞+作用域 來表達完整性 (primary key) \n `apply_to` 目前只對 all 有反應，因此使用 all + 敏感詞即可。 其中 policy 提供兩個方法，replace 則必須在提供 replacement 來供應需要替換的敏感詞字串；obfuscate 則程式自動給出 ***
// @Tags         聊天
// @Accept       json
// @Produce      json
// @Param        sensative_word  body models.SensativeWords true "敏感詞的 object"
// @Success      200  {object}  models.SensativeWords
// @Failure      400  {object}  HttpObj
// @Router       /Chat/AddSensativeWord/ [post]
func (c *Controller) AddSensativeWord(ctx *gin.Context) {
	var json models.SensativeWords
	if err := ctx.BindJSON(&json); err != nil { //cannot identify
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	validate.RegisterValidation("ValidApply", models.ValidApply)
	validate.RegisterValidation("ValidPolicy", models.ValidPolicy)
	validate.RegisterValidation("ValidReplacement", models.ValidReplacement)
	validate.RegisterValidation("ValidGender", models.ValidGender)
	err := validate.Struct(json)
	if err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			ctx.JSON(400, gin.H{"error": "The request body is illegal.",
				"Field": err.Field(), "Tag": err.Tag()})
			return
		}
	}

	t := c.Textobj
	// if flag := t.IsPolicyValid(json); flag == false {
	// 	ctx.JSON(400, gin.H{"error": "This policy logical is not legal!"})
	// 	return
	// }
	if flag := t.IsGenderValid(json); flag == false {
		ctx.JSON(400, gin.H{"error": "This gender logical is not legal!"})
		return
	}

	collection := c.Database.Collection("sensative_words")
	result := collection.FindOne(context.TODO(), bson.M{"sensative_words": json.Text, "apply_to": json.Apply})
	if result.Err() != mongo.ErrNoDocuments {
		ctx.JSON(400, gin.H{"error": "This sensative word is exists!"})
		return
	}

	_, err = collection.InsertOne(context.TODO(), json)
	if err != nil {
		ctx.JSON(400, gin.H{"error": err})
		return
	}
	fmt.Println("Document inserted into collection.")

	//t.AddSensativeWord(json)
	ctx.JSON(200, gin.H{"success": "Add done!"})
}

// DeleteSensativeWord godoc
// @Summary    	 刪除敏感詞
// @Description  刪除敏感詞，且用 詞+作用域 來表達唯一性 (primary key)
// @Tags         聊天
// @Accept       json
// @Produce      json
// @Param        sensative_word  body models.SensativeWords true "敏感詞的 object"
// @Success      200  {object}  models.SensativeWords
// @Failure      400  {object}  HttpObj
// @Router       /Chat/DeleteSensativeWord/ [post]
func (c *Controller) DeleteSensativeWord(ctx *gin.Context) {
	var json models.SensativeWords
	if err := ctx.BindJSON(&json); err != nil { //cannot identify
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	//t := c.Textobj
	collection := c.Database.Collection("sensative_words")
	deleteFilter := bson.M{"sensative_words": json.Text, "apply_to": json.Apply}
	deleteResult, err := collection.DeleteMany(context.TODO(), deleteFilter)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Deleted %v document(s)\n", deleteResult.DeletedCount)
	if deleteResult.DeletedCount == 0 {
		ctx.JSON(400, gin.H{"error": "Database doesn't find this sensative words."})
		return
	}
	ctx.JSON(200, gin.H{"success": "Add done!"})
}

type UserAnalyticsObj struct {
	Sender  string `json: "sender"`
	Count   string `json: "count"`
	Message string `json: "Message"`
}

// AnalyticsEveryone godoc
// @Summary    	 輸出每個人的敏感詞使用紀錄
// @Description  輸出每個人的敏感詞使用紀錄
// @Tags         聊天
// @Accept       json
// @Produce      json
// @Success      200  {object}  UserAnalyticsObj
// @Failure      400  {object}  HttpObj
// @Router       /Chat/Analytics/ [get]
func (c *Controller) AnalyticsEveryone(ctx *gin.Context) {

	//search each sensative word
	collection := c.Database.Collection("sensative_words")
	filter := bson.M{"apply_to": "all"}
	cursor, _ := collection.Find(context.TODO(), filter)
	var results []models.SensativeWords
	if err := cursor.All(context.Background(), &results); err != nil {
		log.Fatal(err)
	}
	regex := ""
	for _, result := range results {
		regex += result.Text + "|"
	}
	regex = regex[:len(regex)-1]

	//ctx.JSON(200, gin.H{"success": regex})

	//counting everyword of everyone
	//t := c.Textobj

	collection = c.Database.Collection("message")
	selectSensativeWords := bson.D{
		{"$match", bson.D{{"message", bson.D{{"$regex", regex}}}}},
	}
	fmt.Println(selectSensativeWords)
	countByUser := bson.D{
		{"$group", bson.D{
			{"_id", "$sender"},
			{"sender", bson.D{{"$first", "$sender"}}},
			{"count", bson.D{{"$sum", 1}}},
			{"messages", bson.D{{"$push", "$message"}}},
		}},
	}

	pipeline := mongo.Pipeline{selectSensativeWords, countByUser}
	cursor, err := collection.Aggregate(context.Background(), pipeline)
	if err != nil {
		log.Fatal(err)
	}
	defer cursor.Close(context.Background())

	var test []bson.M
	fmt.Println("Test start")
	fmt.Println("Regex: ", regex)
	if err = cursor.All(context.TODO(), &test); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Aggregation Result:", test)
	// var infos []models.UserSensativeWordsCount
	// if err := cursor.All(context.Background(), &infos); err != nil {
	// 	log.Fatal(err)
	// }

	ctx.JSON(200, gin.H{"success": test})
}

// Chat godoc
// @Summary    	 輸入他的對話，並將敏感詞移除
// @Description  輸入他的對話，並將敏感詞移除
// @Tags         聊天
// @Accept       json
// @Produce      json
// @Param        message query string true "對話文字"
// @Success      200  {object}  HttpObj
// @Failure      400  {object}  HttpObj
// @Router       /Chat/ [get]
func (c *Controller) Chat(ctx *gin.Context) {
	text := ctx.Query("message")
	//fmt.Println(text)
	collection := c.Database.Collection("sensative_words")
	filter := bson.M{"apply_to": "all"}
	cursor, _ := collection.Find(context.TODO(), filter)
	var results []models.SensativeWords
	if err := cursor.All(context.Background(), &results); err != nil {
		log.Fatal(err)
	}

	c.Textobj.Sensative = results
	text = c.Textobj.TestingText(text)
	ctx.JSON(200, gin.H{"success": "completely", "message": text})

}
